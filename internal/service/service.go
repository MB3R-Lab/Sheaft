package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/MB3R-Lab/Sheaft/internal/analyzer"
	"github.com/MB3R-Lab/Sheaft/internal/artifact"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/report"
)

type Clock interface {
	Now() time.Time
}

type ArtifactLocator interface {
	Resolve() (string, error)
	WatchPaths() []string
}

type HistoryStore interface {
	Save(report.Report) error
}

type AnalyzerFunc func(artifact.Loaded, config.AnalysisConfig, *report.Report) (analyzer.Result, error)

type Service struct {
	cfg         config.ServeConfig
	analysisCfg config.AnalysisConfig
	clock       Clock
	locator     ArtifactLocator
	store       HistoryStore
	analyze     AnalyzerFunc
	registry    *prometheus.Registry
	metrics     *Metrics

	mu        sync.RWMutex
	ready     bool
	lastError string
	current   *analyzer.Result
	history   []report.Report
}

type Status struct {
	Ready         bool                  `json:"ready"`
	LastError     string                `json:"last_error,omitempty"`
	HistorySize   int                   `json:"history_size"`
	GeneratedAt   string                `json:"generated_at,omitempty"`
	Decision      string                `json:"decision,omitempty"`
	InputArtifact *report.InputArtifact `json:"input_artifact,omitempty"`
	Summary       report.Summary        `json:"summary"`
	Profiles      []StatusProfile       `json:"profiles,omitempty"`
}

type StatusProfile struct {
	Name                    string  `json:"name"`
	Decision                string  `json:"decision"`
	WeightedAggregate       float64 `json:"weighted_aggregate"`
	UnweightedAggregate     float64 `json:"unweighted_aggregate"`
	EndpointsBelowThreshold int     `json:"endpoints_below_threshold"`
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type fileHistoryStore struct {
	dir string
}

func (s fileHistoryStore) Save(rep report.Report) error {
	if strings.TrimSpace(s.dir) == "" {
		return nil
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}
	digest := "unknown"
	if rep.InputArtifact != nil && rep.InputArtifact.Digest != "" {
		digest = strings.TrimPrefix(rep.InputArtifact.Digest, "sha256:")
		if len(digest) > 12 {
			digest = digest[:12]
		}
	}
	name := fmt.Sprintf("%s-%s.json", time.Now().UTC().Format("20060102T150405.000000000Z"), digest)
	return report.WriteJSON(filepath.Join(s.dir, name), rep)
}

func New(cfg config.ServeConfig, analysisCfg config.AnalysisConfig) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate serve config: %w", err)
	}
	analysisCfg = analysisCfg.Normalized()
	if err := analysisCfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate analysis config: %w", err)
	}
	registry := prometheus.NewRegistry()
	return newWithDeps(cfg, analysisCfg, realClock{}, newLocator(cfg.Artifact), fileHistoryStore{dir: cfg.History.DiskDir}, analyzer.AnalyzeLoaded, registry), nil
}

func newWithDeps(cfg config.ServeConfig, analysisCfg config.AnalysisConfig, clock Clock, locator ArtifactLocator, store HistoryStore, analyze AnalyzerFunc, registry *prometheus.Registry) *Service {
	return &Service{
		cfg:         cfg,
		analysisCfg: analysisCfg,
		clock:       clock,
		locator:     locator,
		store:       store,
		analyze:     analyze,
		registry:    registry,
		metrics:     newMetrics(registry),
		history:     make([]report.Report, 0, cfg.History.MaxItems),
	}
}

func (s *Service) Run(ctx context.Context) error {
	if err := s.recomputeIfChanged(true); err != nil {
		s.setError(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/current-report", s.handleCurrentReport)
	mux.HandleFunc("/current-diff", s.handleCurrentDiff)
	mux.HandleFunc("/history", s.handleHistory)
	mux.Handle("/metrics", promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{EnableOpenMetrics: true}))

	server := &http.Server{
		Addr:    s.cfg.Listen,
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go s.watchLoop(ctx)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/current-report", s.handleCurrentReport)
	mux.HandleFunc("/current-diff", s.handleCurrentDiff)
	mux.HandleFunc("/history", s.handleHistory)
	mux.Handle("/metrics", promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	return mux
}

func (s *Service) watchLoop(ctx context.Context) {
	var watcher *fsnotify.Watcher
	var events <-chan fsnotify.Event
	var watchErrors <-chan error
	if s.watchFS() {
		w, err := fsnotify.NewWatcher()
		if err == nil {
			for _, path := range s.locator.WatchPaths() {
				_ = w.Add(path)
			}
			watcher = w
			events = w.Events
			watchErrors = w.Errors
		}
	}
	if watcher != nil {
		defer watcher.Close()
	}

	var ticker *time.Ticker
	if s.watchPolling() {
		pollInterval, err := s.cfg.PollDuration()
		if err != nil {
			s.setError(err)
			return
		}
		ticker = time.NewTicker(pollInterval)
		defer ticker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-tickerC(ticker):
			if err := s.recomputeIfChanged(false); err != nil {
				s.setError(err)
			}
		case <-events:
			if err := s.recomputeIfChanged(false); err != nil {
				s.setError(err)
			}
		case err := <-watchErrors:
			if err != nil {
				s.setError(err)
			}
		}
	}
}

func (s *Service) recomputeIfChanged(force bool) error {
	path, err := s.locator.Resolve()
	if err != nil {
		return err
	}
	loaded, err := artifact.Load(path)
	if err != nil {
		s.metrics.recordOutcome("error", 0)
		return err
	}

	s.mu.RLock()
	current := s.current
	var previous *report.Report
	if current != nil {
		previous = &current.Report
	}
	s.mu.RUnlock()

	if !force && current != nil && current.Report.InputArtifact != nil {
		if current.Report.InputArtifact.Digest == loaded.Metadata.Digest && current.Report.InputArtifact.Path == loaded.Metadata.Path {
			return nil
		}
	}

	started := s.clock.Now()
	result, err := s.analyze(loaded, s.analysisCfg, previous)
	duration := s.clock.Now().Sub(started)
	if err != nil {
		s.metrics.recordOutcome("error", duration)
		return err
	}
	s.metrics.recordOutcome("success", duration)

	modelAge := time.Duration(0)
	if producedAt, err := time.Parse(time.RFC3339Nano, result.Report.InputArtifact.ProducedAt); err == nil {
		modelAge = s.clock.Now().Sub(producedAt)
	}
	s.metrics.updateCurrent(result.Report, modelAge)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = true
	s.lastError = ""
	s.current = &result
	s.history = append(s.history, result.Report)
	if len(s.history) > s.cfg.History.MaxItems {
		s.history = slices.Clone(s.history[len(s.history)-s.cfg.History.MaxItems:])
	}
	if s.store != nil {
		_ = s.store.Save(result.Report)
	}
	return nil
}

func (s *Service) CurrentStatus() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := Status{
		Ready:       s.ready,
		LastError:   s.lastError,
		HistorySize: len(s.history),
	}
	if s.current == nil {
		return status
	}
	status.GeneratedAt = s.current.Report.GeneratedAt
	status.Decision = s.current.Report.PolicyEvaluation.Decision
	status.InputArtifact = s.current.Report.InputArtifact
	status.Summary = s.current.Report.Summary
	status.Profiles = make([]StatusProfile, 0, len(s.current.Report.NormalizedProfiles()))
	for _, profile := range s.current.Report.NormalizedProfiles() {
		status.Profiles = append(status.Profiles, StatusProfile{
			Name:                    profile.Name,
			Decision:                profile.Decision,
			WeightedAggregate:       profile.Simulation.WeightedAggregate,
			UnweightedAggregate:     profile.Simulation.UnweightedAggregate,
			EndpointsBelowThreshold: profile.EndpointsBelowThreshold,
		})
	}
	return status
}

func (s *Service) CurrentReport() *report.Report {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.current == nil {
		return nil
	}
	rep := s.current.Report
	return &rep
}

func (s *Service) History() []report.Report {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Clone(s.history)
}

func (s *Service) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Service) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	status := s.CurrentStatus()
	code := http.StatusOK
	if !status.Ready {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, status)
}

func (s *Service) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.CurrentStatus())
}

func (s *Service) handleCurrentReport(w http.ResponseWriter, _ *http.Request) {
	current := s.CurrentReport()
	if current == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no current report"})
		return
	}
	writeJSON(w, http.StatusOK, current)
}

func (s *Service) handleCurrentDiff(w http.ResponseWriter, _ *http.Request) {
	current := s.CurrentReport()
	if current == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no current report"})
		return
	}
	writeJSON(w, http.StatusOK, current.Diffs)
}

func (s *Service) handleHistory(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.History())
}

func (s *Service) setError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = err.Error()
}

func (s *Service) watchFS() bool {
	return s.cfg.WatchFS == nil || *s.cfg.WatchFS
}

func (s *Service) watchPolling() bool {
	return s.cfg.WatchPolling == nil || *s.cfg.WatchPolling
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func tickerC(ticker *time.Ticker) <-chan time.Time {
	if ticker == nil {
		return nil
	}
	return ticker.C
}

type osLocator struct {
	cfg config.ArtifactSource
}

func newLocator(cfg config.ArtifactSource) ArtifactLocator {
	return osLocator{cfg: cfg}
}

func (l osLocator) Resolve() (string, error) {
	mode := l.cfg.Mode
	if mode == "auto" {
		info, err := os.Stat(l.cfg.Path)
		if err != nil {
			return "", fmt.Errorf("stat artifact path: %w", err)
		}
		if info.IsDir() {
			mode = "directory"
		} else {
			mode = "file"
		}
	}

	switch mode {
	case "file":
		return l.cfg.Path, nil
	case "directory":
		return l.resolveDirectory()
	default:
		return "", fmt.Errorf("unsupported artifact mode %q", l.cfg.Mode)
	}
}

func (l osLocator) WatchPaths() []string {
	mode := l.cfg.Mode
	if mode == "auto" {
		if info, err := os.Stat(l.cfg.Path); err == nil {
			if info.IsDir() {
				mode = "directory"
			} else {
				mode = "file"
			}
		}
	}
	switch mode {
	case "directory":
		return []string{l.cfg.Path}
	case "file":
		return []string{filepath.Dir(l.cfg.Path)}
	default:
		return []string{filepath.Dir(l.cfg.Path)}
	}
}

func (l osLocator) resolveDirectory() (string, error) {
	entries, err := os.ReadDir(l.cfg.Path)
	if err != nil {
		return "", fmt.Errorf("read artifact directory: %w", err)
	}
	type candidate struct {
		path string
		mod  time.Time
	}
	var best candidate
	found := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !matchesAny(name, l.cfg.Patterns) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		candidatePath := filepath.Join(l.cfg.Path, name)
		if !found || info.ModTime().After(best.mod) || (info.ModTime().Equal(best.mod) && candidatePath > best.path) {
			best = candidate{path: candidatePath, mod: info.ModTime()}
			found = true
		}
	}
	if !found {
		return "", fmt.Errorf("no matching artifacts found in %s", l.cfg.Path)
	}
	return best.path, nil
}

func matchesAny(name string, patterns []string) bool {
	for _, pattern := range patterns {
		ok, err := filepath.Match(pattern, name)
		if err == nil && ok {
			return true
		}
	}
	return false
}
