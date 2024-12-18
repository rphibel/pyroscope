package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	typesv1 "github.com/grafana/pyroscope/api/gen/proto/go/types/v1"
)

func TestUniquelySortLabelSummaries(t *testing.T) {
	makeLabel := func(name string, values ...string) *typesv1.LabelSummary {
		return &typesv1.LabelSummary{
			Name:   name,
			Values: values,
		}
	}

	tests := []struct {
		name      string
		responses [][]*typesv1.LabelSummary
		want      []*typesv1.LabelSummary
	}{
		{
			name:      "empty",
			responses: [][]*typesv1.LabelSummary{},
			want:      []*typesv1.LabelSummary{},
		},
		{
			name: "unique_labels",
			responses: [][]*typesv1.LabelSummary{
				{
					makeLabel("service_name", "api", "web"),
					makeLabel("host", "host1", "host4"),
				},
				{
					makeLabel("service_name", "database"),
					makeLabel("host", "host2", "host3"),
				},
			},
			want: []*typesv1.LabelSummary{
				{
					Name: "host",
					Values: []string{
						"host1",
						"host2",
						"host3",
						"host4",
					},
				},
				{
					Name: "service_name",
					Values: []string{
						"api",
						"database",
						"web",
					},
				},
			},
		},
		{
			name: "repeated_labels",
			responses: [][]*typesv1.LabelSummary{
				{
					makeLabel("service_name", "api"),
					makeLabel("host", "host1"),
				},
				{
					makeLabel("service_name", "api"),
				},
			},
			want: []*typesv1.LabelSummary{
				{
					Name:   "host",
					Values: []string{"host1"},
				},
				{
					Name:   "service_name",
					Values: []string{"api"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniquelySortLabelSummaries(tt.responses)
			assert.Equal(t, tt.want, got)
		})
	}
}
