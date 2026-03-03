package observability

type ChangeSignal struct {
	TopologyVersion string
	Source          string
}

func ShouldRecompute(signal ChangeSignal) bool {
	return signal.TopologyVersion != "" || signal.Source != ""
}
