package report

import (
	"fmt"
	"strings"

	"github.com/MB3R-Lab/Sheaft/internal/simulation"
)

func writeImportanceSummary(b *strings.Builder, outputs []simulation.ImportanceOutput) {
	if len(outputs) == 0 {
		return
	}
	b.WriteString("## Component criticality (Birnbaum importance)\n\n")
	for _, output := range outputs {
		b.WriteString(fmt.Sprintf("### `%s`\n\nBase profile: `%s`; paired trials: `%d`; confidence: `%.3f`; evaluated components: `%d`.\n\n", output.Name, output.BaseProfile, output.Trials, output.ConfidenceLevel, output.EvaluatedComponents))
		for _, endpoint := range output.Endpoints {
			b.WriteString(fmt.Sprintf("Endpoint `%s`; sampled baseline availability: `%.4f`.\n\n", endpoint.EndpointID, endpoint.BaselineAvailability))
			b.WriteString("| Rank | Component | Importance | Confidence interval | If up | If down | Perfect-component gain | Forced-down impact |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n")
			for _, component := range endpoint.Components {
				id := strings.ReplaceAll(component.ID, "|", "\\|")
				b.WriteString(fmt.Sprintf("| %d | `%s/%s` | %.4f | [%.4f, %.4f] | %.4f | %.4f | %.4f | %.4f |\n", component.Rank, component.Kind, id, component.BirnbaumImportance, component.Confidence.LowerBound, component.Confidence.UpperBound, component.AvailabilityIfUp, component.AvailabilityIfDown, component.PerfectComponentGain, component.ForcedDownImpact))
			}
			b.WriteString("\nRanks use point estimates; overlapping intervals do not establish a reliable ordering. Service importance refers to aggregate service liveness, including replicated services.\n\n")
		}
	}
}
