// Package service contains the grpc/http server implementation
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/CaliseNewton/codetest/core"
	"github.com/CaliseNewton/codetest/core/repository"
	"github.com/CaliseNewton/codetest/core/transforms"
	"github.com/CaliseNewton/codetest/merger"
	"github.com/CaliseNewton/codetest/model"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Upstreams defines dependencies the service has on other services
type Upstreams struct {
	MergerClient merger.ServiceClient
	Repo         repository.Repository
	Transforms   []transforms.TransformClient
}

// NewService creqtes a new instancxe of Service
func NewService(grpcPort, httpport int, upstreams *Upstreams) *Service {
	return &Service{GRPCPort: grpcPort, HTTPPort: httpport, Upstreams: upstreams}
}

// RegisterGRPCServerImplementations registers the grpc service contract implemented by this server
func (host *Service) RegisterGRPCServerImplementations(grpcServer *grpc.Server) {
	core.RegisterServiceServer(grpcServer, host)
}

// Update updates an Event and runs the pipeline of transformations
func (host *Service) Update(ctx context.Context, req *core.UpdateRequest) (*core.UpdateResponse, error) {
	stripClientClosedAt(req.GetEvent())

	existing, err := host.Upstreams.Repo.GetEventByID(ctx, req.GetEvent().GetID())
	if err != nil {
		logrus.WithError(err).Error("Update: failed to retrieve event")
		return nil, err
	}

	resp := &core.UpdateResponse{Message: "Success"}

	update := req.GetEvent()
	if existing == nil {
		resp.Message = fmt.Sprintf("New Event born %v", req.GetEvent().GetID())
	} else {
		update, err = host.Upstreams.MergerClient.MergeEvent(context.Background(), existing, req.GetEvent())
		if err != nil {
			logrus.WithError(err).Error("Update: failed to merge event")
			return nil, err
		}
	}

	for _, t := range host.Upstreams.Transforms {
		upd, tErr := t.TransformEvent(ctx, req.Event, update)
		if tErr != nil {
			logrus.WithError(tErr).Errorf("Update: failed to run transform %v", t.GetName())
		}
		if upd != nil {
			update, err = host.Upstreams.MergerClient.MergeEvent(context.Background(), update, upd)
			if err != nil {
				logrus.WithError(err).Errorf("Update: failed to merge event in transform %v", t.GetName())
				return nil, err
			}
		}
	}

	err = host.Upstreams.Repo.UpdateEvent(ctx, update)
	if err != nil {
		logrus.WithError(err).Error("Update: failed to update event")
		return nil, err
	}

	return resp, nil
}

// stripClientClosedAt removes any ClosedAt sent in a request. ClosedAt is an audit field that only markettransform
// sets, from the market's BettingStatus, so a client supplied value is ignored rather than trusted.
func stripClientClosedAt(event *model.Event) {
	for _, market := range event.GetMarkets() {
		market.ClosedAt = nil
	}
}

// GetSportEvent retrieves a model.Event from the database and returns a core.SportEvent - this is a more UserConsumable representation of the model that is specific to sport events
func (host *Service) GetSportEvent(ctx context.Context, req *core.GetSportEventRequest) (*core.GetSportEventResponse, error) {
	existing, err := host.Upstreams.Repo.GetEventByID(ctx, req.GetEventID())
	if err != nil {
		logrus.WithError(err).Error("GetSportEvent: failed to retrieve event")
		return nil, err
	}

	resp := &core.GetSportEventResponse{}

	if existing == nil {
		return resp, nil
	}

	rslt := &core.SportEvent{}
	rslt.ConvertFromModel(existing)
	resp.Event = rslt

	return resp, nil
}

// GetRacingEvent retrieves a model.Event from the database and returns a core.RacingEvent - this is a more UserConsumable representation of the model that is specific to racing events
func (host *Service) GetRacingEvent(ctx context.Context, req *core.GetRacingEventRequest) (*core.GetRacingEventResponse, error) {
	existing, err := host.Upstreams.Repo.GetEventByID(ctx, req.GetEventID())
	if err != nil {
		logrus.WithError(err).Error("GetRacingEvent: failed to retrieve event")
		return nil, err
	}

	resp := &core.GetRacingEventResponse{}

	// Missing events and events that are not races both return an empty response, matching GetSportEvent
	if existing == nil || existing.GetRacingData() == nil {
		return resp, nil
	}

	rslt := &core.RacingEvent{}
	rslt.ConvertFromModel(existing)
	resp.Event = rslt

	return resp, nil
}

// SearchEvents returns every event matching all of the supplied criteria (date range and/or betting status and/or display)
func (host *Service) SearchEvents(ctx context.Context, req *core.SearchEventsRequest) (*core.SearchEventsResponse, error) {
	search, err := searchFromRequest(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	events, err := host.Upstreams.Repo.SearchEvents(ctx, search)
	if err != nil {
		logrus.WithError(err).Error("SearchEvents: failed to search events")
		return nil, err
	}

	return &core.SearchEventsResponse{Events: events}, nil
}

// searchFromRequest validates a SearchEventsRequest and converts it to repository search criteria
func searchFromRequest(req *core.SearchEventsRequest) (repository.EventSearch, error) {
	search := repository.EventSearch{}

	if req.GetStartFrom() == "" && req.GetStartTo() == "" && !req.HasBettingStatus() && !req.HasHidden() {
		return search, fmt.Errorf("at least one search criteria is required")
	}

	if req.GetStartFrom() != "" {
		from, err := time.Parse(time.RFC3339, req.GetStartFrom())
		if err != nil {
			return search, fmt.Errorf("StartFrom must be an RFC3339 time e.g. 2025-09-19T00:00:00Z: %v", err)
		}
		fromNanos := from.UnixNano()
		search.StartFrom = &fromNanos
	}

	if req.GetStartTo() != "" {
		to, err := time.Parse(time.RFC3339, req.GetStartTo())
		if err != nil {
			return search, fmt.Errorf("StartTo must be an RFC3339 time e.g. 2025-09-20T00:00:00Z: %v", err)
		}
		toNanos := to.UnixNano()
		search.StartTo = &toNanos
	}

	if search.StartFrom != nil && search.StartTo != nil && *search.StartFrom >= *search.StartTo {
		return search, fmt.Errorf("StartFrom must be before StartTo")
	}

	if req.HasBettingStatus() {
		// BettingUnknown is the zero value, which is never stored (omitempty), so it can't be matched
		if req.GetBettingStatus() == model.BettingStatus_BettingUnknown {
			return search, fmt.Errorf("BettingStatus BettingUnknown cannot be searched")
		}
		bettingStatus := req.GetBettingStatus()
		search.BettingStatus = &bettingStatus
	}

	if req.HasHidden() {
		hidden := req.GetHidden()
		search.Hidden = &hidden
	}

	return search, nil
}
