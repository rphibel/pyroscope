package util

import (
	"slices"
	"strings"

	typesv1 "github.com/grafana/pyroscope/api/gen/proto/go/types/v1"
)

// UniquelySortLabelSummaries merges and deduplicates label names and values.
func UniquelySortLabelSummaries(responses [][]*typesv1.LabelSummary) []*typesv1.LabelSummary {
	// Merge label names and their values, deduplicating along the way.
	unique := make(map[string]map[string]struct{})
	for _, response := range responses {
		for _, labels := range response {
			if _, ok := unique[labels.Name]; !ok {
				unique[labels.Name] = make(map[string]struct{})
			}

			for _, value := range labels.Values {
				unique[labels.Name][value] = struct{}{}
			}
		}
	}

	// Merge the deduplicated label names and values back into a slice, sorting
	// the values.
	result := make([]*typesv1.LabelSummary, 0, len(unique))
	for name, values := range unique {
		summary := &typesv1.LabelSummary{
			Name:   name,
			Values: make([]string, 0, len(values)),
		}

		for value := range values {
			summary.Values = append(summary.Values, value)
		}
		slices.Sort(summary.Values)
		result = append(result, summary)
	}

	// Lastly, sort by label name.
	slices.SortFunc(result, func(a, b *typesv1.LabelSummary) int {
		return strings.Compare(a.Name, b.Name)
	})
	return result
}
