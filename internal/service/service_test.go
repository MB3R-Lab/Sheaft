package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/MB3R-Lab/Sheaft/internal/analyzer"
	"github.com/MB3R-Lab/Sheaft/internal/config"
	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
)

func TestService_WatchStatusAndMetrics(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "latest.json")
	if err := model.WriteToFile(artifactPath, testModel(1, "topology-1")); err != nil {
		t.Fatalf("write initial model: %v", err)
	}

	watchFS := false
	watchPolling := true
	serveCfg := config.ServeConfig{
		SchemaVersion: config.ServeSchemaVersion,
		Artifact: config.ArtifactSource{
			Path: artifactPath,
			Mode: "file",
		},
		PollInterval: "50ms",
		WatchFS:      &watchFS,
		WatchPolling: &watchPolling,
		History: config.HistoryConfig{
			MaxItems: 3,
		},
	}.Normalized()
	analysisCfg := config.Policy{
		Mode:               config.ModeWarn,
		DefaultAction:      config.ModeWarn,
		GlobalThreshold:    0.6,
		FailureProbability: 0.5,
		Trials:             4000,
	}.ToAnalysisConfig().Normalized()

	svc := newWithDeps(serveCfg, analysisCfg, realClock{}, newLocator(serveCfg.Artifact), nil, analyzer.AnalyzeLoaded, prometheus.NewRegistry())
	if err := svc.recomputeIfChanged(true); err != nil {
		t.Fatalf("initial recompute failed: %v", err)
	}
	initial := svc.CurrentReport()
	if initial == nil {
		t.Fatal("expected current report after initial recompute")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.watchLoop(ctx)

	if err := model.WriteToFile(artifactPath, testModel(3, "topology-2")); err != nil {
		t.Fatalf("write updated model: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		current := svc.CurrentReport()
		if current != nil && current.InputArtifact != nil && current.InputArtifact.TopologyVersion == "topology-2" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	current := svc.CurrentReport()
	if current == nil || current.InputArtifact == nil {
		t.Fatal("expected current report after watch update")
	}
	if current.InputArtifact.TopologyVersion != "topology-2" {
		t.Fatalf("expected updated topology version, got %+v", current.InputArtifact)
	}
	if current.InputArtifact.Digest == initial.InputArtifact.Digest {
		t.Fatalf("expected digest to change after update")
	}
	if len(svc.History()) < 2 {
		t.Fatalf("expected service history to retain both reports, got %d", len(svc.History()))
	}
	if current.Diffs.Previous == nil {
		t.Fatal("expected previous diff after second recompute")
	}

	handler := svc.Handler()

	statusRecorder := httptest.NewRecorder()
	handler.ServeHTTP(statusRecorder, httptest.NewRequest(http.MethodGet, "/status", nil))
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("status endpoint returned %d", statusRecorder.Code)
	}
	if !strings.Contains(statusRecorder.Body.String(), `"ready":true`) {
		t.Fatalf("expected ready status body, got %s", statusRecorder.Body.String())
	}

	metricsRecorder := httptest.NewRecorder()
	handler.ServeHTTP(metricsRecorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if metricsRecorder.Code != http.StatusOK {
		t.Fatalf("metrics endpoint returned %d", metricsRecorder.Code)
	}
	body := metricsRecorder.Body.String()
	for _, metric := range []string{"recomputes_total", "current_profile_aggregate_availability", "current_endpoint_availability", "active_model_info", "previous_gap"} {
		if !strings.Contains(body, metric) {
			t.Fatalf("expected metrics body to contain %s", metric)
		}
	}
}

func TestWatchLoop_InvalidPollIntervalSetsErrorInsteadOfPanicking(t *testing.T) {
	t.Parallel()

	watchPolling := true
	svc := newWithDeps(
		config.ServeConfig{
			SchemaVersion: config.ServeSchemaVersion,
			Artifact: config.ArtifactSource{
				Path: t.TempDir(),
				Mode: "directory",
			},
			PollInterval: "not-a-duration",
			WatchPolling: &watchPolling,
			History: config.HistoryConfig{
				MaxItems: 1,
			},
		},
		config.Policy{
			Mode:               config.ModeWarn,
			DefaultAction:      config.ModeWarn,
			GlobalThreshold:    0.9,
			FailureProbability: 0.1,
			Trials:             100,
		}.ToAnalysisConfig().Normalized(),
		realClock{},
		newLocator(config.ArtifactSource{Path: t.TempDir(), Mode: "directory", Patterns: []string{"*.json"}}),
		nil,
		analyzer.AnalyzeLoaded,
		prometheus.NewRegistry(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.watchLoop(ctx)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("watchLoop did not return on invalid poll interval")
	}
	cancel()

	status := svc.CurrentStatus()
	if !strings.Contains(status.LastError, "poll_interval") {
		t.Fatalf("expected poll interval error, got %+v", status)
	}
}

func testModel(replicas int, topologyVersion string) model.ResilienceModel {
	return model.ResilienceModel{
		Services: []model.Service{
			{ID: "frontend", Name: "frontend", Replicas: replicas},
		},
		Endpoints: []model.Endpoint{
			{ID: "frontend:GET /health", EntryService: "frontend", SuccessPredicateRef: "frontend:GET /health"},
		},
		Metadata: model.Metadata{
			SourceType:      "bering",
			SourceRef:       "bering://service-test",
			DiscoveredAt:    "2026-03-11T08:00:00Z",
			TopologyVersion: topologyVersion,
			Confidence:      0.8,
			Schema: model.Schema{
				Name:    modelcontract.ExpectedSchemaName,
				Version: modelcontract.ExpectedSchemaVersion,
				URI:     modelcontract.ExpectedSchemaURI,
				Digest:  modelcontract.ExpectedSchemaDigest,
			},
		},
	}
}
