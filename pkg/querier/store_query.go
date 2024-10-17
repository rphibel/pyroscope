package querier

import (
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/common/model"

	ingestv1 "github.com/grafana/pyroscope/api/gen/proto/go/ingester/v1"
	querierv1 "github.com/grafana/pyroscope/api/gen/proto/go/querier/v1"
	typesv1 "github.com/grafana/pyroscope/api/gen/proto/go/types/v1"
	pmath "github.com/grafana/pyroscope/pkg/util/math"
)

// splitQueryToStores splits the query into ingester and store gateway queries using the given cut off time.
// todo(ctovena): Later we should try to deduplicate blocks between ingesters and store gateways (prefer) and simply query both
func splitQueryToStores(start, end model.Time, now model.Time, queryStoreAfter time.Duration, plan blockPlan) (queries storeQueries) {
	if plan != nil {
		// if we have a plan we can use it to split the query, we retain the original start and end time as we want to query the full range for those particular blocks selected.
		queries.queryStoreAfter = 0
		queries.ingester = storeQuery{shouldQuery: true, start: start, end: end}
		queries.storeGateway = storeQuery{shouldQuery: true, start: start, end: end}
		return queries
	}

	queries.queryStoreAfter = queryStoreAfter
	cutOff := now.Add(-queryStoreAfter)
	if start.Before(cutOff) {
		queries.storeGateway = storeQuery{shouldQuery: true, start: start, end: pmath.Min(cutOff, end)}
	}
	if end.After(cutOff) {
		queries.ingester = storeQuery{shouldQuery: true, start: pmath.Max(cutOff, start), end: end}
		// Note that the ranges must not overlap.
		if queries.storeGateway.shouldQuery {
			queries.ingester.start++
		}
	}
	return queries
}

type storeQuery struct {
	start, end  model.Time
	shouldQuery bool
}

func (sq storeQuery) MergeStacktracesRequest(req *querierv1.SelectMergeStacktracesRequest) *querierv1.SelectMergeStacktracesRequest {
	return &querierv1.SelectMergeStacktracesRequest{
		Start:         int64(sq.start),
		End:           int64(sq.end),
		LabelSelector: req.LabelSelector,
		ProfileTypeID: req.ProfileTypeID,
		MaxNodes:      req.MaxNodes,
		Format:        req.Format,
	}
}

func (sq storeQuery) MergeSeriesRequest(req *querierv1.SelectSeriesRequest, profileType *typesv1.ProfileType) *ingestv1.MergeProfilesLabelsRequest {
	return &ingestv1.MergeProfilesLabelsRequest{
		Request: &ingestv1.SelectProfilesRequest{
			Type:          profileType,
			LabelSelector: req.LabelSelector,
			Start:         int64(sq.start),
			End:           int64(sq.end),
			Aggregation:   req.Aggregation,
		},
		By:                 req.GroupBy,
		StackTraceSelector: req.StackTraceSelector,
	}
}

func (sq storeQuery) MergeSpanProfileRequest(req *querierv1.SelectMergeSpanProfileRequest) *querierv1.SelectMergeSpanProfileRequest {
	return &querierv1.SelectMergeSpanProfileRequest{
		Start:         int64(sq.start),
		End:           int64(sq.end),
		ProfileTypeID: req.ProfileTypeID,
		LabelSelector: req.LabelSelector,
		SpanSelector:  req.SpanSelector,
		MaxNodes:      req.MaxNodes,
		Format:        req.Format,
	}
}

func (sq storeQuery) MergeProfileRequest(req *querierv1.SelectMergeProfileRequest) *querierv1.SelectMergeProfileRequest {
	return &querierv1.SelectMergeProfileRequest{
		ProfileTypeID:      req.ProfileTypeID,
		LabelSelector:      req.LabelSelector,
		Start:              int64(sq.start),
		End:                int64(sq.end),
		MaxNodes:           req.MaxNodes,
		StackTraceSelector: req.StackTraceSelector,
	}
}

func (sq storeQuery) SeriesRequest(req *querierv1.SeriesRequest) *ingestv1.SeriesRequest {
	return &ingestv1.SeriesRequest{
		Start:      int64(sq.start),
		End:        int64(sq.end),
		Matchers:   req.Matchers,
		LabelNames: req.LabelNames,
	}
}

func (sq storeQuery) LabelNamesRequest(req *typesv1.LabelNamesRequest) *typesv1.LabelNamesRequest {
	return &typesv1.LabelNamesRequest{
		Matchers: req.Matchers,
		Start:    int64(sq.start),
		End:      int64(sq.end),
	}
}

func (sq storeQuery) LabelValuesRequest(req *typesv1.LabelValuesRequest) *typesv1.LabelValuesRequest {
	return &typesv1.LabelValuesRequest{
		Name:     req.Name,
		Matchers: req.Matchers,
		Start:    int64(sq.start),
		End:      int64(sq.end),
	}
}

func (sq storeQuery) LabelSummaryRequest(req *typesv1.LabelSummariesRequest) *typesv1.LabelSummariesRequest {
	return &typesv1.LabelSummariesRequest{
		Matchers: req.Matchers,
		Start:    int64(sq.start),
		End:      int64(sq.end),
	}
}

func (sq storeQuery) ProfileTypesRequest(req *querierv1.ProfileTypesRequest) *ingestv1.ProfileTypesRequest {
	return &ingestv1.ProfileTypesRequest{
		Start: int64(sq.start),
		End:   int64(sq.end),
	}
}

type storeQueries struct {
	ingester, storeGateway storeQuery
	queryStoreAfter        time.Duration
}

func (sq storeQueries) Log(logger log.Logger) {
	logger.Log(
		"msg", "storeQueries",
		"queryStoreAfter", sq.queryStoreAfter.String(),
		"ingester", sq.ingester.shouldQuery,
		"ingester.start", sq.ingester.start.Time().Format(time.RFC3339Nano), "ingester.end", sq.ingester.end.Time().Format(time.RFC3339Nano),
		"store-gateway", sq.storeGateway.shouldQuery,
		"store-gateway.start", sq.storeGateway.start.Time().Format(time.RFC3339Nano), "store-gateway.end", sq.storeGateway.end.Time().Format(time.RFC3339Nano),
	)
}
