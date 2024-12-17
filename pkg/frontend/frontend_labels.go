package frontend

import (
	"context"

	"connectrpc.com/connect"

	typesv1 "github.com/grafana/pyroscope/api/gen/proto/go/types/v1"
)

func (f *Frontend) Labels(ctx context.Context, req *connect.Request[typesv1.LabelsRequest]) (*connect.Response[typesv1.LabelsResponse], error) {
	return nil, nil
}
