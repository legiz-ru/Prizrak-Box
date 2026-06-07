package internal

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"github.com/metacubex/age"
	"github.com/metacubex/age/armor"
)

const ageArmorHeader = "-----BEGIN AGE ENCRYPTED FILE-----"
const ageBinaryHeader = "age-encryption.org/v1"

// IsAgeEncrypted detects age-encrypted content (armored or binary).
func IsAgeEncrypted(content string) bool {
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, ageArmorHeader) || strings.HasPrefix(trimmed, ageBinaryHeader)
}

// DecryptAge decrypts age-encrypted content using the provided secret key.
// Supports both armored (PEM-like) and binary age format.
// secretKey must be in bech32 format: AGE-SECRET-KEY-1... or AGE-SECRET-KEY-HYBRID-1...
func DecryptAge(encryptedContent, secretKey string) (string, error) {
	identity, err := parseAgeIdentity(secretKey)
	if err != nil {
		return "", errors.New("invalid age-secret-key: " + err.Error())
	}

	src := strings.NewReader(strings.TrimSpace(encryptedContent))

	var reader io.Reader = src
	if strings.HasPrefix(strings.TrimSpace(encryptedContent), ageArmorHeader) {
		reader = armor.NewReader(src)
	}

	decReader, err := age.Decrypt(reader, identity)
	if err != nil {
		return "", errors.New("age decryption failed: " + err.Error())
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, decReader); err != nil {
		return "", errors.New("age decryption read failed: " + err.Error())
	}

	return buf.String(), nil
}

// GenerateAgeKeyPair generates an age keypair of the given type.
// keyType: "mlkem768-x25519" (default/recommended) or "x25519".
// Returns secretKey and publicKey as bech32 strings.
func GenerateAgeKeyPair(keyType string) (secretKey, publicKey string, err error) {
	if keyType == "x25519" {
		id, e := age.GenerateX25519Identity()
		if e != nil {
			return "", "", e
		}
		return id.String(), id.Recipient().String(), nil
	}

	// Default: MLKEM768-X25519 hybrid (post-quantum)
	id, e := age.GenerateHybridIdentity()
	if e != nil {
		return "", "", e
	}
	return id.String(), id.Recipient().String(), nil
}

// parseAgeIdentity parses an age secret key string into an Identity.
func parseAgeIdentity(secretKey string) (age.Identity, error) {
	trimmed := strings.TrimSpace(secretKey)

	// Try hybrid first (AGE-SECRET-KEY-HYBRID-1...)
	if hybridId, err := age.ParseHybridIdentity(trimmed); err == nil {
		return hybridId, nil
	}

	// Fall back to X25519 (AGE-SECRET-KEY-1...)
	if x25519Id, err := age.ParseX25519Identity(trimmed); err == nil {
		return x25519Id, nil
	}

	return nil, errors.New("unrecognized key format")
}
