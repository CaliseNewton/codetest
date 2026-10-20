package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CaliseNewton/codetest/model"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/protobuf/encoding/protojson"
)

// Top level fields added next to the protojson event so it can be searched with proper types.
// protojson writes int64 as strings and omits false bools, which makes range and "not hidden" queries unreliable.
const (
	searchStartTime     = "startTime"     // BSON date
	searchBettingStatus = "bettingStatus" // enum name e.g. "BettingOpen"
	searchHidden        = "hidden"        // always present, unlike protojson which omits false
)

type mongoRepo struct {
	client *mongo.Client
	events *mongo.Collection
}

// NewMongoRepository creates a new instance of a Repository using MongoDB as the persistance layer
func NewMongoRepository(ctx context.Context, uri string, database string) (Repository, error) {
	// Connect to Mongo
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		logrus.Errorf("could not create MongoDB client: %v", err)
		return nil, err
	}

	// Check connection
	rslt := &mongoRepo{client: client, events: client.Database(database).Collection("events")}
	if !rslt.HealthCheck(ctx) {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("failed_to_init_mongo")
	}

	return rslt, nil
}

func (c *mongoRepo) HealthCheck(ctx context.Context) bool {
	if err := c.client.Ping(ctx, nil); err != nil {
		logrus.Errorf("could not connect to MongoDB: %v", err)
		return false
	}

	return true
}

// UpdateEvent upserts the event as its protojson document, keyed by the event ID, with the search fields added
func (c *mongoRepo) UpdateEvent(ctx context.Context, event *model.Event) error {
	data, mErr := protojson.Marshal(event)
	if mErr != nil {
		logrus.Errorf("could not marshall event %v", mErr)
		return mErr
	}

	var doc bson.M
	if err := bson.UnmarshalExtJSON(data, false, &doc); err != nil {
		logrus.Errorf("could not convert event to document %v", err)
		return err
	}
	doc["_id"] = event.ID
	if event.GetStartTime() != nil {
		doc[searchStartTime] = time.Unix(0, event.GetStartTime().GetValue()).UTC()
	}
	doc[searchBettingStatus] = event.GetBettingStatus().GetValue().String()
	doc[searchHidden] = event.GetHidden()

	_, err := c.events.ReplaceOne(ctx, bson.M{"_id": event.ID}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		logrus.Errorf("could not update event %v", err)
		return err
	}

	return nil
}

func (c *mongoRepo) GetEventByID(ctx context.Context, id string) (*model.Event, error) {
	var doc bson.M
	err := c.events.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		logrus.Infof("Event not found")
		return nil, nil
	} else if err != nil {
		logrus.Errorf("could not get event %v", err)
		return nil, err
	}

	// _id and the search fields are not part of the Event model
	for _, key := range []string{"_id", searchStartTime, searchBettingStatus, searchHidden} {
		delete(doc, key)
	}
	data, err := bson.MarshalExtJSON(doc, false, false)
	if err != nil {
		logrus.Errorf("could not convert document to json %v", err)
		return nil, err
	}

	event := &model.Event{}
	if err := protojson.Unmarshal(data, event); err != nil {
		logrus.Errorf("failed to unmarshal event %v", err)
		return nil, err
	}

	return event, nil
}

func (c *mongoRepo) DeleteEventByID(ctx context.Context, id string) error {
	_, err := c.events.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		logrus.Errorf("could not delete event %v", err)
	}

	return err
}
