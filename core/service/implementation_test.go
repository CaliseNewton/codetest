package service_test

import (
	"context"
	"testing"

	"github.com/CaliseNewton/codetest/core"
	"github.com/CaliseNewton/codetest/core/repository"
	"github.com/CaliseNewton/codetest/core/service"
	"github.com/CaliseNewton/codetest/core/transforms"
	"github.com/CaliseNewton/codetest/core/transforms/markettransform"
	"github.com/CaliseNewton/codetest/core/transforms/racingtransform"
	"github.com/CaliseNewton/codetest/core/transforms/sporttransform"
	"github.com/CaliseNewton/codetest/merger"
	"github.com/CaliseNewton/codetest/model"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestService_IntegrationTest_NewEvent(t *testing.T) {
	repo, err := repository.NewMongoRepository(context.Background(), "mongodb://localhost:27017", "codetest_service_search_test")
	assert.NoError(t, err)
	defer repo.DeleteEventByID(context.Background(), "integration-test-1")
	host := &service.Service{
		Upstreams: &service.Upstreams{
			MergerClient: merger.NewInlineMergerClient(),
			Repo:         repo,
			Transforms: []transforms.TransformClient{
				sporttransform.NewSportTransformClient(),
			},
		},
	}

	output, err := host.Update(context.Background(), &core.UpdateRequest{
		Event: &model.Event{
			ID:          "integration-test-1",
			Name:        &model.OptionalString{Value: "Test event"},
			EventTypeID: &model.OptionalString{Value: "soccer"},
			StartTime:   &model.OptionalInt64{Value: 1758244443000000000}, // Friday, September 19, 2025 11:14:03 AM GMT+10:00
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "New Event born integration-test-1", output.Message)

	output, err = host.Update(context.Background(), &core.UpdateRequest{
		Event: &model.Event{
			ID: "integration-test-1",
			Markets: []*model.Market{
				{
					ID:   "mkt01",
					Name: &model.OptionalString{Value: "New Market"},
				},
			},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "Success", output.Message)

	final, err := host.GetSportEvent(context.Background(), &core.GetSportEventRequest{EventID: "integration-test-1"})
	assert.NoError(t, err)
	assert.Equal(t, "Test event", final.Event.Name)
	assert.Contains(t, final.Event.StartTime, "2025-09-19")
	assert.Equal(t, "soccer", final.Event.SportTypeID)
	assert.Equal(t, "Soccer", final.Event.SportName)
	assert.Equal(t, "New Market", final.Event.Markets[0].Name.Value)
	assert.Equal(t, false, final.Event.Hidden)

	// A sport event is not a race, so GetRacingEvent returns an empty response
	racing, err := host.GetRacingEvent(context.Background(), &core.GetRacingEventRequest{EventID: "integration-test-1"})
	assert.NoError(t, err)
	assert.Nil(t, racing.Event)
}

func TestService_IntegrationTest_NewRacingEvent(t *testing.T) {
	repo, err := repository.NewMongoRepository(context.Background(), "mongodb://localhost:27017", "codetest_service_search_test")
	assert.NoError(t, err)
	defer repo.DeleteEventByID(context.Background(), "integration-test-race-1")
	host := &service.Service{
		Upstreams: &service.Upstreams{
			MergerClient: merger.NewInlineMergerClient(),
			Repo:         repo,
			Transforms: []transforms.TransformClient{
				sporttransform.NewSportTransformClient(),
				racingtransform.NewRacingTransformClient(),
			},
		},
	}

	output, err := host.Update(context.Background(), &core.UpdateRequest{
		Event: &model.Event{
			ID:          "integration-test-race-1",
			EventTypeID: &model.OptionalString{Value: "horse_racing"},
			StartTime:   &model.OptionalInt64{Value: 1758244443000000000}, // Friday, September 19, 2025 11:14:03 AM GMT+10:00
			Hidden:      true,
			RacingData: &model.RaceEvent{
				Venue:          &model.OptionalString{Value: "Randwick"},
				RaceNumber:     &model.OptionalInt64{Value: 3},
				DistanceMetres: &model.OptionalInt64{Value: 1200},
				TrackCondition: &model.OptionalString{Value: "Good 4"},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "New Event born integration-test-race-1", output.Message)

	// Partial update: only the track condition changes, Hidden is resent as the merge takes it from the update
	output, err = host.Update(context.Background(), &core.UpdateRequest{
		Event: &model.Event{
			ID:         "integration-test-race-1",
			Hidden:     true,
			RacingData: &model.RaceEvent{TrackCondition: &model.OptionalString{Value: "Heavy 8"}},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "Success", output.Message)

	final, err := host.GetRacingEvent(context.Background(), &core.GetRacingEventRequest{EventID: "integration-test-race-1"})
	assert.NoError(t, err)
	assert.Equal(t, "Randwick R3", final.Event.Name)
	assert.Equal(t, "Horse", final.Event.RaceType)
	assert.Equal(t, "Randwick", final.Event.Venue)
	assert.Equal(t, int64(3), final.Event.RaceNumber)
	assert.Equal(t, int64(1200), final.Event.DistanceMetres)
	assert.Equal(t, "Heavy 8", final.Event.TrackCondition)
	assert.Contains(t, final.Event.StartTime, "2025-09-19")
	assert.Equal(t, true, final.Event.Hidden)
}

func TestService_IntegrationTest_GetRacingEventMissing(t *testing.T) {
	repo, err := repository.NewMongoRepository(context.Background(), "mongodb://localhost:27017", "codetest_service_search_test")
	assert.NoError(t, err)
	host := &service.Service{Upstreams: &service.Upstreams{Repo: repo}}

	resp, err := host.GetRacingEvent(context.Background(), &core.GetRacingEventRequest{EventID: "integration-test-does-not-exist"})
	assert.NoError(t, err)
	assert.Nil(t, resp.Event)
}

func TestService_IntegrationTest_MarketClosedAt(t *testing.T) {
	repo, err := repository.NewMongoRepository(context.Background(), "localhost:6379", "")
	assert.NoError(t, err)
	defer repo.DeleteEventByID(context.Background(), "integration-test-market-1")
	host := &service.Service{
		Upstreams: &service.Upstreams{
			MergerClient: merger.NewInlineMergerClient(),
			Repo:         repo,
			Transforms: []transforms.TransformClient{
				sporttransform.NewSportTransformClient(),
				racingtransform.NewRacingTransformClient(),
				markettransform.NewMarketTransformClient(),
			},
		},
	}
	setStatus := func(status model.BettingStatus, clientClosedAt *model.OptionalInt64) {
		_, err := host.Update(context.Background(), &core.UpdateRequest{
			Event: &model.Event{
				ID:     "integration-test-market-1",
				Hidden: true,
				Markets: []*model.Market{
					{ID: "mkt01", BettingStatus: &model.OptionalBettingStatus{Value: status}, ClosedAt: clientClosedAt},
				},
			},
		})
		assert.NoError(t, err)
	}
	getMarket := func() (*model.Market, bool) {
		final, err := host.GetSportEvent(context.Background(), &core.GetSportEventRequest{EventID: "integration-test-market-1"})
		assert.NoError(t, err)
		return final.Event.Markets[0], final.Event.Hidden
	}

	// Open market, and a client supplied ClosedAt is ignored
	setStatus(model.BettingStatus_BettingOpen, &model.OptionalInt64{Value: 1})
	mkt, _ := getMarket()
	assert.Nil(t, mkt.ClosedAt)

	// First close sets ClosedAt and keeps the event hidden
	setStatus(model.BettingStatus_BettingClosed, nil)
	mkt, hidden := getMarket()
	assert.NotNil(t, mkt.ClosedAt)
	assert.True(t, hidden)
	firstClose := mkt.ClosedAt.Value

	// Reopen then close again, ClosedAt still records the first close
	setStatus(model.BettingStatus_BettingOpen, nil)
	setStatus(model.BettingStatus_BettingClosed, &model.OptionalInt64{Value: 1})
	mkt, _ = getMarket()
	assert.Equal(t, firstClose, mkt.ClosedAt.Value)
}

// Invalid requests are rejected before the repository is used, so no database is needed
func TestService_SearchEvents_Validation(t *testing.T) {
	host := &service.Service{Upstreams: &service.Upstreams{}}
	unknown := model.BettingStatus_BettingUnknown

	tests := []struct {
		name string
		req  *core.SearchEventsRequest
	}{
		{name: "no criteria", req: &core.SearchEventsRequest{}},
		{name: "StartFrom not RFC3339", req: &core.SearchEventsRequest{StartFrom: "19/09/2025"}},
		{name: "StartTo not RFC3339", req: &core.SearchEventsRequest{StartTo: "2025-09-19"}},
		{name: "StartFrom after StartTo", req: &core.SearchEventsRequest{StartFrom: "2025-09-20T00:00:00Z", StartTo: "2025-09-19T00:00:00Z"}},
		{name: "StartFrom equal to StartTo", req: &core.SearchEventsRequest{StartFrom: "2025-09-19T00:00:00Z", StartTo: "2025-09-19T00:00:00Z"}},
		{name: "BettingUnknown", req: &core.SearchEventsRequest{BettingStatus: &unknown}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := host.SearchEvents(context.Background(), tt.req)
			assert.Nil(t, resp)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestService_IntegrationTest_SearchEvents(t *testing.T) {
	repo, err := repository.NewMongoRepository(context.Background(), "mongodb://localhost:27017", "codetest_service_search_test")
	assert.NoError(t, err)
	host := &service.Service{
		Upstreams: &service.Upstreams{
			MergerClient: merger.NewInlineMergerClient(),
			Repo:         repo,
		},
	}
	defer repo.DeleteEventByID(context.Background(), "integration-test-search-1")
	defer repo.DeleteEventByID(context.Background(), "integration-test-search-2")

	for id, start := range map[string]int64{
		"integration-test-search-1": 1758276000000000000, // 2025-09-19T10:00:00Z
		"integration-test-search-2": 1758362400000000000, // 2025-09-20T10:00:00Z
	} {
		_, err := host.Update(context.Background(), &core.UpdateRequest{Event: &model.Event{
			ID:            id,
			StartTime:     &model.OptionalInt64{Value: start},
			BettingStatus: &model.OptionalBettingStatus{Value: model.BettingStatus_BettingOpen},
		}})
		assert.NoError(t, err)
	}

	open := model.BettingStatus_BettingOpen
	resp, err := host.SearchEvents(context.Background(), &core.SearchEventsRequest{
		StartFrom:     "2025-09-19T00:00:00Z",
		StartTo:       "2025-09-20T00:00:00Z",
		BettingStatus: &open,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Events, 1)
	assert.Equal(t, "integration-test-search-1", resp.Events[0].GetID())
}
