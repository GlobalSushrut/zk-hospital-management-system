package consent

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

// ConsentStatus represents the status of a consent
type ConsentStatus string

const (
	// StatusPending indicates consent not yet given or revoked
	StatusPending ConsentStatus = "pending"
	// StatusApproved indicates approved consent
	StatusApproved ConsentStatus = "approved"
	// StatusRevoked indicates explicitly revoked consent
	StatusRevoked ConsentStatus = "revoked"
	// StatusExpired indicates consent has expired
	StatusExpired ConsentStatus = "expired"
)

// ConsentType represents the type of consent
type ConsentType string

const (
	// TypeViewRecords permission to view medical records
	TypeViewRecords ConsentType = "view_records"
	// TypeShareRecords permission to share records with a third party
	TypeShareRecords ConsentType = "share_records"
	// TypeModifyTreatment permission to modify treatment plan
	TypeModifyTreatment ConsentType = "modify_treatment"
	// TypeFullAccess permission for full access
	TypeFullAccess ConsentType = "full_access"
	// TypeEmergencyAccess permission for emergency access
	TypeEmergencyAccess ConsentType = "emergency_access"
)

// ConsentParty represents a party in the consent approval chain
type ConsentParty struct {
	PartyID  string        `bson:"party_id" json:"party_id"`
	Role     string        `bson:"role" json:"role"`
	Status   ConsentStatus `bson:"status" json:"status"`
	Timestamp time.Time    `bson:"timestamp" json:"timestamp"`
	Reason   string        `bson:"reason,omitempty" json:"reason,omitempty"`
	ZKProof  string        `bson:"zk_proof,omitempty" json:"zk_proof,omitempty"`
}

// Consent represents a multi-party consent agreement
type Consent struct {
	ID            string         `bson:"_id" json:"id"`
	PatientID     string         `bson:"patient_id" json:"patient_id"`
	Type          ConsentType    `bson:"type" json:"type"`
	Description   string         `bson:"description" json:"description"`
	Parties       []ConsentParty `bson:"parties" json:"parties"`
	Status        ConsentStatus  `bson:"status" json:"status"`
	CreatedAt     time.Time      `bson:"created_at" json:"created_at"`
	ExpiresAt     time.Time      `bson:"expires_at" json:"expires_at"`
	LastUpdatedAt time.Time      `bson:"last_updated_at" json:"last_updated_at"`
	Resources     []string       `bson:"resources,omitempty" json:"resources,omitempty"`
	AllParties    bool           `bson:"all_parties" json:"all_parties"`
}

// ConsentManager handles consent operations
type ConsentManager struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

// NewConsentManager creates a new consent manager
func NewConsentManager(ctx context.Context, mongoURI string) (*ConsentManager, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	db := client.Database("consentmanager")
	collection := db.Collection("consents")

	// Create indexes
	indexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "patient_id", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "parties.party_id", Value: 1}},
			Options: options.Index(),
		},
	}

	_, err = collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %v", err)
	}

	return &ConsentManager{
		client:     client,
		db:         db,
		collection: collection,
	}, nil
}

// Close closes the MongoDB connection
func (cm *ConsentManager) Close(ctx context.Context) error {
	return cm.client.Disconnect(ctx)
}

// CreateConsent creates a new multi-party consent request
func (cm *ConsentManager) CreateConsent(ctx context.Context, patientID string, consentType ConsentType, description string, partyIDs []string, roles []string, expiryDays int, allPartiesRequired bool, resources []string) (string, error) {
	if len(partyIDs) != len(roles) {
		return "", errors.New("party IDs and roles must have the same length")
	}

	consentID := uuid.New().String()
	now := time.Now().UTC()
	expiresAt := now.AddDate(0, 0, expiryDays) // Default to X days expiry

	// Create consent parties
	parties := make([]ConsentParty, len(partyIDs))
	for i, partyID := range partyIDs {
		parties[i] = ConsentParty{
			PartyID:   partyID,
			Role:      roles[i],
			Status:    StatusPending,
			Timestamp: now,
		}
	}

	// Create consent document
	consent := Consent{
		ID:            consentID,
		PatientID:     patientID,
		Type:          consentType,
		Description:   description,
		Parties:       parties,
		Status:        StatusPending,
		CreatedAt:     now,
		ExpiresAt:     expiresAt,
		LastUpdatedAt: now,
		Resources:     resources,
		AllParties:    allPartiesRequired,
	}

	// Insert into MongoDB
	_, err := cm.collection.InsertOne(ctx, consent)
	if err != nil {
		return "", fmt.Errorf("failed to insert consent: %v", err)
	}

	return consentID, nil
}

// UpdatePartyConsent updates a party's consent status
func (cm *ConsentManager) UpdatePartyConsent(ctx context.Context, consentID, partyID string, status ConsentStatus, reason string, zkProof string) error {
	// Find the consent
	var consent Consent
	filter := bson.M{"_id": consentID}
	err := cm.collection.FindOne(ctx, filter).Decode(&consent)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("consent not found")
		}
		return fmt.Errorf("error fetching consent: %v", err)
	}

	// Check if consent is expired
	now := time.Now().UTC()
	if now.After(consent.ExpiresAt) {
		return errors.New("consent has expired")
	}

	// Find the party index
	partyIndex := -1
	for i, party := range consent.Parties {
		if party.PartyID == partyID {
			partyIndex = i
			break
		}
	}

	if partyIndex == -1 {
		return errors.New("party not found in consent")
	}

	// Update the party status
	updateFilter := bson.M{"_id": consentID, "parties.party_id": partyID}
	update := bson.M{
		"$set": bson.M{
			"parties.$.status":    status,
			"parties.$.timestamp": now,
			"parties.$.reason":    reason,
			"parties.$.zk_proof":  zkProof,
			"last_updated_at":     now,
		},
	}

	_, err = cm.collection.UpdateOne(ctx, updateFilter, update)
	if err != nil {
		return fmt.Errorf("failed to update party consent: %v", err)
	}

	// Re-fetch the consent to check if all parties have approved
	err = cm.collection.FindOne(ctx, filter).Decode(&consent)
	if err != nil {
		return fmt.Errorf("error re-fetching consent: %v", err)
	}

	// Check overall consent status
	allApproved := true
	anyRejected := false

	for _, party := range consent.Parties {
		if party.Status == StatusRevoked {
			anyRejected = true
			break
		}
		if consent.AllParties && party.Status != StatusApproved {
			allApproved = false
		}
	}

	var newStatus ConsentStatus
	if anyRejected {
		newStatus = StatusRevoked
	} else if allApproved {
		newStatus = StatusApproved
	} else {
		newStatus = StatusPending
	}

	// Update overall consent status if changed
	if consent.Status != newStatus {
		statusUpdate := bson.M{
			"$set": bson.M{
				"status":         newStatus,
				"last_updated_at": now,
			},
		}
		_, err = cm.collection.UpdateOne(ctx, filter, statusUpdate)
		if err != nil {
			return fmt.Errorf("failed to update consent status: %v", err)
		}
	}

	return nil
}

// GetConsent retrieves a consent by ID
func (cm *ConsentManager) GetConsent(ctx context.Context, consentID string) (*Consent, error) {
	filter := bson.M{"_id": consentID}
	var consent Consent
	err := cm.collection.FindOne(ctx, filter).Decode(&consent)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Consent not found
		}
		return nil, fmt.Errorf("error fetching consent: %v", err)
	}

	// Check if consent is expired but not marked as such
	now := time.Now().UTC()
	if now.After(consent.ExpiresAt) && consent.Status != StatusExpired {
		// Update status to expired
		update := bson.M{
			"$set": bson.M{
				"status":         StatusExpired,
				"last_updated_at": now,
			},
		}
		_, err = cm.collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return nil, fmt.Errorf("failed to update expired consent: %v", err)
		}
		
		consent.Status = StatusExpired
	}

	return &consent, nil
}

// GetPatientConsents retrieves all consents for a patient
func (cm *ConsentManager) GetPatientConsents(ctx context.Context, patientID string) ([]Consent, error) {
	filter := bson.M{"patient_id": patientID}
	cursor, err := cm.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error querying consents: %v", err)
	}
	defer cursor.Close(ctx)

	var consents []Consent
	if err := cursor.All(ctx, &consents); err != nil {
		return nil, fmt.Errorf("error parsing consents: %v", err)
	}

	// Check for expired consents
	now := time.Now().UTC()
	for i, consent := range consents {
		if now.After(consent.ExpiresAt) && consent.Status != StatusExpired {
			consents[i].Status = StatusExpired
			
			// Update in database
			update := bson.M{
				"$set": bson.M{
					"status":         StatusExpired,
					"last_updated_at": now,
				},
			}
			_, err = cm.collection.UpdateOne(ctx, bson.M{"_id": consent.ID}, update)
			if err != nil {
				return nil, fmt.Errorf("failed to update expired consent: %v", err)
			}
		}
	}

	return consents, nil
}

// GetActiveConsentsByParty retrieves active consents for a party
func (cm *ConsentManager) GetActiveConsentsByParty(ctx context.Context, partyID string) ([]Consent, error) {
	now := time.Now().UTC()
	filter := bson.M{
		"parties.party_id": partyID,
		"expires_at": bson.M{"$gt": now},
		"status": bson.M{"$in": []string{string(StatusApproved), string(StatusPending)}},
	}
	
	cursor, err := cm.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error querying consents: %v", err)
	}
	defer cursor.Close(ctx)

	var consents []Consent
	if err := cursor.All(ctx, &consents); err != nil {
		return nil, fmt.Errorf("error parsing consents: %v", err)
	}

	return consents, nil
}

// VerifyConsent verifies if a party has consent to access resources
func (cm *ConsentManager) VerifyConsent(ctx context.Context, partyID string, patientID string, resourceID string, requiredType ConsentType) (bool, string, error) {
	now := time.Now().UTC()
	
	// First check for any resource-specific consent
	resourceFilter := bson.M{
		"patient_id": patientID,
		"parties.party_id": partyID,
		"resources": resourceID,
		"type": requiredType,
		"expires_at": bson.M{"$gt": now},
		"status": StatusApproved,
	}
	
	var resourceConsent Consent
	err := cm.collection.FindOne(ctx, resourceFilter).Decode(&resourceConsent)
	if err == nil {
		// Found valid resource-specific consent
		return true, resourceConsent.ID, nil
	} else if err != mongo.ErrNoDocuments {
		return false, "", fmt.Errorf("error checking resource consent: %v", err)
	}
	
	// If no resource-specific consent, check for general type consent
	generalFilter := bson.M{
		"patient_id": patientID,
		"parties.party_id": partyID,
		"type": requiredType,
		"expires_at": bson.M{"$gt": now},
		"status": StatusApproved,
		"resources": bson.M{"$size": 0}, // No specific resources (general consent)
	}
	
	var generalConsent Consent
	err = cm.collection.FindOne(ctx, generalFilter).Decode(&generalConsent)
	if err == nil {
		// Found valid general consent
		return true, generalConsent.ID, nil
	} else if err != mongo.ErrNoDocuments {
		return false, "", fmt.Errorf("error checking general consent: %v", err)
	}
	
	// Finally check for full access consent
	fullAccessFilter := bson.M{
		"patient_id": patientID,
		"parties.party_id": partyID,
		"type": TypeFullAccess,
		"expires_at": bson.M{"$gt": now},
		"status": StatusApproved,
	}
	
	var fullAccessConsent Consent
	err = cm.collection.FindOne(ctx, fullAccessFilter).Decode(&fullAccessConsent)
	if err == nil {
		// Found valid full access consent
		return true, fullAccessConsent.ID, nil
	} else if err != mongo.ErrNoDocuments {
		return false, "", fmt.Errorf("error checking full access consent: %v", err)
	}
	
	// No valid consent found
	return false, "", nil
}

// RevokeAllConsents revokes all consents for a patient
func (cm *ConsentManager) RevokeAllConsents(ctx context.Context, patientID string, reason string) error {
	now := time.Now().UTC()
	
	filter := bson.M{
		"patient_id": patientID, 
		"status": bson.M{"$ne": StatusRevoked},
	}
	
	update := bson.M{
		"$set": bson.M{
			"status": StatusRevoked,
			"last_updated_at": now,
		},
	}
	
	_, err := cm.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to revoke consents: %v", err)
	}
	
	return nil
}

// CheckEmergencyAccess checks if party has emergency access
func (cm *ConsentManager) CheckEmergencyAccess(ctx context.Context, partyID string, patientID string) (bool, error) {
	now := time.Now().UTC()
	
	filter := bson.M{
		"patient_id": patientID,
		"parties.party_id": partyID,
		"type": TypeEmergencyAccess,
		"expires_at": bson.M{"$gt": now},
		"status": StatusApproved,
	}
	
	count, err := cm.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("error checking emergency access: %v", err)
	}
	
	return count > 0, nil
}
