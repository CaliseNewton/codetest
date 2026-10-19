package racingtransform

import (
	"context"
	"testing"

	"github.com/CaliseNewton/codetest/model"
	"github.com/stretchr/testify/assert"
)

func TestTransformEvent(t *testing.T) {
	horse := &model.OptionalString{Value: "horse_racing"}

	tests := []struct {
		name          string
		fullModel     *model.Event
		wantDelta     bool
		wantRaceType  *model.OptionalRaceType
		wantName      *model.OptionalString
		wantHiddenSet bool
	}{
		{
			name:      "not a racing event",
			fullModel: &model.Event{ID: "e1", EventTypeID: &model.OptionalString{Value: "soccer"}},
			wantDelta: false,
		},
		{
			name:      "no event type",
			fullModel: &model.Event{ID: "e1"},
			wantDelta: false,
		},
		{
			name:         "sets race type from event type",
			fullModel:    &model.Event{ID: "e1", EventTypeID: &model.OptionalString{Value: "greyhound_racing"}},
			wantDelta:    true,
			wantRaceType: &model.OptionalRaceType{Value: model.RaceType_Greyhound},
		},
		{
			name: "builds name from venue and race number",
			fullModel: &model.Event{ID: "e1", EventTypeID: horse, RacingData: &model.RaceEvent{
				RaceType:   &model.OptionalRaceType{Value: model.RaceType_Horse},
				Venue:      &model.OptionalString{Value: "Randwick"},
				RaceNumber: &model.OptionalInt64{Value: 3},
			}},
			wantDelta: true,
			wantName:  &model.OptionalString{Value: "Randwick R3"},
		},
		{
			name: "keeps a supplied name",
			fullModel: &model.Event{ID: "e1", EventTypeID: horse, Name: &model.OptionalString{Value: "The Golden Slipper"}, RacingData: &model.RaceEvent{
				RaceType:   &model.OptionalRaceType{Value: model.RaceType_Horse},
				Venue:      &model.OptionalString{Value: "Rosehill"},
				RaceNumber: &model.OptionalInt64{Value: 7},
			}},
			wantDelta: false,
		},
		{
			name: "no name without race number",
			fullModel: &model.Event{ID: "e1", EventTypeID: horse, RacingData: &model.RaceEvent{
				RaceType: &model.OptionalRaceType{Value: model.RaceType_Horse},
				Venue:    &model.OptionalString{Value: "Randwick"},
			}},
			wantDelta: false,
		},
		{
			name:          "carries hidden forward",
			fullModel:     &model.Event{ID: "e1", EventTypeID: &model.OptionalString{Value: "harness_racing"}, Hidden: true},
			wantDelta:     true,
			wantRaceType:  &model.OptionalRaceType{Value: model.RaceType_Harness},
			wantHiddenSet: true,
		},
	}

	client := NewRacingTransformClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta, err := client.TransformEvent(context.Background(), &model.Event{ID: tt.fullModel.ID}, tt.fullModel)
			assert.NoError(t, err)

			if !tt.wantDelta {
				assert.Nil(t, delta)
				return
			}

			assert.NotNil(t, delta)
			assert.Equal(t, tt.fullModel.ID, delta.GetID())
			assert.Equal(t, tt.wantRaceType.GetValue(), delta.GetRacingData().GetRaceType().GetValue())
			assert.Equal(t, tt.wantRaceType == nil, delta.GetRacingData().GetRaceType() == nil)
			assert.Equal(t, tt.wantName.GetValue(), delta.GetName().GetValue())
			assert.Equal(t, tt.wantHiddenSet, delta.GetHidden())
		})
	}
}

func TestGetName(t *testing.T) {
	assert.Equal(t, "RacingTransform", NewRacingTransformClient().GetName())
}
