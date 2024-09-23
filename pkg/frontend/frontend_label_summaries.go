package frontend

import (
	"context"

	"connectrpc.com/connect"

	typesv1 "github.com/grafana/pyroscope/api/gen/proto/go/types/v1"
)

func (f *Frontend) LabelSummaries(ctx context.Context, c *connect.Request[typesv1.LabelSummariesRequest]) (*connect.Response[typesv1.LabelSummariesResponse], error) {
	panic("unimplemented")
}
