package yag

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TreatmentVector represents an actual vs recommended treatment path
type TreatmentVector struct {
	ID               string     `bson:"_id" json:"id"`
	PatientID        string     `bson:"patient_id" json:"patient_id"`
	Symptom          string     `bson:"symptom" json:"symptom"`
	RecommendedPath  []string   `bson:"recommended_path" json:"recommended_path"`
	ActualPath       []string   `bson:"actual_path" json:"actual_path"`
	MisalignmentScore float64   `bson:"misalignment_score" json:"misalignment_score"`
	StartedAt        time.Time  `bson:"started_at" json:"started_at"`
	LastUpdatedAt    time.Time  `bson:"last_updated_at" json:"last_updated_at"`
	CompletedAt      *time.Time `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
	FeedbackNotes    []string   `bson:"feedback_notes,omitempty" json:"feedback_notes,omitempty"`
	Outcome          string     `bson:"outcome,omitempty" json:"outcome,omitempty"`
	OutcomeSuccess   *bool      `bson:"outcome_success,omitempty" json:"outcome_success,omitempty"`
	DoctorID         string     `bson:"doctor_id" json:"doctor_id"`
	Alerts           []Alert    `bson:"alerts,omitempty" json:"alerts,omitempty"`
}

// Alert represents a warning or notification about treatment misalignment
type Alert struct {
	Type         string    `bson:"type" json:"type"`
	Description  string    `bson:"description" json:"description"`
	Severity     int       `bson:"severity" json:"severity"` // 1-5, 5 being most severe
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	ResolvedAt   *time.Time `bson:"resolved_at,omitempty" json:"resolved_at,omitempty"`
	ActionTaken  string    `bson:"action_taken,omitempty" json:"action_taken,omitempty"`
}

// MisalignmentTracker tracks and analyzes treatment path deviations
type MisalignmentTracker struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	yagUpdater *YAGUpdater
}

// NewMisalignmentTracker creates a new misalignment tracker
func NewMisalignmentTracker(ctx context.Context, mongoURI string, yagUpdater *YAGUpdater) (*MisalignmentTracker, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	db := client.Database("yagupdater")
	collection := db.Collection("treatment_vectors")

	// Create indexes
	indexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "patient_id", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "doctor_id", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "symptom", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "misalignment_score", Value: -1}},
			Options: options.Index(),
		},
	}

	_, err = collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %v", err)
	}

	return &MisalignmentTracker{
		client:     client,
		db:         db,
		collection: collection,
		yagUpdater: yagUpdater,
	}, nil
}

// Close closes the MongoDB connection
func (mt *MisalignmentTracker) Close(ctx context.Context) error {
	return mt.client.Disconnect(ctx)
}

// StartTreatmentVector initiates tracking for a new treatment path
func (mt *MisalignmentTracker) StartTreatmentVector(ctx context.Context, patientID, symptom, doctorID string) (string, error) {
	// Get recommended path
	recommendedPath, _, err := mt.yagUpdater.GetRecommendedPath(ctx, symptom)
	if err != nil {
		return "", fmt.Errorf("failed to get recommended path: %v", err)
	}

	if len(recommendedPath) == 0 {
		return "", errors.New("no recommended path available for this symptom")
	}

	now := time.Now().UTC()
	vector := TreatmentVector{
		ID:                patientID + "-" + symptom + "-" + now.Format(time.RFC3339),
		PatientID:         patientID,
		Symptom:           symptom,
		RecommendedPath:   recommendedPath,
		ActualPath:        []string{}, // Empty at start
		MisalignmentScore: 0,
		StartedAt:         now,
		LastUpdatedAt:     now,
		DoctorID:          doctorID,
	}

	// Insert into MongoDB
	_, err = mt.collection.InsertOne(ctx, vector)
	if err != nil {
		return "", fmt.Errorf("failed to insert treatment vector: %v", err)
	}

	return vector.ID, nil
}

// UpdateActualPath adds a step to the actual treatment path
func (mt *MisalignmentTracker) UpdateActualPath(ctx context.Context, vectorID string, step string) error {
	// Find the vector
	var vector TreatmentVector
	filter := bson.M{"_id": vectorID}
	err := mt.collection.FindOne(ctx, filter).Decode(&vector)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("treatment vector not found")
		}
		return fmt.Errorf("error fetching treatment vector: %v", err)
	}

	// Check if treatment is already completed
	if vector.CompletedAt != nil {
		return errors.New("treatment is already completed")
	}

	// Add step to actual path
	now := time.Now().UTC()
	updatedPath := append(vector.ActualPath, step)
	
	// Calculate misalignment score
	score := mt.calculateMisalignment(vector.RecommendedPath, updatedPath)
	
	// Check for alerts
	alerts := vector.Alerts
	
	// Alert if step not in recommended path
	stepInRecommended := false
	for _, recStep := range vector.RecommendedPath {
		if recStep == step {
			stepInRecommended = true
			break
		}
	}
	
	if !stepInRecommended {
		alerts = append(alerts, Alert{
			Type:        "unexpected_step",
			Description: fmt.Sprintf("Step '%s' not in recommended path", step),
			Severity:    3,
			CreatedAt:   now,
		})
	}
	
	// Alert if step order is wrong
	if len(updatedPath) <= len(vector.RecommendedPath) && len(updatedPath) > 0 {
		expectedStep := vector.RecommendedPath[len(updatedPath)-1]
		if step != expectedStep {
			alerts = append(alerts, Alert{
				Type:        "wrong_step_order",
				Description: fmt.Sprintf("Expected step '%s' but got '%s'", expectedStep, step),
				Severity:    4,
				CreatedAt:   now,
			})
		}
	}
	
	// Alert if misalignment score is high
	if score > 0.5 && (len(vector.Alerts) == 0 || vector.MisalignmentScore < 0.5) {
		alerts = append(alerts, Alert{
			Type:        "high_misalignment",
			Description: fmt.Sprintf("Treatment has significant deviation (score: %.2f)", score),
			Severity:    5,
			CreatedAt:   now,
		})
	}

	// Update the vector
	update := bson.M{
		"$set": bson.M{
			"actual_path":        updatedPath,
			"misalignment_score": score,
			"last_updated_at":    now,
			"alerts":             alerts,
		},
	}

	_, err = mt.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update treatment vector: %v", err)
	}

	return nil
}

// CompleteTreatmentVector marks a treatment as completed
func (mt *MisalignmentTracker) CompleteTreatmentVector(ctx context.Context, vectorID string, outcome string, success bool) error {
	now := time.Now().UTC()
	
	// Update the vector
	filter := bson.M{"_id": vectorID}
	update := bson.M{
		"$set": bson.M{
			"completed_at":     now,
			"last_updated_at":  now,
			"outcome":          outcome,
			"outcome_success":  success,
		},
	}

	result, err := mt.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to complete treatment vector: %v", err)
	}
	
	if result.MatchedCount == 0 {
		return errors.New("treatment vector not found")
	}
	
	// If successful, update YAG with the actual path
	if success {
		var vector TreatmentVector
		err := mt.collection.FindOne(ctx, filter).Decode(&vector)
		if err != nil {
			return fmt.Errorf("error fetching completed vector: %v", err)
		}
		
		// Only update YAG if treatment was successful
		err = mt.yagUpdater.UpdatePath(ctx, vector.Symptom, vector.ActualPath, 0.9, vector.DoctorID)
		if err != nil {
			return fmt.Errorf("failed to update YAG with successful path: %v", err)
		}
		
		// Update success rate
		err = mt.yagUpdater.UpdateSuccessRate(ctx, vector.Symptom, vector.ActualPath, 1.0)
		if err != nil {
			return fmt.Errorf("failed to update success rate: %v", err)
		}
	}
	
	return nil
}

// AddFeedback adds feedback notes to a treatment vector
func (mt *MisalignmentTracker) AddFeedback(ctx context.Context, vectorID string, feedback string) error {
	// Update the vector
	filter := bson.M{"_id": vectorID}
	update := bson.M{
		"$push": bson.M{
			"feedback_notes": feedback,
		},
		"$set": bson.M{
			"last_updated_at": time.Now().UTC(),
		},
	}

	result, err := mt.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to add feedback: %v", err)
	}
	
	if result.MatchedCount == 0 {
		return errors.New("treatment vector not found")
	}
	
	return nil
}

// ResolveAlert marks an alert as resolved
func (mt *MisalignmentTracker) ResolveAlert(ctx context.Context, vectorID string, alertIndex int, actionTaken string) error {
	now := time.Now().UTC()
	
	// Find the vector to check alert index
	var vector TreatmentVector
	filter := bson.M{"_id": vectorID}
	err := mt.collection.FindOne(ctx, filter).Decode(&vector)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("treatment vector not found")
		}
		return fmt.Errorf("error fetching treatment vector: %v", err)
	}
	
	if alertIndex < 0 || alertIndex >= len(vector.Alerts) {
		return errors.New("alert index out of bounds")
	}
	
	// Update specific alert
	updatePath := fmt.Sprintf("alerts.%d.resolved_at", alertIndex)
	actionPath := fmt.Sprintf("alerts.%d.action_taken", alertIndex)
	
	update := bson.M{
		"$set": bson.M{
			updatePath:        now,
			actionPath:        actionTaken,
			"last_updated_at": now,
		},
	}
	
	_, err = mt.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to resolve alert: %v", err)
	}
	
	return nil
}

// GetActiveVectorsByPatient gets active treatment vectors for a patient
func (mt *MisalignmentTracker) GetActiveVectorsByPatient(ctx context.Context, patientID string) ([]TreatmentVector, error) {
	filter := bson.M{
		"patient_id":  patientID,
		"completed_at": nil,
	}
	
	cursor, err := mt.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error querying treatment vectors: %v", err)
	}
	defer cursor.Close(ctx)
	
	var vectors []TreatmentVector
	if err := cursor.All(ctx, &vectors); err != nil {
		return nil, fmt.Errorf("error parsing treatment vectors: %v", err)
	}
	
	return vectors, nil
}

// GetMisalignedVectors gets vectors with high misalignment scores
func (mt *MisalignmentTracker) GetMisalignedVectors(ctx context.Context, doctorID string, threshold float64) ([]TreatmentVector, error) {
	filter := bson.M{
		"doctor_id":         doctorID,
		"completed_at":      nil,
		"misalignment_score": bson.M{"$gt": threshold},
	}
	
	// Sort by misalignment score (highest first)
	findOptions := options.Find().SetSort(bson.D{{Key: "misalignment_score", Value: -1}})
	
	cursor, err := mt.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("error querying misaligned vectors: %v", err)
	}
	defer cursor.Close(ctx)
	
	var vectors []TreatmentVector
	if err := cursor.All(ctx, &vectors); err != nil {
		return nil, fmt.Errorf("error parsing misaligned vectors: %v", err)
	}
	
	return vectors, nil
}

// calculateMisalignment calculates the misalignment score between recommended and actual paths
func (mt *MisalignmentTracker) calculateMisalignment(recommended, actual []string) float64 {
	// If either path is empty, return maximum misalignment
	if len(recommended) == 0 || len(actual) == 0 {
		return 1.0
	}
	
	// Calculate Levenshtein (edit) distance
	m := len(recommended)
	n := len(actual)
	
	// Create distance matrix
	d := make([][]int, m+1)
	for i := range d {
		d[i] = make([]int, n+1)
	}
	
	// Initialize first row and column
	for i := 0; i <= m; i++ {
		d[i][0] = i
	}
	for j := 0; j <= n; j++ {
		d[0][j] = j
	}
	
	// Calculate edit distance
	for j := 1; j <= n; j++ {
		for i := 1; i <= m; i++ {
			if recommended[i-1] == actual[j-1] {
				d[i][j] = d[i-1][j-1] // No operation
			} else {
				// Minimum of:
				// - Deletion (d[i-1][j] + 1)
				// - Insertion (d[i][j-1] + 1)
				// - Substitution (d[i-1][j-1] + 1)
				d[i][j] = min(d[i-1][j]+1, min(d[i][j-1]+1, d[i-1][j-1]+1))
			}
		}
	}
	
	// Normalize the distance to [0,1]
	maxLen := math.Max(float64(m), float64(n))
	if maxLen == 0 {
		return 0.0
	}
	
	normalizedDistance := float64(d[m][n]) / maxLen
	return normalizedDistance
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
