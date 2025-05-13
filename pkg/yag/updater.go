package yag

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TreatmentPath represents a medical treatment path for a symptom
type TreatmentPath struct {
	ID            string        `bson:"_id,omitempty" json:"id,omitempty"`
	Symptom       string        `bson:"symptom" json:"symptom"`
	Paths         [][]string    `bson:"paths" json:"paths"`
	Confidence    []float64     `bson:"confidence" json:"confidence"`
	SuccessRates  []float64     `bson:"success_rates" json:"success_rates"`
	LastUpdated   time.Time     `bson:"last_updated" json:"last_updated"`
	Contributors  []string      `bson:"contributors" json:"contributors"`
	Metadata      PathMetadata  `bson:"metadata" json:"metadata"`
}

// PathMetadata contains additional information about treatment paths
type PathMetadata struct {
	TotalUsages          int       `bson:"total_usages" json:"total_usages"`
	AverageTimeToSuccess float64   `bson:"average_time_to_success" json:"average_time_to_success"`
	CommonDeviation      []string  `bson:"common_deviation" json:"common_deviation"`
	LastVerified         time.Time `bson:"last_verified" json:"last_verified"`
	IsCritical           bool      `bson:"is_critical" json:"is_critical"`
}

// YAGUpdater manages treatment paths and AI learning
type YAGUpdater struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

// NewYAGUpdater creates a new YAG updater
func NewYAGUpdater(ctx context.Context, mongoURI string) (*YAGUpdater, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	db := client.Database("yagupdater")
	collection := db.Collection("treatment_paths")

	// Create index on symptom for faster lookup
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "symptom", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %v", err)
	}

	return &YAGUpdater{
		client:     client,
		db:         db,
		collection: collection,
	}, nil
}

// Close closes the MongoDB connection
func (yu *YAGUpdater) Close(ctx context.Context) error {
	return yu.client.Disconnect(ctx)
}

// UpdatePath adds or updates a treatment path for a symptom
func (yu *YAGUpdater) UpdatePath(ctx context.Context, symptom string, path []string, confidence float64, doctor string) error {
	if len(path) == 0 {
		return errors.New("treatment path cannot be empty")
	}

	now := time.Now().UTC()

	// Check if symptom already exists
	var treatmentPath TreatmentPath
	filter := bson.M{"symptom": symptom}
	err := yu.collection.FindOne(ctx, filter).Decode(&treatmentPath)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create new entry
			newPath := TreatmentPath{
				Symptom:      symptom,
				Paths:        [][]string{path},
				Confidence:   []float64{confidence},
				SuccessRates: []float64{0.0}, // Initial success rate
				LastUpdated:  now,
				Contributors: []string{doctor},
				Metadata: PathMetadata{
					TotalUsages:          1,
					AverageTimeToSuccess: 0.0,
					CommonDeviation:      []string{},
					LastVerified:         now,
					IsCritical:           false,
				},
			}

			_, err := yu.collection.InsertOne(ctx, newPath)
			if err != nil {
				return fmt.Errorf("failed to insert new treatment path: %v", err)
			}
			return nil
		}
		return fmt.Errorf("error checking treatment path: %v", err)
	}

	// Path exists, check if this exact path already exists
	pathExists := false
	pathIndex := -1
	for i, existingPath := range treatmentPath.Paths {
		if compareStringSlices(existingPath, path) {
			pathExists = true
			pathIndex = i
			break
		}
	}

	if pathExists {
		// Update existing path's confidence and usage count
		newConfidence := (treatmentPath.Confidence[pathIndex] + confidence) / 2
		update := bson.M{
			"$set": bson.M{
				fmt.Sprintf("confidence.%d", pathIndex): newConfidence,
				"last_updated":                           now,
			},
			"$inc": bson.M{
				"metadata.total_usages": 1,
			},
			"$addToSet": bson.M{
				"contributors": doctor,
			},
		}

		_, err = yu.collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("failed to update existing path: %v", err)
		}
	} else {
		// Add new path
		update := bson.M{
			"$push": bson.M{
				"paths":         path,
				"confidence":    confidence,
				"success_rates": 0.0,
			},
			"$set": bson.M{
				"last_updated": now,
			},
			"$inc": bson.M{
				"metadata.total_usages": 1,
			},
			"$addToSet": bson.M{
				"contributors": doctor,
			},
		}

		_, err = yu.collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("failed to add new path: %v", err)
		}
	}

	return nil
}

// UpdateSuccessRate updates the success rate for a treatment path
func (yu *YAGUpdater) UpdateSuccessRate(ctx context.Context, symptom string, path []string, successRate float64) error {
	// Find the treatment path
	var treatmentPath TreatmentPath
	filter := bson.M{"symptom": symptom}
	err := yu.collection.FindOne(ctx, filter).Decode(&treatmentPath)
	if err != nil {
		return fmt.Errorf("error finding treatment path: %v", err)
	}

	// Find the path index
	pathIndex := -1
	for i, existingPath := range treatmentPath.Paths {
		if compareStringSlices(existingPath, path) {
			pathIndex = i
			break
		}
	}

	if pathIndex == -1 {
		return errors.New("treatment path not found")
	}

	// Update success rate
	update := bson.M{
		"$set": bson.M{
			fmt.Sprintf("success_rates.%d", pathIndex): successRate,
			"last_updated":                             time.Now().UTC(),
		},
	}

	_, err = yu.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update success rate: %v", err)
	}

	return nil
}

// GetPaths retrieves all treatment paths for a symptom
func (yu *YAGUpdater) GetPaths(ctx context.Context, symptom string) (*TreatmentPath, error) {
	var treatmentPath TreatmentPath
	filter := bson.M{"symptom": symptom}
	err := yu.collection.FindOne(ctx, filter).Decode(&treatmentPath)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No paths found
		}
		return nil, fmt.Errorf("error fetching treatment paths: %v", err)
	}

	return &treatmentPath, nil
}

// GetAllSymptoms retrieves all symptoms in the database
func (yu *YAGUpdater) GetAllSymptoms(ctx context.Context) ([]string, error) {
	// Project only the symptom field
	projection := bson.M{"symptom": 1, "_id": 0}
	
	// Find all documents with only the symptom field
	cursor, err := yu.collection.Find(ctx, bson.M{}, options.Find().SetProjection(projection))
	if err != nil {
		return nil, fmt.Errorf("error fetching symptoms: %v", err)
	}
	defer cursor.Close(ctx)
	
	var results []struct {
		Symptom string `bson:"symptom"`
	}
	
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("error parsing symptoms: %v", err)
	}
	
	// Extract symptoms into a string slice
	symptoms := make([]string, len(results))
	for i, result := range results {
		symptoms[i] = result.Symptom
	}
	
	return symptoms, nil
}

// MarkPathCritical marks a treatment path as critical
func (yu *YAGUpdater) MarkPathCritical(ctx context.Context, symptom string, isCritical bool) error {
	filter := bson.M{"symptom": symptom}
	update := bson.M{
		"$set": bson.M{
			"metadata.is_critical":   isCritical,
			"metadata.last_verified": time.Now().UTC(),
		},
	}
	
	result, err := yu.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update path criticality: %v", err)
	}
	
	if result.MatchedCount == 0 {
		return errors.New("symptom not found")
	}
	
	return nil
}

// GetRecommendedPath gets the most successful path for a symptom
func (yu *YAGUpdater) GetRecommendedPath(ctx context.Context, symptom string) ([]string, float64, error) {
	treatmentPath, err := yu.GetPaths(ctx, symptom)
	if err != nil {
		return nil, 0, err
	}
	
	if treatmentPath == nil || len(treatmentPath.Paths) == 0 {
		return nil, 0, errors.New("no paths found for symptom")
	}
	
	// Find path with highest success rate
	bestIndex := 0
	bestRate := treatmentPath.SuccessRates[0]
	
	for i, rate := range treatmentPath.SuccessRates {
		if rate > bestRate {
			bestRate = rate
			bestIndex = i
		}
	}
	
	return treatmentPath.Paths[bestIndex], treatmentPath.Confidence[bestIndex], nil
}

// compareStringSlices checks if two string slices are equal
func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	
	return true
}
