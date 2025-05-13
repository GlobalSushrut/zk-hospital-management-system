package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

const (
	// MinKeySize is the minimum RSA key size allowed (in bits)
	MinKeySize = 2048

	// DefaultKeyRotationPeriod is the default period for rotating keys
	DefaultKeyRotationPeriod = 90 * 24 * time.Hour // 90 days
	
	// MaxFailedAttempts is the maximum number of failed attempts before lockout
	MaxFailedAttempts = 5
	
	// LockoutDuration is the duration of account lockout after max failed attempts
	LockoutDuration = 15 * time.Minute
)

// SecurityManager provides advanced security features
type SecurityManager struct {
	keyManager       *KeyManager
	sidechannelGuard *SideChannelGuard
	rateLimit        *RateLimiter
	failedAttempts   map[string]int
	lockoutTimes     map[string]time.Time
	mutex            sync.RWMutex
}

// NewSecurityManager creates a new security manager
func NewSecurityManager() *SecurityManager {
	return &SecurityManager{
		keyManager:       NewKeyManager(DefaultKeyRotationPeriod),
		sidechannelGuard: NewSideChannelGuard(),
		rateLimit:        NewRateLimiter(100, 10*time.Second), // 100 requests per 10 seconds
		failedAttempts:   make(map[string]int),
		lockoutTimes:     make(map[string]time.Time),
	}
}

// GetKeyManager returns the internal key manager
func (sm *SecurityManager) GetKeyManager() *KeyManager {
	return sm.keyManager
}

// KeyManager handles cryptographic keys with secure rotation
type KeyManager struct {
	activeKey       *rsa.PrivateKey
	previousKeys    []*rsa.PrivateKey
	keyID           string
	rotationPeriod  time.Duration
	lastRotation    time.Time
	mutex           sync.RWMutex
	stopRotation    chan struct{}
	keyEncryptedPEM []byte
}

// NewKeyManager creates a new key manager with automatic rotation
func NewKeyManager(rotationPeriod time.Duration) *KeyManager {
	km := &KeyManager{
		rotationPeriod: rotationPeriod,
		previousKeys:   make([]*rsa.PrivateKey, 0, 3), // Keep last 3 keys
		stopRotation:   make(chan struct{}),
	}
	
	// Generate initial key
	if err := km.rotateKey(); err != nil {
		log.Printf("Failed to generate initial key: %v", err)
	}
	
	return km
}

// StartAutoRotation begins automatic key rotation
func (km *KeyManager) StartAutoRotation() {
	go func() {
		ticker := time.NewTicker(km.rotationPeriod)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				if err := km.rotateKey(); err != nil {
					log.Printf("Key rotation failed: %v", err)
				}
			case <-km.stopRotation:
				return
			}
		}
	}()
}

// StopAutoRotation stops automatic key rotation
func (km *KeyManager) StopAutoRotation() {
	close(km.stopRotation)
}

// SetRotationPeriod updates the key rotation period
func (km *KeyManager) SetRotationPeriod(period time.Duration) {
	km.mutex.Lock()
	defer km.mutex.Unlock()
	km.rotationPeriod = period
}

// rotateKey generates a new key and retires the old one
func (km *KeyManager) rotateKey() error {
	// Generate new RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, MinKeySize)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}
	
	km.mutex.Lock()
	defer km.mutex.Unlock()
	
	// Move current key to previous keys
	if km.activeKey != nil {
		km.previousKeys = append(km.previousKeys, km.activeKey)
		
		// Keep only the last 3 keys
		if len(km.previousKeys) > 3 {
			km.previousKeys = km.previousKeys[len(km.previousKeys)-3:]
		}
	}
	
	// Set new active key
	km.activeKey = privateKey
	km.lastRotation = time.Now()
	
	// Generate a unique key ID
	h := sha256.New()
	h.Write(x509.MarshalPKCS1PublicKey(&privateKey.PublicKey))
	km.keyID = fmt.Sprintf("%x", h.Sum(nil)[:8])
	
	return nil
}

// GetActivePublicKey returns the current active public key in PEM format
func (km *KeyManager) GetActivePublicKey() (string, string, error) {
	km.mutex.RLock()
	defer km.mutex.RUnlock()
	
	if km.activeKey == nil {
		return "", "", errors.New("no active key available")
	}
	
	// Marshal the public key to PKCS1 DER
	pubKeyDER := x509.MarshalPKCS1PublicKey(&km.activeKey.PublicKey)
	
	// PEM encode the DER data
	pubKeyBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKeyDER,
	}
	
	pubKeyPEM := pem.EncodeToMemory(pubKeyBlock)
	
	return string(pubKeyPEM), km.keyID, nil
}

// Encrypt data with the active public key
func (km *KeyManager) Encrypt(data []byte) ([]byte, string, error) {
	km.mutex.RLock()
	defer km.mutex.RUnlock()
	
	if km.activeKey == nil {
		return nil, "", errors.New("no active key available")
	}
	
	// Use RSA-OAEP for encryption
	encrypted, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		&km.activeKey.PublicKey,
		data,
		nil,
	)
	if err != nil {
		return nil, "", fmt.Errorf("encryption failed: %w", err)
	}
	
	return encrypted, km.keyID, nil
}

// Decrypt data with the specified key ID
func (km *KeyManager) Decrypt(data []byte, keyID string) ([]byte, error) {
	km.mutex.RLock()
	defer km.mutex.RUnlock()
	
	// Check active key first
	if km.keyID == keyID {
		decrypted, err := rsa.DecryptOAEP(
			sha256.New(),
			rand.Reader,
			km.activeKey,
			data,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
		return decrypted, nil
	}
	
	// Check previous keys
	for _, prevKey := range km.previousKeys {
		// Generate key ID for comparison
		h := sha256.New()
		h.Write(x509.MarshalPKCS1PublicKey(&prevKey.PublicKey))
		prevKeyID := fmt.Sprintf("%x", h.Sum(nil)[:8])
		
		if prevKeyID == keyID {
			decrypted, err := rsa.DecryptOAEP(
				sha256.New(),
				rand.Reader,
				prevKey,
				data,
				nil,
			)
			if err != nil {
				return nil, fmt.Errorf("decryption with previous key failed: %w", err)
			}
			return decrypted, nil
		}
	}
	
	return nil, fmt.Errorf("key with ID %s not found", keyID)
}

// SideChannelGuard protects against side-channel attacks
type SideChannelGuard struct {
	// Options for side-channel mitigation
	constantTimeComparison bool
	preventTimingAttacks   bool
	padResponses           bool
}

// NewSideChannelGuard creates a new side-channel protection instance
func NewSideChannelGuard() *SideChannelGuard {
	return &SideChannelGuard{
		constantTimeComparison: true,
		preventTimingAttacks:   true,
		padResponses:           true,
	}
}

// ConstantTimeCompare compares two byte slices in constant time
func (scg *SideChannelGuard) ConstantTimeCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// ConstantTimeStringCompare compares two strings in constant time
func (scg *SideChannelGuard) ConstantTimeStringCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ObfuscateError returns a generic error to prevent information leakage
func (scg *SideChannelGuard) ObfuscateError(originalErr error) error {
	if originalErr == nil {
		return nil
	}
	
	// For security-sensitive operations, return a generic error
	return errors.New("operation failed")
}

// AddResponsePadding adds random padding to a response to prevent size analysis
func (scg *SideChannelGuard) AddResponsePadding(data []byte) []byte {
	if !scg.padResponses {
		return data
	}
	
	// Add random padding between 16-128 bytes
	paddingSize := 16 + (time.Now().Nanosecond() % 112)
	padding := make([]byte, paddingSize)
	
	// Fill with random data
	if _, err := rand.Read(padding); err != nil {
		// If random fails, just use zeros (less secure but better than nothing)
		padding = make([]byte, paddingSize)
	}
	
	// Add a header with padding info
	header := fmt.Sprintf("X-Padding: %d\r\n", paddingSize)
	
	// Combine the header, original data, and padding
	result := make([]byte, len(header)+len(data)+len(padding))
	copy(result, []byte(header))
	copy(result[len(header):], data)
	copy(result[len(header)+len(data):], padding)
	
	return result
}

// RateLimiter helps prevent brute force and DoS attacks
type RateLimiter struct {
	requestCounts      map[string]int
	windowSize         time.Duration
	maxRequestsPerWindow int
	mutex              sync.Mutex
	cleanupTicker      *time.Ticker
	stopCleanup        chan struct{}
	ipWindows          map[string][]time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, windowSize time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requestCounts:      make(map[string]int),
		ipWindows:          make(map[string][]time.Time),
		maxRequestsPerWindow: maxRequests,
		windowSize:         windowSize,
		stopCleanup:        make(chan struct{}),
	}
	
	// Start the cleanup goroutine
	rl.cleanupTicker = time.NewTicker(windowSize / 2)
	go rl.cleanupOldRequests()
	
	return rl
}

// CleanupOldRequests removes expired request counts
func (rl *RateLimiter) cleanupOldRequests() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mutex.Lock()
			now := time.Now()
			for ip, times := range rl.ipWindows {
				newTimes := make([]time.Time, 0, len(times))
				for _, t := range times {
					if now.Sub(t) < rl.windowSize {
						newTimes = append(newTimes, t)
					}
				}
				if len(newTimes) == 0 {
					delete(rl.ipWindows, ip)
					delete(rl.requestCounts, ip)
				} else {
					rl.ipWindows[ip] = newTimes
					rl.requestCounts[ip] = len(newTimes)
				}
			}
			rl.mutex.Unlock()
		case <-rl.stopCleanup:
			rl.cleanupTicker.Stop()
			return
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	
	// Initialize if not exists
	if _, exists := rl.ipWindows[ip]; !exists {
		rl.ipWindows[ip] = make([]time.Time, 0, rl.maxRequestsPerWindow)
	}
	
	// Filter out old timestamps
	newTimes := make([]time.Time, 0, len(rl.ipWindows[ip]))
	for _, t := range rl.ipWindows[ip] {
		if now.Sub(t) < rl.windowSize {
			newTimes = append(newTimes, t)
		}
	}
	
	// Check if we're over the limit
	if len(newTimes) >= rl.maxRequestsPerWindow {
		return false
	}
	
	// Add current request time
	newTimes = append(newTimes, now)
	rl.ipWindows[ip] = newTimes
	rl.requestCounts[ip] = len(newTimes)
	
	return true
}

// GetRemainingRequests returns how many more requests an IP can make
func (rl *RateLimiter) GetRemainingRequests(ip string) int {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	count, exists := rl.requestCounts[ip]
	if !exists {
		return rl.maxRequestsPerWindow
	}
	
	remaining := rl.maxRequestsPerWindow - count
	if remaining < 0 {
		remaining = 0
	}
	
	return remaining
}

// RecordFailedAttempt records a failed authentication attempt
func (sm *SecurityManager) RecordFailedAttempt(userID string) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	// Check if account is locked
	lockTime, locked := sm.lockoutTimes[userID]
	if locked && time.Since(lockTime) < LockoutDuration {
		// Account is still locked
		return false
	} else if locked {
		// Lockout period has expired, reset counts
		delete(sm.lockoutTimes, userID)
		delete(sm.failedAttempts, userID)
	}
	
	// Increment failed attempts
	sm.failedAttempts[userID]++
	
	// Check if we need to lock the account
	if sm.failedAttempts[userID] >= MaxFailedAttempts {
		sm.lockoutTimes[userID] = time.Now()
		return false
	}
	
	return true
}

// ResetFailedAttempts resets the failed attempt counter for a user
func (sm *SecurityManager) ResetFailedAttempts(userID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	delete(sm.failedAttempts, userID)
	delete(sm.lockoutTimes, userID)
}

// IsAccountLocked checks if an account is currently locked out
func (sm *SecurityManager) IsAccountLocked(userID string) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	lockTime, locked := sm.lockoutTimes[userID]
	if !locked {
		return false
	}
	
	// Check if lockout period has expired
	if time.Since(lockTime) >= LockoutDuration {
		return false
	}
	
	return true
}

// RateLimitAllowed checks if a request from the given IP should be allowed
func (sm *SecurityManager) RateLimitAllowed(ip string) bool {
	return sm.rateLimit.Allow(ip)
}

// GenerateSecureToken generates a cryptographically secure random token
func (sm *SecurityManager) GenerateSecureToken(length int) (string, error) {
	if length < 16 {
		length = 16 // Enforce minimum token length
	}
	
	// Create a byte slice to hold random bytes
	b := make([]byte, length)
	
	// Fill with random data
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	
	// Convert to base64 to make it URL-safe
	return base64.URLEncoding.EncodeToString(b), nil
}

// ConstantTimeCompare compares two strings in constant time to prevent timing attacks
func (sm *SecurityManager) ConstantTimeCompare(a, b string) bool {
	return sm.sidechannelGuard.ConstantTimeStringCompare(a, b)
}

// ObfuscateError returns a generic error to prevent information leakage
func (sm *SecurityManager) ObfuscateError(originalErr error) error {
	return sm.sidechannelGuard.ObfuscateError(originalErr)
}

// AddResponsePadding adds random padding to prevent size analysis
func (sm *SecurityManager) AddResponsePadding(data []byte) []byte {
	return sm.sidechannelGuard.AddResponsePadding(data)
}

// Encrypt data with the active public key
func (sm *SecurityManager) Encrypt(data []byte) ([]byte, string, error) {
	return sm.keyManager.Encrypt(data)
}

// Decrypt data with the specified key ID
func (sm *SecurityManager) Decrypt(data []byte, keyID string) ([]byte, error) {
	return sm.keyManager.Decrypt(data, keyID)
}

// GetActivePublicKey returns the current active public key in PEM format
func (sm *SecurityManager) GetActivePublicKey() (string, string, error) {
	return sm.keyManager.GetActivePublicKey()
}

// StartKeyRotation begins automatic key rotation
func (sm *SecurityManager) StartKeyRotation() {
	sm.keyManager.StartAutoRotation()
}

// StopKeyRotation stops automatic key rotation
func (sm *SecurityManager) StopKeyRotation() {
	sm.keyManager.StopAutoRotation()
}
