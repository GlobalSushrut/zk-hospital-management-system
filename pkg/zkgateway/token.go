package zkgateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/telemedicine/zkhealth/pkg/zkproof"
)

// ZKToken represents a zero-knowledge token for API authentication
type ZKToken struct {
	TokenID     string    `bson:"token_id" json:"token_id"`
	PartyID     string    `bson:"party_id" json:"party_id"`
	Claim       string    `bson:"claim" json:"claim"`
	ZKProof     string    `bson:"zk_proof" json:"zk_proof"`
	Nonce       string    `bson:"nonce" json:"nonce"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	ExpiresAt   time.Time `bson:"expires_at" json:"expires_at"`
	LastUsedAt  time.Time `bson:"last_used_at" json:"last_used_at"`
	IsRevoked   bool      `bson:"is_revoked" json:"is_revoked"`
}

// TokenGenerator manages ZK token generation and validation
type TokenGenerator struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	zkIdentity *zkproof.ZKIdentity
}

// NewTokenGenerator creates a new token generator
func NewTokenGenerator(ctx context.Context, mongoURI string, zkIdentity *zkproof.ZKIdentity) (*TokenGenerator, error) {
	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	// Initialize database and collection
	db := client.Database("zkgateway")
	collection := db.Collection("tokens")

	// Create indexes for token lookup and expiration
	indexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "party_id", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "zk_proof", Value: 1}},
			Options: options.Index(),
		},
	}

	_, err = collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %v", err)
	}

	// Setup TTL index for automatic token expiration
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create TTL index: %v", err)
	}

	return &TokenGenerator{
		client:     client,
		db:         db,
		collection: collection,
		zkIdentity: zkIdentity,
	}, nil
}

// Close closes the MongoDB connection
func (tg *TokenGenerator) Close(ctx context.Context) error {
	return tg.client.Disconnect(ctx)
}

// GenerateToken generates a new ZK token for the given party and claim
func (tg *TokenGenerator) GenerateToken(ctx context.Context, partyID, claim string, validityHours int) (*ZKToken, error) {
	// Validate that the party has this claim
	isValid, err := tg.zkIdentity.ValidateClaim(ctx, partyID, claim)
	if err != nil {
		return nil, fmt.Errorf("failed to validate claim: %v", err)
	}
	if !isValid {
		return nil, errors.New("invalid claim for party")
	}

	// Get ZK proof
	zkProof, err := tg.zkIdentity.RegisterIdentity(ctx, partyID, claim)
	if err != nil {
		return nil, fmt.Errorf("failed to get ZK proof: %v", err)
	}

	// Generate a nonce
	nonce := generateNonce()

	// Set token validity period
	now := time.Now().UTC()
	if validityHours <= 0 {
		validityHours = 24 // Default to 24 hours
	}
	expiresAt := now.Add(time.Duration(validityHours) * time.Hour)

	// Generate token ID
	tokenID := generateTokenID(partyID, claim, zkProof, nonce)

	// Create token
	token := &ZKToken{
		TokenID:    tokenID,
		PartyID:    partyID,
		Claim:      claim,
		ZKProof:    zkProof,
		Nonce:      nonce,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		LastUsedAt: now,
		IsRevoked:  false,
	}

	// Store token in MongoDB
	_, err = tg.collection.InsertOne(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to store token: %v", err)
	}

	return token, nil
}

// ValidateToken validates a token and updates its last used timestamp
func (tg *TokenGenerator) ValidateToken(ctx context.Context, tokenID string) (bool, *ZKToken, error) {
	// Find token in MongoDB
	var token ZKToken
	filter := bson.M{"token_id": tokenID}
	err := tg.collection.FindOne(ctx, filter).Decode(&token)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil, nil // Token not found
		}
		return false, nil, fmt.Errorf("error fetching token: %v", err)
	}

	// Check if token is expired or revoked
	now := time.Now().UTC()
	if now.After(token.ExpiresAt) || token.IsRevoked {
		return false, &token, nil // Token expired or revoked
	}

	// Verify that the party still has the claim
	isValid, err := tg.zkIdentity.ValidateClaim(ctx, token.PartyID, token.Claim)
	if err != nil {
		return false, &token, fmt.Errorf("failed to validate claim: %v", err)
	}
	if !isValid {
		// Revoke token if claim is no longer valid
		_, err = tg.collection.UpdateOne(
			ctx,
			filter,
			bson.M{"$set": bson.M{"is_revoked": true}},
		)
		return false, &token, nil
	}

	// Update last used timestamp
	_, err = tg.collection.UpdateOne(
		ctx,
		filter,
		bson.M{"$set": bson.M{"last_used_at": now}},
	)
	if err != nil {
		return true, &token, fmt.Errorf("failed to update last used timestamp: %v", err)
	}

	return true, &token, nil
}

// RevokeToken revokes a token
func (tg *TokenGenerator) RevokeToken(ctx context.Context, tokenID string) error {
	filter := bson.M{"token_id": tokenID}
	update := bson.M{"$set": bson.M{"is_revoked": true}}
	
	result, err := tg.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %v", err)
	}
	
	if result.MatchedCount == 0 {
		return errors.New("token not found")
	}
	
	return nil
}

// RevokeAllTokensForParty revokes all tokens for a party
func (tg *TokenGenerator) RevokeAllTokensForParty(ctx context.Context, partyID string) (int64, error) {
	filter := bson.M{"party_id": partyID, "is_revoked": false}
	update := bson.M{"$set": bson.M{"is_revoked": true}}
	
	result, err := tg.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to revoke tokens: %v", err)
	}
	
	return result.ModifiedCount, nil
}

// GetActiveTokensForParty gets all active tokens for a party
func (tg *TokenGenerator) GetActiveTokensForParty(ctx context.Context, partyID string) ([]ZKToken, error) {
	now := time.Now().UTC()
	filter := bson.M{
		"party_id": partyID,
		"is_revoked": false,
		"expires_at": bson.M{"$gt": now},
	}
	
	cursor, err := tg.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error querying tokens: %v", err)
	}
	defer cursor.Close(ctx)
	
	var tokens []ZKToken
	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, fmt.Errorf("error parsing tokens: %v", err)
	}
	
	return tokens, nil
}

// CleanupExpiredTokens removes expired tokens from the database
// Note: With TTL index, this should happen automatically, but this is a manual cleanup
func (tg *TokenGenerator) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	now := time.Now().UTC()
	filter := bson.M{"expires_at": bson.M{"$lt": now}}
	
	result, err := tg.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired tokens: %v", err)
	}
	
	return result.DeletedCount, nil
}

// generateTokenID generates a unique token ID based on party, claim, proof, and nonce
func generateTokenID(partyID, claim, zkProof, nonce string) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%d", partyID, claim, zkProof, nonce, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// generateNonce generates a random nonce
func generateNonce() string {
	data := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // Return first 16 bytes (32 hex chars)
}
