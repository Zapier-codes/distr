// Package devicefingerprint turns the fingerprint a visitor's browser computed into what the free-tier gate stores.
//
// The browser sends the SHA-256 of a handful of device signals, never the signals. Distr then keys that value with a
// secret salt (HMAC-SHA256) before it is stored or compared, so the stored value alone cannot be matched against
// fingerprints computed elsewhere, and a leaked table does not identify anyone without the salt.
//
// This is a fraud-deterrence heuristic against repeated free claims, not an identity system, and the value is used
// for nothing else.
package devicefingerprint

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
)

// ClientHashLength is the length of the hex digest the browser sends.
const ClientHashLength = 64

var clientHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

var ErrInvalidClientHash = errors.New("the device fingerprint is not a lower case hex SHA-256 digest")

// ValidClientHash reports whether s has the shape of the digest the browser sends.
func ValidClientHash(s string) bool {
	return clientHashPattern.MatchString(s)
}

// Hash keys the browser's digest with the salt and returns the 32 bytes that are stored.
func Hash(salt, clientHash string) ([]byte, error) {
	if salt == "" {
		return nil, errors.New("the device fingerprint salt is empty")
	}
	if !ValidClientHash(clientHash) {
		return nil, ErrInvalidClientHash
	}
	raw, err := hex.DecodeString(clientHash)
	if err != nil {
		return nil, ErrInvalidClientHash
	}
	mac := hmac.New(sha256.New, []byte(salt))
	mac.Write(raw)
	return mac.Sum(nil), nil
}
