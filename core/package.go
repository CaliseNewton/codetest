// Package core contains the proto definitions of the Core Service
package core

import (
	"time"

	"github.com/CaliseNewton/codetest/model"
)

//go:generate ./gen-proto.sh

// commonEventFields holds the fields shared by every consumer facing event representation
type commonEventFields struct {
	ID            string
	Name          string
	StartTime     string
	BettingStatus string
	Markets       []*model.Market
	Hidden        bool
}

// convertCommonFields extracts the fields shared by SportEvent and RacingEvent from a model.Event
func convertCommonFields(model *model.Event) commonEventFields {
	return commonEventFields{
		ID:            model.ID,
		Name:          model.GetName().GetValue(),
		StartTime:     time.Unix(0, model.StartTime.GetValue()).Format(time.RFC3339),
		BettingStatus: model.GetBettingStatus().GetValue().String(),
		Markets:       model.Markets,
		Hidden:        model.GetHidden(),
	}
}

// ConvertFromModel converts a model.Event to a core.SportEvent
func (to *SportEvent) ConvertFromModel(model *model.Event) {
	common := convertCommonFields(model)
	to.ID = common.ID
	to.Name = common.Name
	to.StartTime = common.StartTime
	to.BettingStatus = common.BettingStatus
	to.Markets = common.Markets
	to.Hidden = common.Hidden
	to.SportTypeID = model.GetEventTypeID().GetValue()
	to.League = model.GetSportData().GetLeague().GetValue()
	to.SportName = model.GetSportData().GetName().GetValue()
	to.Round = model.GetSportData().GetRound().GetValue()
	to.Region = model.GetSportData().GetRegion().GetValue()
}

// ConvertFromModel converts a model.Event to a core.RacingEvent
func (to *RacingEvent) ConvertFromModel(model *model.Event) {
	common := convertCommonFields(model)
	to.ID = common.ID
	to.Name = common.Name
	to.StartTime = common.StartTime
	to.BettingStatus = common.BettingStatus
	to.Markets = common.Markets
	to.Hidden = common.Hidden
	to.RaceType = model.GetRacingData().GetRaceType().GetValue().String()
	to.Venue = model.GetRacingData().GetVenue().GetValue()
	to.RaceNumber = model.GetRacingData().GetRaceNumber().GetValue()
	to.DistanceMetres = model.GetRacingData().GetDistanceMetres().GetValue()
	to.TrackCondition = model.GetRacingData().GetTrackCondition().GetValue()
	to.Weather = model.GetRacingData().GetWeather().GetValue()
	to.Country = model.GetRacingData().GetCountry().GetValue()
	to.RaceClass = model.GetRacingData().GetRaceClass().GetValue()
}
