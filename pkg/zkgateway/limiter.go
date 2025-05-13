package zkgateway

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// RateLimitRule defines a rate limiting rule for an API endpoint
type RateLimitRule struct {
	Endpoint          string   `bson:"endpoint" json:"endpoint"`           // API endpoint pattern
	RequestsPerMinute int      `bson:"requests_per_minute" json:"requests_per_minute"`
	RequestsPerHour   int      `bson:"requests_per_hour" json:"requests_per_hour"`
	RequestsPerDay    int      `bson:"requests_per_day" json:"requests_per_day"`
	AppliesTo         []string `bson:"applies_to" json:"applies_to"`       // List of party claims this rule applies to
	BlockDuration     int      `bson:"block_duration" json:"block_duration"` // Blocking duration in minutes
}

// RateLimitEntry tracks rate limit usage for a party and endpoint
type RateLimitEntry struct {
	PartyID           string    `bson:"party_id" json:"party_id"`
	Endpoint          string    `bson:"endpoint" json:"endpoint"`
	MinuteCount       int       `bson:"minute_count" json:"minute_count"`
	HourCount         int       `bson:"hour_count" json:"hour_count"`
	DayCount          int       `bson:"day_count" json:"day_count"`
	LastMinuteReset   time.Time `bson:"last_minute_reset" json:"last_minute_reset"`
	LastHourReset     time.Time `bson:"last_hour_reset" json:"last_hour_reset"`
	LastDayReset      time.Time `bson:"last_day_reset" json:"last_day_reset"`
	BlockedUntil      time.Time `bson:"blocked_until" json:"blocked_until"`
}

// RateLimiter manages API rate limiting based on party identity
type RateLimiter struct {
	client        *mongo.Client
	db            *mongo.Database
	rulesColl     *mongo.Collection
	entriesColl   *mongo.Collection
	defaultRules  []RateLimitRule
	cache         map[string]RateLimitEntry // Cache for recently accessed entries
	cacheLock     sync.RWMutex
	cacheExpiry   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(ctx context.Context, mongoURI string) (*RateLimiter, error) {
	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	// Initialize database and collections
	db := client.Database("zkgateway")
	rulesColl := db.Collection("rate_limit_rules")
	entriesColl := db.Collection("rate_limit_entries")

	// Create indexes for the rules collection
	rulesIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "endpoint", Value: 1}},
			Options: options.Index(),
		},
	}

	_, err = rulesColl.Indexes().CreateMany(ctx, rulesIndexes)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule indexes: %v", err)
	}

	// Create indexes for the entries collection
	entriesIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "party_id", Value: 1}, {Key: "endpoint", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "blocked_until", Value: 1}},
			Options: options.Index(),
		},
	}

	_, err = entriesColl.Indexes().CreateMany(ctx, entriesIndexes)
	if err != nil {
		return nil, fmt.Errorf("failed to create entry indexes: %v", err)
	}

	// Create default rules
	defaultRules := []RateLimitRule{
		{
			Endpoint:          "/api/*",
			RequestsPerMinute: 60,
			RequestsPerHour:   300,
			RequestsPerDay:    1000,
			AppliesTo:         []string{"*"}, // Applies to all
			BlockDuration:     15,            // 15 minutes block
		},
		{
			Endpoint:          "/api/auth/*",
			RequestsPerMinute: 10,
			RequestsPerHour:   30,
			RequestsPerDay:    100,
			AppliesTo:         []string{"*"}, // Applies to all
			BlockDuration:     30,            // 30 minutes block
		},
	}

	limiter := &RateLimiter{
		client:       client,
		db:           db,
		rulesColl:    rulesColl,
		entriesColl:  entriesColl,
		defaultRules: defaultRules,
		cache:        make(map[string]RateLimitEntry),
		cacheExpiry:  5 * time.Minute,
	}

	// Insert default rules if none exist
	count, err := rulesColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count rate limit rules: %v", err)
	}

	if count == 0 {
		var documents []interface{}
		for _, rule := range defaultRules {
			documents = append(documents, rule)
		}

		_, err = rulesColl.InsertMany(ctx, documents)
		if err != nil {
			return nil, fmt.Errorf("failed to insert default rate limit rules: %v", err)
		}
	}

	return limiter, nil
}

// Close closes the MongoDB connection
func (rl *RateLimiter) Close(ctx context.Context) error {
	return rl.client.Disconnect(ctx)
}

// AddRule adds a new rate limit rule
func (rl *RateLimiter) AddRule(ctx context.Context, rule RateLimitRule) error {
	// Validate rule
	if rule.Endpoint == "" {
		return errors.New("endpoint cannot be empty")
	}
	if len(rule.AppliesTo) == 0 {
		return errors.New("applies_to cannot be empty")
	}

	// Check if rule already exists
	filter := bson.M{"endpoint": rule.Endpoint}
	update := bson.M{"$set": rule}
	opts := options.Update().SetUpsert(true)

	_, err := rl.rulesColl.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to add rate limit rule: %v", err)
	}

	return nil
}

// DeleteRule deletes a rate limit rule
func (rl *RateLimiter) DeleteRule(ctx context.Context, endpoint string) error {
	filter := bson.M{"endpoint": endpoint}
	result, err := rl.rulesColl.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete rate limit rule: %v", err)
	}

	if result.DeletedCount == 0 {
		return errors.New("rule not found")
	}

	return nil
}

// GetRules gets all rate limit rules
func (rl *RateLimiter) GetRules(ctx context.Context) ([]RateLimitRule, error) {
	cursor, err := rl.rulesColl.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rate limit rules: %v", err)
	}
	defer cursor.Close(ctx)

	var rules []RateLimitRule
	if err := cursor.All(ctx, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse rate limit rules: %v", err)
	}

	return rules, nil
}

// AllowRequest checks if a request is allowed based on rate limits
func (rl *RateLimiter) AllowRequest(ctx context.Context, partyID, endpoint, claim string) (bool, error) {
	// Generate cache key
	cacheKey := fmt.Sprintf("%s:%s", partyID, endpoint)
	
	// Check if party is blocked
	rl.cacheLock.RLock()
	if entry, exists := rl.cache[cacheKey]; exists {
		if time.Now().UTC().Before(entry.BlockedUntil) {
			rl.cacheLock.RUnlock()
			return false, nil
		}
	}
	rl.cacheLock.RUnlock()

	// Find applicable rule
	rules, err := rl.GetRules(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get rate limit rules: %v", err)
	}

	// Find the most specific rule that applies
	var applicableRule *RateLimitRule
	for _, rule := range rules {
		// Check if endpoint pattern matches
		if matchEndpoint(rule.Endpoint, endpoint) {
			// Check if claim applies
			if containsWildcardOrValue(rule.AppliesTo, claim) {
				applicableRule = &rule
				break
			}
		}
	}

	// If no rule found, use default
	if applicableRule == nil {
		for _, rule := range rl.defaultRules {
			if matchEndpoint(rule.Endpoint, endpoint) {
				if containsWildcardOrValue(rule.AppliesTo, claim) {
					applicableRule = &rule
					break
				}
			}
		}
	}

	// If still no rule, allow the request
	if applicableRule == nil {
		return true, nil
	}

	// Get or create rate limit entry
	entry, err := rl.getOrCreateEntry(ctx, partyID, endpoint)
	if err != nil {
		return false, fmt.Errorf("failed to get rate limit entry: %v", err)
	}

	// Check if blocked
	now := time.Now().UTC()
	if now.Before(entry.BlockedUntil) {
		return false, nil
	}

	// Reset counters if needed
	if now.Sub(entry.LastMinuteReset) >= time.Minute {
		entry.MinuteCount = 0
		entry.LastMinuteReset = now
	}
	if now.Sub(entry.LastHourReset) >= time.Hour {
		entry.HourCount = 0
		entry.LastHourReset = now
	}
	if now.Sub(entry.LastDayReset) >= 24*time.Hour {
		entry.DayCount = 0
		entry.LastDayReset = now
	}

	// Increment counters
	entry.MinuteCount++
	entry.HourCount++
	entry.DayCount++

	// Check limits
	blocked := false
	if applicableRule.RequestsPerMinute > 0 && entry.MinuteCount > applicableRule.RequestsPerMinute {
		blocked = true
	}
	if applicableRule.RequestsPerHour > 0 && entry.HourCount > applicableRule.RequestsPerHour {
		blocked = true
	}
	if applicableRule.RequestsPerDay > 0 && entry.DayCount > applicableRule.RequestsPerDay {
		blocked = true
	}

	// Update entry in database
	filter := bson.M{"party_id": partyID, "endpoint": endpoint}
	update := bson.M{
		"$set": bson.M{
			"minute_count":      entry.MinuteCount,
			"hour_count":        entry.HourCount,
			"day_count":         entry.DayCount,
			"last_minute_reset": entry.LastMinuteReset,
			"last_hour_reset":   entry.LastHourReset,
			"last_day_reset":    entry.LastDayReset,
		},
	}

	if blocked {
		blockDuration := time.Duration(applicableRule.BlockDuration) * time.Minute
		entry.BlockedUntil = now.Add(blockDuration)
		update["$set"].(bson.M)["blocked_until"] = entry.BlockedUntil
	}

	_, err = rl.entriesColl.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return !blocked, fmt.Errorf("failed to update rate limit entry: %v", err)
	}

	// Update cache
	rl.cacheLock.Lock()
	rl.cache[cacheKey] = entry
	rl.cacheLock.Unlock()

	return !blocked, nil
}

// getOrCreateEntry gets an existing rate limit entry or creates a new one
func (rl *RateLimiter) getOrCreateEntry(ctx context.Context, partyID, endpoint string) (RateLimitEntry, error) {
	cacheKey := fmt.Sprintf("%s:%s", partyID, endpoint)
	
	// Check cache first
	rl.cacheLock.RLock()
	if entry, exists := rl.cache[cacheKey]; exists {
		rl.cacheLock.RUnlock()
		return entry, nil
	}
	rl.cacheLock.RUnlock()

	// Query database
	filter := bson.M{"party_id": partyID, "endpoint": endpoint}
	var entry RateLimitEntry
	
	err := rl.entriesColl.FindOne(ctx, filter).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create new entry
			now := time.Now().UTC()
			entry = RateLimitEntry{
				PartyID:          partyID,
				Endpoint:         endpoint,
				MinuteCount:      0,
				HourCount:        0,
				DayCount:         0,
				LastMinuteReset:  now,
				LastHourReset:    now,
				LastDayReset:     now,
				BlockedUntil:     time.Time{}, // Zero time
			}
			return entry, nil
		}
		return RateLimitEntry{}, fmt.Errorf("failed to fetch rate limit entry: %v", err)
	}

	return entry, nil
}

// CleanupEntries removes expired rate limit entries
func (rl *RateLimiter) CleanupEntries(ctx context.Context) (int64, error) {
	// Keep entries that were active in the last 30 days
	threshold := time.Now().UTC().AddDate(0, 0, -30)
	
	filter := bson.M{
		"last_day_reset": bson.M{"$lt": threshold},
		"blocked_until":  bson.M{"$lt": time.Now().UTC()},
	}
	
	result, err := rl.entriesColl.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup rate limit entries: %v", err)
	}
	
	return result.DeletedCount, nil
}

// UnblockParty removes any rate limiting blocks for a party
func (rl *RateLimiter) UnblockParty(ctx context.Context, partyID string) error {
	filter := bson.M{"party_id": partyID}
	update := bson.M{
		"$set": bson.M{
			"blocked_until": time.Time{}, // Zero time
		},
	}
	
	_, err := rl.entriesColl.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to unblock party: %v", err)
	}
	
	// Clear from cache
	rl.cacheLock.Lock()
	for key, entry := range rl.cache {
		if entry.PartyID == partyID {
			entry.BlockedUntil = time.Time{}
			rl.cache[key] = entry
		}
	}
	rl.cacheLock.Unlock()
	
	return nil
}

// Helper functions

// matchEndpoint checks if an endpoint matches a pattern
func matchEndpoint(pattern, endpoint string) bool {
	// TODO: Implement proper pattern matching with wildcards
	// For now, simple wildcard check
	if pattern == "*" || pattern == "/*" || pattern == "/api/*" {
		return true
	}
	
	// Exact match
	if pattern == endpoint {
		return true
	}
	
	// Wildcard at the end
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(endpoint) >= len(prefix) && endpoint[:len(prefix)] == prefix
	}
	
	return false
}

// containsWildcardOrValue checks if a slice contains a wildcard or a specific value
func containsWildcardOrValue(slice []string, value string) bool {
	for _, item := range slice {
		if item == "*" || item == value {
			return true
		}
	}
	return false
}
