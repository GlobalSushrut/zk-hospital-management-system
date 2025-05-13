package eventlog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EventStatus represents the status of an event
type EventStatus string

const (
	// StatusPending indicates an event that is in progress
	StatusPending EventStatus = "pending"
	// StatusCompleted indicates a successfully completed event
	StatusCompleted EventStatus = "completed"
	// StatusFailed indicates a failed event
	StatusFailed EventStatus = "failed"
	// StatusRetrying indicates an event being retried
	StatusRetrying EventStatus = "retrying"
)

// Event represents a system event
type Event struct {
	EventID    string      `bson:"event_id" json:"event_id"`
	Type       string      `bson:"type" json:"type"`
	Party      string      `bson:"party" json:"party"`
	Payload    interface{} `bson:"payload" json:"payload"`
	Status     EventStatus `bson:"status" json:"status"`
	Timestamp  time.Time   `bson:"timestamp" json:"timestamp"`
	ResolvedAt *time.Time  `bson:"resolved_at,omitempty" json:"resolved_at,omitempty"`
	Retries    int         `bson:"retries" json:"retries"`
}

// EventLogger handles event logging and tracking
type EventLogger struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

// NewEventLogger creates a new event logger
func NewEventLogger(ctx context.Context, mongoURI string) (*EventLogger, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	db := client.Database("eventlogger")
	collection := db.Collection("events")

	// Create indexes
	indexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "event_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "party", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "timestamp", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "type", Value: 1}},
			Options: options.Index(),
		},
	}

	_, err = collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %v", err)
	}

	return &EventLogger{
		client:     client,
		db:         db,
		collection: collection,
	}, nil
}

// Close closes the MongoDB connection
func (el *EventLogger) Close(ctx context.Context) error {
	return el.client.Disconnect(ctx)
}

// LogEvent logs a new event
func (el *EventLogger) LogEvent(ctx context.Context, eventType, partyID string, payload interface{}) (string, error) {
	// Generate a new UUID
	eventID := uuid.New().String()

	// Create event
	event := Event{
		EventID:   eventID,
		Type:      eventType,
		Party:     partyID,
		Payload:   payload,
		Status:    StatusPending,
		Timestamp: time.Now().UTC(),
		Retries:   0,
	}

	// Insert into MongoDB
	_, err := el.collection.InsertOne(ctx, event)
	if err != nil {
		return "", fmt.Errorf("failed to insert event: %v", err)
	}

	return eventID, nil
}

// ResolveEvent marks an event as resolved
func (el *EventLogger) ResolveEvent(ctx context.Context, eventID string, status EventStatus) error {
	if status != StatusCompleted && status != StatusFailed {
		return errors.New("invalid resolution status, must be completed or failed")
	}

	now := time.Now().UTC()
	filter := bson.M{"event_id": eventID}
	update := bson.M{
		"$set": bson.M{
			"status":      status,
			"resolved_at": now,
		},
	}

	result, err := el.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update event: %v", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("event not found")
	}

	return nil
}

// GetEvent retrieves an event by ID
func (el *EventLogger) GetEvent(ctx context.Context, eventID string) (*Event, error) {
	filter := bson.M{"event_id": eventID}
	var event Event
	err := el.collection.FindOne(ctx, filter).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("event not found")
		}
		return nil, fmt.Errorf("error fetching event: %v", err)
	}

	return &event, nil
}

// GetEventsByParty retrieves all events for a party
func (el *EventLogger) GetEventsByParty(ctx context.Context, partyID string) ([]Event, error) {
	filter := bson.M{"party": partyID}
	
	findOptions := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	
	cursor, err := el.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("error querying events: %v", err)
	}
	defer cursor.Close(ctx)
	
	var events []Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("error decoding events: %v", err)
	}
	
	return events, nil
}

// GetPendingEvents retrieves pending events
func (el *EventLogger) GetPendingEvents(ctx context.Context, maxRetries int) ([]Event, error) {
	filter := bson.M{
		"status":  StatusPending,
		"retries": bson.M{"$lt": maxRetries},
	}
	
	findOptions := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})
	
	cursor, err := el.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("error querying pending events: %v", err)
	}
	defer cursor.Close(ctx)
	
	var events []Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("error decoding events: %v", err)
	}
	
	return events, nil
}

// RetryEvent increments the retry count for an event
func (el *EventLogger) RetryEvent(ctx context.Context, eventID string) error {
	filter := bson.M{"event_id": eventID}
	update := bson.M{
		"$inc": bson.M{"retries": 1},
		"$set": bson.M{"status": StatusRetrying},
	}
	
	result, err := el.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update event retry count: %v", err)
	}
	
	if result.MatchedCount == 0 {
		return errors.New("event not found")
	}
	
	return nil
}
