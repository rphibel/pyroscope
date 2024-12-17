package util

import (
	"slices"
	"strings"

	typesv1 "github.com/grafana/pyroscope/api/gen/proto/go/types/v1"
)

func UniqueSortLabels(responses [][]*typesv1.LabelValues) []*typesv1.LabelValues {
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
	result := make([]*typesv1.LabelValues, 0, len(unique))
	for name, values := range unique {
		label := &typesv1.LabelValues{
			Name:   name,
			Values: make([]string, 0, len(values)),
		}

		for value := range values {
			label.Values = append(label.Values, value)
		}
		slices.Sort(label.Values)
		result = append(result, label)
	}

	// Lastly, sort by label name.
	slices.SortFunc(result, func(a, b *typesv1.LabelValues) int {
		return strings.Compare(a.Name, b.Name)
	})
	return result
}
