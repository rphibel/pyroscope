package query_frontend

import (
	"context"

	"connectrpc.com/connect"

	typesv1 "github.com/grafana/pyroscope/api/gen/proto/go/types/v1"
)

func (q *QueryFrontend) LabelSummaries(
	ctx context.Context,
	c *connect.Request[typesv1.LabelSummariesRequest],
) (*connect.Response[typesv1.LabelSummariesResponse], error) {
	// TODO(bryanhuhta): Implement
	panic("unimplemented")
}
