package provenance

const (
	SourceTypeOTelTraces = "otel_traces"
)

func DefaultConfidence(sourceType string) float64 {
	switch sourceType {
	case SourceTypeOTelTraces:
		return 0.72
	default:
		return 0.5
	}
}
