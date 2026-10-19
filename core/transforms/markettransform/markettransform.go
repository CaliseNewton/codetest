// Package markettransform supplies a markettransformClient
package markettransform

import (
	"context"
	"time"

	"github.com/CaliseNewton/codetest/core/transforms"
	"github.com/CaliseNewton/codetest/model"
)

type markettransformClient struct {
	now func() time.Time
}

// NewMarketTransformClient creates a new Market transform client
func NewMarketTransformClient() transforms.TransformClient {
	return newMarketTransformClient(time.Now)
}

// newMarketTransformClient allows tests to supply a fixed clock
func newMarketTransformClient(now func() time.Time) transforms.TransformClient {
	return &markettransformClient{now: now}
}

// TransformEvent sets ClosedAt on every market that is closed and does not have a ClosedAt yet.
// Only the market's own BettingStatus counts, and an existing ClosedAt is never overwritten so it always records the first close.
func (t *markettransformClient) TransformEvent(_ context.Context, parialUpdate, fullModel *model.Event) (*model.Event, error) {
	var closedMarkets []*model.Market
	closedAt := t.now().UnixNano()

	for _, market := range fullModel.GetMarkets() {
		if market.GetBettingStatus().GetValue() != model.BettingStatus_BettingClosed || market.GetClosedAt() != nil {
			continue
		}
		closedMarkets = append(closedMarkets, &model.Market{ID: market.GetID(), ClosedAt: &model.OptionalInt64{Value: closedAt}})
	}

	if len(closedMarkets) == 0 {
		return nil, nil
	}

	// Carry Hidden forward from the full model, otherwise the merge treats the missing bool as false and resets it
	return &model.Event{
		ID:      parialUpdate.ID,
		Hidden:  fullModel.GetHidden(),
		Markets: closedMarkets,
	}, nil
}

func (t *markettransformClient) GetName() string {
	return "MarketTransform"
}
