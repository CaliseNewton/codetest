package markettransform

import (
	"context"
	"testing"
	"time"

	"github.com/CaliseNewton/codetest/model"
	"github.com/stretchr/testify/assert"
)

func market(id string, status model.BettingStatus, closedAt *model.OptionalInt64) *model.Market {
	return &model.Market{ID: id, BettingStatus: &model.OptionalBettingStatus{Value: status}, ClosedAt: closedAt}
}

func TestTransformEvent(t *testing.T) {
	fixedNow := time.Date(2025, 9, 19, 11, 14, 3, 0, time.UTC)
	existingClose := &model.OptionalInt64{Value: 1}

	tests := []struct {
		name         string
		fullModel    *model.Event
		wantClosedAt []string // IDs of markets expected in the delta
		wantHidden   bool
	}{
		{
			name:      "no markets",
			fullModel: &model.Event{ID: "e1"},
		},
		{
			name:      "open and suspended markets are ignored",
			fullModel: &model.Event{ID: "e1", Markets: []*model.Market{market("m1", model.BettingStatus_BettingOpen, nil), market("m2", model.BettingStatus_BettingSuspended, nil)}},
		},
		{
			name:         "closed market gets ClosedAt",
			fullModel:    &model.Event{ID: "e1", Markets: []*model.Market{market("m1", model.BettingStatus_BettingClosed, nil)}},
			wantClosedAt: []string{"m1"},
		},
		{
			name:      "existing ClosedAt is kept",
			fullModel: &model.Event{ID: "e1", Markets: []*model.Market{market("m1", model.BettingStatus_BettingClosed, existingClose)}},
		},
		{
			name: "only the closed markets in a mix",
			fullModel: &model.Event{ID: "e1", Markets: []*model.Market{
				market("m1", model.BettingStatus_BettingOpen, nil),
				market("m2", model.BettingStatus_BettingClosed, nil),
				market("m3", model.BettingStatus_BettingClosed, existingClose),
			}},
			wantClosedAt: []string{"m2"},
		},
		{
			name:      "closed event with open markets does not close the markets",
			fullModel: &model.Event{ID: "e1", BettingStatus: &model.OptionalBettingStatus{Value: model.BettingStatus_BettingClosed}, Markets: []*model.Market{market("m1", model.BettingStatus_BettingOpen, nil)}},
		},
		{
			name:         "carries hidden forward",
			fullModel:    &model.Event{ID: "e1", Hidden: true, Markets: []*model.Market{market("m1", model.BettingStatus_BettingClosed, nil)}},
			wantClosedAt: []string{"m1"},
			wantHidden:   true,
		},
	}

	client := newMarketTransformClient(func() time.Time { return fixedNow })
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta, err := client.TransformEvent(context.Background(), &model.Event{ID: tt.fullModel.ID}, tt.fullModel)
			assert.NoError(t, err)

			if len(tt.wantClosedAt) == 0 {
				assert.Nil(t, delta)
				return
			}

			assert.Equal(t, tt.fullModel.ID, delta.GetID())
			assert.Equal(t, tt.wantHidden, delta.GetHidden())
			assert.Len(t, delta.GetMarkets(), len(tt.wantClosedAt))
			for i, id := range tt.wantClosedAt {
				assert.Equal(t, id, delta.GetMarkets()[i].GetID())
				assert.Equal(t, fixedNow.UnixNano(), delta.GetMarkets()[i].GetClosedAt().GetValue())
				assert.Nil(t, delta.GetMarkets()[i].GetBettingStatus(), "delta should only carry ClosedAt")
			}
		})
	}
}

func TestGetName(t *testing.T) {
	assert.Equal(t, "MarketTransform", NewMarketTransformClient().GetName())
}
