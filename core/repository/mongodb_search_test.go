package repository

import (
	"context"
	"testing"
	"time"

	"github.com/CaliseNewton/codetest/model"
	"github.com/stretchr/testify/assert"
)

func nanos(rfc3339 string) *int64 {
	t, _ := time.Parse(time.RFC3339, rfc3339)
	n := t.UnixNano()
	return &n
}

func searchEvent(id string, start *int64, status model.BettingStatus, hidden bool) *model.Event {
	event := &model.Event{ID: id, BettingStatus: &model.OptionalBettingStatus{Value: status}, Hidden: hidden}
	if start != nil {
		event.StartTime = &model.OptionalInt64{Value: *start}
	}
	return event
}

func Test_mongoRepo_SearchEvents(t *testing.T) {
	ctx := context.Background()
	// Own database so other events (e.g. from Postman) can't appear in the results
	repo, err := NewMongoRepository(ctx, "mongodb://localhost:27017", "codetest_search_test")
	assert.NoError(t, err)
	events := repo.(*mongoRepo).events
	assert.NoError(t, events.Drop(ctx))
	defer events.Drop(ctx)

	open, closed, suspended := model.BettingStatus_BettingOpen, model.BettingStatus_BettingClosed, model.BettingStatus_BettingSuspended
	hidden, visible := true, false
	for _, e := range []*model.Event{
		searchEvent("e1", nanos("2025-09-19T10:00:00Z"), open, false),
		searchEvent("e2", nanos("2025-09-19T23:00:00Z"), closed, true),
		searchEvent("e3", nanos("2025-09-20T00:00:00Z"), open, true), // exactly on the day boundary
		searchEvent("e4", nil, suspended, false),                     // no StartTime
	} {
		assert.NoError(t, repo.UpdateEvent(ctx, e))
	}

	tests := []struct {
		name    string
		search  EventSearch
		wantIDs []string // in StartTime order
	}{
		{
			name:    "whole day, StartTo is exclusive",
			search:  EventSearch{StartFrom: nanos("2025-09-19T00:00:00Z"), StartTo: nanos("2025-09-20T00:00:00Z")},
			wantIDs: []string{"e1", "e2"},
		},
		{
			name:    "StartFrom is inclusive",
			search:  EventSearch{StartFrom: nanos("2025-09-20T00:00:00Z")},
			wantIDs: []string{"e3"},
		},
		{
			name:    "StartTo only, events without a StartTime are excluded",
			search:  EventSearch{StartTo: nanos("2025-09-19T12:00:00Z")},
			wantIDs: []string{"e1"},
		},
		{
			name:    "betting status",
			search:  EventSearch{BettingStatus: &open},
			wantIDs: []string{"e1", "e3"},
		},
		{
			name:    "hidden",
			search:  EventSearch{Hidden: &hidden},
			wantIDs: []string{"e2", "e3"},
		},
		{
			name:    "visible, Hidden false is not stored but still matches",
			search:  EventSearch{Hidden: &visible},
			wantIDs: []string{"e4", "e1"}, // a missing StartTime sorts first
		},
		{
			name:    "all criteria combined",
			search:  EventSearch{StartFrom: nanos("2025-09-19T00:00:00Z"), StartTo: nanos("2025-09-21T00:00:00Z"), BettingStatus: &open, Hidden: &hidden},
			wantIDs: []string{"e3"},
		},
		{
			name:    "no matches returns an empty slice",
			search:  EventSearch{BettingStatus: &suspended, Hidden: &hidden},
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := repo.SearchEvents(ctx, tt.search)
			assert.NoError(t, err)

			gotIDs := make([]string, 0, len(results))
			for _, event := range results {
				gotIDs = append(gotIDs, event.GetID())
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}
