package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type tokenPayload struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
}

func SignToken(userID string, secret []byte, ttl time.Duration) (string, error) {
	if userID == "" {
		return "", errors.New("empty user id")
	}
	if len(secret) == 0 {
		return "", errors.New("empty secret")
	}
	now := time.Now().Unix()
	payload := tokenPayload{
		Sub: userID,
		Exp: time.Now().Add(ttl).Unix(),
		Iat: now,
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(rawPayload)
	signature := sign([]byte(encodedPayload), secret)

	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func VerifyToken(token string, secret []byte) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("empty secret")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid token format")
	}

	rawSignature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("invalid token signature encoding")
	}

	expected := sign([]byte(parts[0]), secret)
	if !hmac.Equal(rawSignature, expected) {
		return "", errors.New("invalid token signature")
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("invalid token payload encoding")
	}

	var payload tokenPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return "", errors.New("invalid token payload")
	}
	if payload.Sub == "" {
		return "", errors.New("empty token subject")
	}
	if payload.Exp <= time.Now().Unix() {
		return "", errors.New("token expired")
	}

	return payload.Sub, nil
}

func sign(data, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	return mac.Sum(nil)
}
