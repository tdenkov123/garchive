package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateDevToken(secret, issuer, audience, ownerID string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("secret is required")
	}
	now := time.Now()
	claims := Claims{
		OwnerID: ownerID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   ownerID,
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func DevSecretFromString(s string) string {
	if s != "" {
		return s
	}
	sum := sha256.Sum256([]byte("garchive-dev-secret"))
	return hex.EncodeToString(sum[:])
}

func ValidateHMAC(secret, message, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	expected := mac.Sum(nil)
	got, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, got)
}
