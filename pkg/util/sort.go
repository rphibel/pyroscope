package util

import (
	"slices"
	"sort"

	typesv1 "github.com/grafana/pyroscope/api/gen/proto/go/types/v1"
)

// UniqueSortSummaries sorts the summaries by name, combining any duplicates.
// This operation is done in-place, so the input slice is modified and returned.
//
// It assumes each label summary has its values sorted.
func UniqueSortSummaries(summaries []*typesv1.LabelSummary) []*typesv1.LabelSummary {
	uniqueSummaries := map[string][]*typesv1.LabelSummary{}
	for _, s := range summaries {
		uniqueSummaries[s.Name] = append(uniqueSummaries[s.Name], s)
	}

	summaries = summaries[:0]
	for _, sums := range uniqueSummaries {
		if len(sums) == 0 {
			continue
		}

		if len(sums) == 1 {
			summaries = append(summaries, sums[0])
			continue
		}

		// Aggregate and deduplicate the values.
		for i := 1; i < len(sums); i++ {
			sums[0].Values = append(sums[0].Values, sums[i].Values...)
		}
		sums[0].Values = slices.Compact(sums[0].Values)
		summaries = append(summaries, sums[0])
	}

	sorter := labelSummarySorter(summaries)
	sort.Sort(sorter)
	return summaries
}

// labelSummarySorter defines a sort implementation for LabelSummary.
type labelSummarySorter []*typesv1.LabelSummary

func (a labelSummarySorter) Len() int           { return len(a) }
func (a labelSummarySorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a labelSummarySorter) Less(i, j int) bool { return a[i].Name < a[j].Name }
