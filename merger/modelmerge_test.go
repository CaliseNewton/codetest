package merger

import (
	"context"
	"testing"

	"github.com/CaliseNewton/codetest/model"
	"github.com/stretchr/testify/assert"
)

func TestMergeRaceEvent_NilCases(t *testing.T) {
	ctx := context.Background()
	race := &model.RaceEvent{Venue: &model.OptionalString{Value: "Randwick"}}

	assert.Same(t, race, MergeRaceEvent(ctx, race, nil))
	assert.Same(t, race, MergeRaceEvent(ctx, nil, race))
	assert.Nil(t, MergeRaceEvent(ctx, nil, nil))
}

func TestMergeRaceEvent_PartialUpdateKeepsUnsentFields(t *testing.T) {
	ctx := context.Background()
	left := &model.RaceEvent{
		RaceType:       &model.OptionalRaceType{Value: model.RaceType_Horse},
		Venue:          &model.OptionalString{Value: "Randwick"},
		RaceNumber:     &model.OptionalInt64{Value: 3},
		DistanceMetres: &model.OptionalInt64{Value: 1200},
		TrackCondition: &model.OptionalString{Value: "Good 4"},
		Weather:        &model.OptionalString{Value: "Fine"},
		Country:        &model.OptionalString{Value: "AUS"},
		RaceClass:      &model.OptionalString{Value: "Group 1"},
	}
	right := &model.RaceEvent{
		TrackCondition: &model.OptionalString{Value: "Heavy 8"},
	}

	result := MergeRaceEvent(ctx, left, right)

	assert.Equal(t, model.RaceType_Horse, result.GetRaceType().GetValue())
	assert.Equal(t, "Randwick", result.GetVenue().GetValue())
	assert.Equal(t, int64(3), result.GetRaceNumber().GetValue())
	assert.Equal(t, int64(1200), result.GetDistanceMetres().GetValue())
	assert.Equal(t, "Heavy 8", result.GetTrackCondition().GetValue())
	assert.Equal(t, "Fine", result.GetWeather().GetValue())
	assert.Equal(t, "AUS", result.GetCountry().GetValue())
	assert.Equal(t, "Group 1", result.GetRaceClass().GetValue())
}

func TestMergeOptionalRaceType_RightOverwritesLeft(t *testing.T) {
	ctx := context.Background()
	left := &model.OptionalRaceType{Value: model.RaceType_Horse}
	right := &model.OptionalRaceType{Value: model.RaceType_Greyhound, Deleted: true}

	result := MergeOptionalRaceType(ctx, left, right)

	assert.Equal(t, model.RaceType_Greyhound, result.GetValue())
	assert.True(t, result.GetDeleted())
}

func TestMergeEvent_MergesRacingData(t *testing.T) {
	ctx := context.Background()
	left := &model.Event{ID: "e1", RacingData: &model.RaceEvent{Venue: &model.OptionalString{Value: "Randwick"}}}
	right := &model.Event{ID: "e1", RacingData: &model.RaceEvent{RaceNumber: &model.OptionalInt64{Value: 3}}}

	result := MergeEvent(ctx, left, right)

	assert.Equal(t, "Randwick", result.GetRacingData().GetVenue().GetValue())
	assert.Equal(t, int64(3), result.GetRacingData().GetRaceNumber().GetValue())
}

// Pins the current behaviour that Hidden is always taken from the right operand, because a plain bool
// cannot tell "not sent" apart from false. Any update or transform delta that leaves Hidden out makes the
// event visible again. See the Assumptions section of CALISE-README.md.
func TestMergeEvent_HiddenTakenFromRight(t *testing.T) {
	ctx := context.Background()
	hidden := &model.Event{ID: "e1", Hidden: true}

	assert.True(t, MergeEvent(ctx, hidden, &model.Event{ID: "e1", Hidden: true}).GetHidden())
	assert.False(t, MergeEvent(ctx, hidden, &model.Event{ID: "e1"}).GetHidden())
	assert.True(t, MergeEvent(ctx, &model.Event{ID: "e1"}, hidden).GetHidden())
}
