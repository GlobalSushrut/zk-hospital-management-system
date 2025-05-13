package zkproof

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Identity represents a party's identity record in the system
type Identity struct {
	PartyID    string    `bson:"party_id"`
	Claim      string    `bson:"claim"`
	Timestamp  time.Time `bson:"timestamp"`
	ZKProof    string    `bson:"zk_proof"`
	LastUpdate time.Time `bson:"last_update"`
}

// ZKIdentity manages identity proofs and verification
type ZKIdentity struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

// NewZKIdentity creates a new ZK identity manager
func NewZKIdentity(ctx context.Context, mongoURI string) (*ZKIdentity, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	db := client.Database("zkidentity")
	collection := db.Collection("identities")

	// Create unique index on party_id and claim
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "party_id", Value: 1}, {Key: "claim", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %v", err)
	}

	return &ZKIdentity{
		client:     client,
		db:         db,
		collection: collection,
	}, nil
}

// Close closes the MongoDB connection
func (zk *ZKIdentity) Close(ctx context.Context) error {
	return zk.client.Disconnect(ctx)
}

// GenerateZKProof generates a zero-knowledge proof for an identity and claim
func (zk *ZKIdentity) GenerateZKProof(partyID, claim string, timestamp time.Time) string {
	// Concatenate partyID, claim, and timestamp for the proof
	data := fmt.Sprintf("%s||%s||%d", partyID, claim, timestamp.Unix())
	
	// Generate SHA-256 hash
	hash := sha256.Sum256([]byte(data))
	
	// Convert to hexadecimal string
	return hex.EncodeToString(hash[:])
}

// RegisterIdentity registers a new identity with a claim
func (zk *ZKIdentity) RegisterIdentity(ctx context.Context, partyID, claim string) (string, error) {
	// Check if the identity already exists
	filter := bson.M{"party_id": partyID, "claim": claim}
	var existingIdentity Identity
	err := zk.collection.FindOne(ctx, filter).Decode(&existingIdentity)
	if err == nil {
		// Identity already exists
		return existingIdentity.ZKProof, nil
	} else if err != mongo.ErrNoDocuments {
		// Error occurred during query
		return "", fmt.Errorf("error checking for existing identity: %v", err)
	}

	// Create new identity
	now := time.Now().UTC()
	zkProof := zk.GenerateZKProof(partyID, claim, now)

	identity := Identity{
		PartyID:    partyID,
		Claim:      claim,
		Timestamp:  now,
		ZKProof:    zkProof,
		LastUpdate: now,
	}

	// Insert into MongoDB
	_, err = zk.collection.InsertOne(ctx, identity)
	if err != nil {
		return "", fmt.Errorf("failed to insert identity: %v", err)
	}

	return zkProof, nil
}

// ValidateClaim validates if a claim is valid for a party
func (zk *ZKIdentity) ValidateClaim(ctx context.Context, partyID, claim string) (bool, error) {
	// Fetch the original registration
	filter := bson.M{"party_id": partyID, "claim": claim}
	var identity Identity
	err := zk.collection.FindOne(ctx, filter).Decode(&identity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil // Claim not found
		}
		return false, fmt.Errorf("error fetching identity: %v", err)
	}

	// Re-generate the proof using original timestamp
	zkProof := zk.GenerateZKProof(partyID, claim, identity.Timestamp)

	// Compare with stored proof
	return zkProof == identity.ZKProof, nil
}

// GetIdentityByProof retrieves identity details using a proof
func (zk *ZKIdentity) GetIdentityByProof(ctx context.Context, zkProof string) (*Identity, error) {
	filter := bson.M{"zk_proof": zkProof}
	var identity Identity
	err := zk.collection.FindOne(ctx, filter).Decode(&identity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Proof not found
		}
		return nil, fmt.Errorf("error fetching identity by proof: %v", err)
	}

	return &identity, nil
}

// UpdateIdentity updates an existing identity
func (zk *ZKIdentity) UpdateIdentity(ctx context.Context, partyID, claim string) (string, error) {
	// Check if the identity exists
	filter := bson.M{"party_id": partyID, "claim": claim}
	var existingIdentity Identity
	err := zk.collection.FindOne(ctx, filter).Decode(&existingIdentity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", fmt.Errorf("identity not found")
		}
		return "", fmt.Errorf("error checking for existing identity: %v", err)
	}

	// Generate new proof with updated timestamp
	now := time.Now().UTC()
	zkProof := zk.GenerateZKProof(partyID, claim, now)

	// Update the document
	update := bson.M{
		"$set": bson.M{
			"zk_proof":    zkProof,
			"last_update": now,
		},
	}

	_, err = zk.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return "", fmt.Errorf("failed to update identity: %v", err)
	}

	return zkProof, nil
}

// GetIdentityByPartyID retrieves all identity records for a party ID
func (zk *ZKIdentity) GetIdentityByPartyID(ctx context.Context, partyID string) ([]Identity, error) {
	filter := bson.M{"party_id": partyID}
	
	// Find all records for this party ID
	cursor, err := zk.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error fetching identities: %v", err)
	}
	defer cursor.Close(ctx)
	
	// Decode into slice of Identity
	var identities []Identity
	if err := cursor.All(ctx, &identities); err != nil {
		return nil, fmt.Errorf("error decoding identities: %v", err)
	}
	
	if len(identities) == 0 {
		return nil, nil // No identities found
	}
	
	return identities, nil
}
