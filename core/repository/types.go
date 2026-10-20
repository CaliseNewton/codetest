package repository

import (
	"context"

	"github.com/CaliseNewton/codetest/model"
)

// EventSearch holds the optional search criteria. A nil field means "don't filter on this"
type EventSearch struct {
	StartFrom     *int64 // unix nanos, inclusive
	StartTo       *int64 // unix nanos, exclusive
	BettingStatus *model.BettingStatus
	Hidden        *bool
}

// Repository is an interface for something that can Retrieve, Update and remove Events from a persistance layer
type Repository interface {
	HealthCheck(ctx context.Context) bool
	GetEventByID(ctx context.Context, id string) (*model.Event, error)
	UpdateEvent(ctx context.Context, event *model.Event) error
	DeleteEventByID(ctx context.Context, id string) error
	SearchEvents(ctx context.Context, search EventSearch) ([]*model.Event, error)
}
