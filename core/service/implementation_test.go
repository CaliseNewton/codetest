package service_test

import (
	"context"
	"testing"

	"github.com/CaliseNewton/codetest/core"
	"github.com/CaliseNewton/codetest/core/repository"
	"github.com/CaliseNewton/codetest/core/service"
	"github.com/CaliseNewton/codetest/core/transforms"
	"github.com/CaliseNewton/codetest/core/transforms/racingtransform"
	"github.com/CaliseNewton/codetest/core/transforms/sporttransform"
	"github.com/CaliseNewton/codetest/merger"
	"github.com/CaliseNewton/codetest/model"
	"github.com/stretchr/testify/assert"
)

func TestService_IntegrationTest_NewEvent(t *testing.T) {
	repo, err := repository.NewRedisRepository(context.Background(), "localhost:6379", "")
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
	repo, err := repository.NewRedisRepository(context.Background(), "localhost:6379", "")
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
	repo, err := repository.NewRedisRepository(context.Background(), "localhost:6379", "")
	assert.NoError(t, err)
	host := &service.Service{Upstreams: &service.Upstreams{Repo: repo}}

	resp, err := host.GetRacingEvent(context.Background(), &core.GetRacingEventRequest{EventID: "integration-test-does-not-exist"})
	assert.NoError(t, err)
	assert.Nil(t, resp.Event)
}
