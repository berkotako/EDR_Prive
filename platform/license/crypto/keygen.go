// License Key Generation and Validation using Ed25519 signatures

package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LicensePayload contains the encoded license information
type LicensePayload struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Tier         string    `json:"tier"`
	IssuedAt     int64     `json:"iat"`
	ExpiresAt    int64     `json:"exp,omitempty"`
	MaxAgents    int       `json:"max_agents"`
}

// KeyPair holds Ed25519 public and private keys
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// GenerateKeyPair creates a new Ed25519 key pair for license signing
func GenerateKeyPair() (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

// GenerateLicenseKey creates a signed license key
func GenerateLicenseKey(payload LicensePayload, privateKey ed25519.PrivateKey) (string, error) {
	// Serialize payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Sign the payload
	signature := ed25519.Sign(privateKey, payloadJSON)

	// Encode payload and signature
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	// Format: PRIVE-{version}-{payload}.{signature}
	licenseKey := fmt.Sprintf("PRIVE-V1-%s.%s", payloadB64, signatureB64)

	// Add dashes for readability (every 5 chars after prefix)
	return formatLicenseKey(licenseKey), nil
}

// ValidateLicenseKey verifies the signature and returns the payload
func ValidateLicenseKey(licenseKey string, publicKey ed25519.PublicKey) (*LicensePayload, error) {
	// Remove formatting dashes
	licenseKey = strings.ReplaceAll(licenseKey, "-", "")

	// Extract components
	if !strings.HasPrefix(licenseKey, "PRIVEV1") {
		return nil, fmt.Errorf("invalid license key format")
	}

	parts := strings.SplitN(licenseKey[7:], ".", 2) // Skip "PRIVEV1"
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid license key structure")
	}

	// Decode payload and signature
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding: %w", err)
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Verify signature
	if !ed25519.Verify(publicKey, payloadJSON, signature) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Deserialize payload
	var payload LicensePayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Check expiration
	if payload.ExpiresAt > 0 {
		expiryTime := time.Unix(payload.ExpiresAt, 0)
		if time.Now().After(expiryTime) {
			return nil, fmt.Errorf("license expired on %s", expiryTime.Format("2006-01-02"))
		}
	}

	return &payload, nil
}

// formatLicenseKey adds dashes for readability
// Example: PRIVE-V1-XXXXX-XXXXX-XXXXX-XXXXX...
func formatLicenseKey(key string) string {
	// Keep the PRIVE-V1 prefix intact
	prefix := "PRIVE-V1-"
	body := key[len(prefix):]

	// Split body into chunks of 5 characters
	var chunks []string
	for i := 0; i < len(body); i += 5 {
		end := i + 5
		if end > len(body) {
			end = len(body)
		}
		chunks = append(chunks, body[i:end])
	}

	return prefix + strings.Join(chunks, "-")
}

// ExportPublicKey exports public key as base64 for storage
func ExportPublicKey(publicKey ed25519.PublicKey) string {
	return base64.StdEncoding.EncodeToString(publicKey)
}

// ImportPublicKey imports a base64-encoded public key
func ImportPublicKey(encoded string) (ed25519.PublicKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size")
	}

	return ed25519.PublicKey(decoded), nil
}

// GenerateTrialKey creates a 14-day trial license
func GenerateTrialKey(email string, privateKey ed25519.PrivateKey) (string, error) {
	payload := LicensePayload{
		ID:        fmt.Sprintf("TRIAL-%d", time.Now().Unix()),
		Email:     email,
		Tier:      "professional",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().AddDate(0, 0, 14).Unix(), // 14 days
		MaxAgents: 10,
	}

	return GenerateLicenseKey(payload, privateKey)
}
