package otel

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MB3R-Lab/Sheaft/internal/model"
	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
	"github.com/MB3R-Lab/Sheaft/internal/provenance"
)

type spanRecord struct {
	Source   string
	Target   string
	Endpoint string
	Kind     model.EdgeKind
	Blocking bool
	Replicas int
}

func Discover(inputPath string) (model.ResilienceModel, error) {
	paths, err := collectInputFiles(inputPath)
	if err != nil {
		return model.ResilienceModel{}, err
	}
	if len(paths) == 0 {
		return model.ResilienceModel{}, fmt.Errorf("no json trace files found at %q", inputPath)
	}

	serviceReplicas := map[string]int{}
	edgeSet := map[string]model.Edge{}
	endpointSet := map[string]model.Endpoint{}
	recordsFound := 0

	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return model.ResilienceModel{}, fmt.Errorf("read trace input %q: %w", path, err)
		}

		records, err := parseRecords(raw)
		if err != nil {
			return model.ResilienceModel{}, fmt.Errorf("parse trace input %q: %w", path, err)
		}
		recordsFound += len(records)

		for _, rec := range records {
			if rec.Source == "" {
				continue
			}
			if rec.Replicas <= 0 {
				rec.Replicas = 1
			}
			if serviceReplicas[rec.Source] < rec.Replicas {
				serviceReplicas[rec.Source] = rec.Replicas
			}
			if rec.Target != "" && serviceReplicas[rec.Target] == 0 {
				serviceReplicas[rec.Target] = 1
			}
			if rec.Target != "" && rec.Target != rec.Source {
				key := edgeKey(rec.Source, rec.Target, rec.Kind, rec.Blocking)
				edgeSet[key] = model.Edge{
					From:     rec.Source,
					To:       rec.Target,
					Kind:     rec.Kind,
					Blocking: rec.Blocking,
				}
			}
			if rec.Endpoint != "" {
				id := rec.Source + ":" + rec.Endpoint
				endpointSet[id] = model.Endpoint{
					ID:                  id,
					EntryService:        rec.Source,
					SuccessPredicateRef: id,
				}
			}
		}
	}

	if recordsFound == 0 {
		return model.ResilienceModel{}, errors.New("no span records discovered from input traces")
	}
	if len(serviceReplicas) == 0 {
		return model.ResilienceModel{}, errors.New("discovery produced no services")
	}

	services := make([]model.Service, 0, len(serviceReplicas))
	for id, replicas := range serviceReplicas {
		services = append(services, model.Service{
			ID:       id,
			Name:     id,
			Replicas: replicas,
		})
	}
	sort.Slice(services, func(i, j int) bool { return services[i].ID < services[j].ID })

	edges := make([]model.Edge, 0, len(edgeSet))
	for _, edge := range edgeSet {
		edges = append(edges, edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		if edges[i].Kind != edges[j].Kind {
			return edges[i].Kind < edges[j].Kind
		}
		return !edges[i].Blocking && edges[j].Blocking
	})

	endpoints := make([]model.Endpoint, 0, len(endpointSet))
	for _, endpoint := range endpointSet {
		endpoints = append(endpoints, endpoint)
	}
	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].ID < endpoints[j].ID })

	mdl := model.ResilienceModel{
		Services:  services,
		Edges:     edges,
		Endpoints: endpoints,
		Metadata: model.Metadata{
			SourceType:   provenance.SourceTypeOTelTraces,
			SourceRef:    inputPath,
			DiscoveredAt: time.Now().UTC().Format(time.RFC3339),
			Confidence:   provenance.DefaultConfidence(provenance.SourceTypeOTelTraces),
			Schema: model.Schema{
				Name:    modelcontract.ExpectedSchemaName,
				Version: modelcontract.ExpectedSchemaVersion,
				URI:     modelcontract.ExpectedSchemaURI,
				Digest:  modelcontract.ExpectedSchemaDigest,
			},
		},
	}
	if err := mdl.Validate(); err != nil {
		return model.ResilienceModel{}, fmt.Errorf("validate discovered model: %w", err)
	}
	return mdl, nil
}

func collectInputFiles(inputPath string) ([]string, error) {
	info, err := os.Stat(inputPath)
	if err != nil {
		return nil, fmt.Errorf("stat input path: %w", err)
	}
	if !info.IsDir() {
		return []string{inputPath}, nil
	}

	files := []string{}
	err = filepath.WalkDir(inputPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".json") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk input directory: %w", err)
	}

	sort.Strings(files)
	return files, nil
}

func parseRecords(raw []byte) ([]spanRecord, error) {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("decode trace json: %w", err)
	}

	records := parseNormalized(root)
	if len(records) > 0 {
		return records, nil
	}

	return parseOTelLike(root), nil
}

func parseNormalized(root any) []spanRecord {
	obj, ok := root.(map[string]any)
	if !ok {
		return nil
	}
	rawSpans, ok := obj["spans"].([]any)
	if !ok {
		return nil
	}

	records := make([]spanRecord, 0, len(rawSpans))
	for _, rawSpan := range rawSpans {
		span, ok := rawSpan.(map[string]any)
		if !ok {
			continue
		}
		source := firstNonEmpty(
			toString(span["service"]),
			toString(span["service_name"]),
			toString(span["service.name"]),
		)
		if source == "" {
			continue
		}
		target := firstNonEmpty(
			toString(span["peer_service"]),
			toString(span["target"]),
			toString(span["callee"]),
			toString(span["peer.service"]),
		)
		endpoint := firstNonEmpty(
			toString(span["endpoint"]),
			toString(span["route"]),
			toString(span["name"]),
		)
		if endpoint == "" {
			endpoint = "unknown"
		}
		kind := parseKind(firstNonEmpty(toString(span["kind"]), "sync"))
		blocking := parseBoolWithDefault(span["blocking"], kind != model.EdgeKindAsync)
		replicas := parseIntWithDefault(span["replicas"], 1)
		records = append(records, spanRecord{
			Source:   source,
			Target:   target,
			Endpoint: endpoint,
			Kind:     kind,
			Blocking: blocking,
			Replicas: replicas,
		})
	}
	return records
}

func parseOTelLike(root any) []spanRecord {
	records := []spanRecord{}

	var walk func(any, string)
	walk = func(node any, inheritedService string) {
		switch v := node.(type) {
		case map[string]any:
			serviceCtx := firstNonEmpty(
				inheritedService,
				toString(v["service"]),
				attrString(v["attributes"], "service.name"),
			)
			if rawSpans, ok := v["spans"].([]any); ok {
				for _, rawSpan := range rawSpans {
					span, ok := rawSpan.(map[string]any)
					if !ok {
						continue
					}
					rec := parseOTelSpan(span, serviceCtx)
					if rec.Source != "" {
						records = append(records, rec)
					}
				}
			}
			for key, child := range v {
				if key == "spans" {
					continue
				}
				walk(child, serviceCtx)
			}
		case []any:
			for _, child := range v {
				walk(child, inheritedService)
			}
		}
	}

	walk(root, "")
	return records
}

func parseOTelSpan(span map[string]any, inheritedService string) spanRecord {
	attrs := span["attributes"]
	source := firstNonEmpty(
		toString(span["service"]),
		attrString(attrs, "service.name"),
		inheritedService,
	)
	target := firstNonEmpty(
		toString(span["peer_service"]),
		attrString(attrs, "peer.service"),
		attrString(attrs, "server.address"),
		attrString(attrs, "net.peer.name"),
		attrString(attrs, "rpc.service"),
	)
	endpoint := firstNonEmpty(
		toString(span["endpoint"]),
		attrString(attrs, "http.route"),
		attrString(attrs, "url.path"),
		toString(span["name"]),
	)
	if endpoint == "" {
		endpoint = "unknown"
	}

	kind := parseKind(firstNonEmpty(
		toString(span["kind"]),
		attrString(attrs, "sheaft.kind"),
	))

	if kind == "" && attrString(attrs, "messaging.system") != "" {
		kind = model.EdgeKindAsync
	}
	if kind == "" {
		kind = model.EdgeKindSync
	}

	blocking := parseBoolWithDefault(
		firstNonNil(span["blocking"], attrAny(attrs, "sheaft.blocking")),
		kind != model.EdgeKindAsync,
	)

	replicas := parseIntWithDefault(
		firstNonNil(span["replicas"], attrAny(attrs, "service.replicas")),
		1,
	)

	return spanRecord{
		Source:   source,
		Target:   target,
		Endpoint: endpoint,
		Kind:     kind,
		Blocking: blocking,
		Replicas: replicas,
	}
}

func parseKind(raw string) model.EdgeKind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "async":
		return model.EdgeKindAsync
	case "sync", "":
		return model.EdgeKindSync
	default:
		return model.EdgeKindSync
	}
}

func edgeKey(from, to string, kind model.EdgeKind, blocking bool) string {
	return fmt.Sprintf("%s|%s|%s|%t", from, to, kind, blocking)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	default:
		return ""
	}
}

func parseBoolWithDefault(v any, fallback bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, err := strconv.ParseBool(t)
		if err == nil {
			return b
		}
	case float64:
		return t != 0
	case int:
		return t != 0
	}
	return fallback
}

func parseIntWithDefault(v any, fallback int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		i, err := strconv.Atoi(t)
		if err == nil {
			return i
		}
	}
	return fallback
}

func attrString(attrs any, key string) string {
	val := attrAny(attrs, key)
	return toString(otelValue(val))
}

func attrAny(attrs any, key string) any {
	switch typed := attrs.(type) {
	case map[string]any:
		return typed[key]
	case []any:
		for _, item := range typed {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			k := toString(obj["key"])
			if k != key {
				continue
			}
			if value, ok := obj["value"]; ok {
				return value
			}
			return obj[key]
		}
	}
	return nil
}

func otelValue(v any) any {
	obj, ok := v.(map[string]any)
	if !ok {
		return v
	}
	for _, key := range []string{"stringValue", "intValue", "doubleValue", "boolValue", "value"} {
		if val, exists := obj[key]; exists {
			return otelValue(val)
		}
	}
	return v
}
