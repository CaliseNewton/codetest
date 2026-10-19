// Package sporttransform supplies a sporttransformClient
package sporttransform

import (
	"context"

	"github.com/CaliseNewton/codetest/core/transforms"
	"github.com/CaliseNewton/codetest/model"
)

type sporttransformClient struct{}

// NewSportTransformClient creates a new Sport transform client
func NewSportTransformClient() transforms.TransformClient {
	return &sporttransformClient{}
}

var sportTypeMap = map[string]string{
	"soccer":       "Soccer",
	"rugby_league": "Rugby League",
}

// TransformEvent performs sport specifc transformation on the Event
func (t *sporttransformClient) TransformEvent(_ context.Context, parialUpdate, fullModel *model.Event) (*model.Event, error) {
	var outDelta *model.Event
	if parialUpdate.EventTypeID == nil {
		return outDelta, nil // if the EventTypeID didnt update on this update skip processing the event
	}

	if fullModel.GetSportData().GetName() != nil {
		return outDelta, nil // no need to update the name if the name already set
	}

	sportName := sportTypeMap[fullModel.GetEventTypeID().GetValue()]
	if sportName == "" {
		return outDelta, nil // unknown sport type so dont change anything
	}

	// Carry Hidden forward from the full model, otherwise the merge treats the missing bool as false and resets it
	outDelta = &model.Event{
		ID:        parialUpdate.ID,
		Hidden:    fullModel.GetHidden(),
		SportData: &model.SportEvent{Name: &model.OptionalString{Value: sportName}},
	}

	return outDelta, nil
}

func (t *sporttransformClient) GetName() string {
	return "SportsTransform"
}
