// Package racingtransform supplies a racingtransformClient
package racingtransform

import (
	"context"
	"fmt"

	"github.com/CaliseNewton/codetest/core/transforms"
	"github.com/CaliseNewton/codetest/model"
)

type racingtransformClient struct{}

// NewRacingTransformClient creates a new Racing transform client
func NewRacingTransformClient() transforms.TransformClient {
	return &racingtransformClient{}
}

var raceTypeMap = map[string]model.RaceType{
	"horse_racing":     model.RaceType_Horse,
	"greyhound_racing": model.RaceType_Greyhound,
	"harness_racing":   model.RaceType_Harness,
}

// TransformEvent performs racing specific transformation on the Event.
// It only fills in values that are missing, so running it again on the same event is a no-op.
func (t *racingtransformClient) TransformEvent(_ context.Context, parialUpdate, fullModel *model.Event) (*model.Event, error) {
	raceType, isRace := raceTypeMap[fullModel.GetEventTypeID().GetValue()]
	if !isRace {
		return nil, nil // not a racing event so dont change anything
	}

	racingDelta := &model.RaceEvent{}
	changed := false

	if fullModel.GetRacingData().GetRaceType() == nil {
		racingDelta.RaceType = &model.OptionalRaceType{Value: raceType}
		changed = true
	}

	var nameDelta *model.OptionalString
	venue := fullModel.GetRacingData().GetVenue().GetValue()
	raceNumber := fullModel.GetRacingData().GetRaceNumber().GetValue()
	if fullModel.GetName() == nil && venue != "" && raceNumber > 0 {
		nameDelta = &model.OptionalString{Value: fmt.Sprintf("%v R%v", venue, raceNumber)}
		changed = true
	}

	if !changed {
		return nil, nil
	}

	// Carry Hidden forward from the full model, otherwise the merge treats the missing bool as false and resets it
	return &model.Event{
		ID:         parialUpdate.ID,
		Name:       nameDelta,
		Hidden:     fullModel.GetHidden(),
		RacingData: racingDelta,
	}, nil
}

func (t *racingtransformClient) GetName() string {
	return "RacingTransform"
}
