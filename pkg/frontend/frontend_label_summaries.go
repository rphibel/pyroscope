package frontend

import (
	"context"

	"connectrpc.com/connect"

	"github.com/grafana/dskit/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/common/model"

	"github.com/grafana/pyroscope/api/gen/proto/go/querier/v1/querierv1connect"
	typesv1 "github.com/grafana/pyroscope/api/gen/proto/go/types/v1"
	phlaremodel "github.com/grafana/pyroscope/pkg/model"
	"github.com/grafana/pyroscope/pkg/util/connectgrpc"
	"github.com/grafana/pyroscope/pkg/validation"
)

func (f *Frontend) LabelSummaries(ctx context.Context, req *connect.Request[typesv1.LabelSummariesRequest]) (*connect.Response[typesv1.LabelSummariesResponse], error) {
	opentracing.SpanFromContext(ctx).
		SetTag("start", model.Time(req.Msg.Start).Time().String()).
		SetTag("end", model.Time(req.Msg.End).Time().String())

	ctx = connectgrpc.WithProcedure(ctx, querierv1connect.QuerierServiceLabelSummariesProcedure)

	tenantIDs, err := tenant.TenantIDs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	interval, ok := phlaremodel.GetTimeRange(req.Msg)
	if ok {
		validated, err := validation.ValidateRangeRequest(f.limits, tenantIDs, interval, model.Now())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		if validated.IsEmpty {
			return connect.NewResponse(&typesv1.LabelSummariesResponse{}), nil
		}

		req.Msg.Start = int64(validated.Start)
		req.Msg.End = int64(validated.End)
	}

	return connectgrpc.RoundTripUnary[typesv1.LabelSummariesRequest, typesv1.LabelSummariesResponse](ctx, f, req)
}
