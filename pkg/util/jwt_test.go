package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-jwt-testing"

func TestGenerateTokenPair(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		email          string
		role           string
		secret         string
		accessExpiry   time.Duration
		refreshExpiry  time.Duration
		wantErr        bool
	}{
		{
			name:          "Valid token generation",
			userID:        1,
			email:         "test@example.com",
			role:          "user",
			secret:        testSecret,
			accessExpiry:  15 * time.Minute,
			refreshExpiry: 7 * 24 * time.Hour,
			wantErr:       false,
		},
		{
			name:          "With admin role",
			userID:        2,
			email:         "admin@example.com",
			role:          "admin",
			secret:        testSecret,
			accessExpiry:  15 * time.Minute,
			refreshExpiry: 7 * 24 * time.Hour,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := GenerateTokenPair(
				tt.userID,
				tt.email,
				tt.role,
				tt.secret,
				tt.accessExpiry,
				tt.refreshExpiry,
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tokens)
			} else {
				require.NoError(t, err)
				require.NotNil(t, tokens)
				assert.NotEmpty(t, tokens.AccessToken)
				assert.NotEmpty(t, tokens.RefreshToken)
				assert.NotEqual(t, tokens.AccessToken, tokens.RefreshToken)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	userID := uint(123)
	email := "test@example.com"
	role := "user"

	// Generate a valid token
	tokens, err := GenerateTokenPair(
		userID,
		email,
		role,
		testSecret,
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		secret  string
		wantErr error
	}{
		{
			name:    "Valid access token",
			token:   tokens.AccessToken,
			secret:  testSecret,
			wantErr: nil,
		},
		{
			name:    "Valid refresh token",
			token:   tokens.RefreshToken,
			secret:  testSecret,
			wantErr: nil,
		},
		{
			name:    "Invalid secret",
			token:   tokens.AccessToken,
			secret:  "wrong-secret",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "Invalid token format",
			token:   "invalid.token.format",
			secret:  testSecret,
			wantErr: ErrInvalidToken,
		},
		{
			name:    "Empty token",
			token:   "",
			secret:  testSecret,
			wantErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateToken(tt.token, tt.secret)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, claims)
			} else {
				require.NoError(t, err)
				require.NotNil(t, claims)
				assert.Equal(t, userID, claims.UserID)
				assert.Equal(t, email, claims.Email)
				assert.Equal(t, role, claims.Role)
			}
		})
	}
}

func TestExpiredToken(t *testing.T) {
	userID := uint(1)
	email := "test@example.com"
	role := "user"

	// Generate token with very short expiry
	tokens, err := GenerateTokenPair(
		userID,
		email,
		role,
		testSecret,
		1*time.Nanosecond, // Very short expiry
		1*time.Nanosecond,
	)
	require.NoError(t, err)

	// Wait a bit to ensure token expires
	time.Sleep(10 * time.Millisecond)

	// Try to validate expired token
	claims, err := ValidateToken(tokens.AccessToken, testSecret)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrExpiredToken)
	assert.Nil(t, claims)
}

func TestTokenClaims(t *testing.T) {
	userID := uint(42)
	email := "user@example.com"
	role := "admin"

	tokens, err := GenerateTokenPair(
		userID,
		email,
		role,
		testSecret,
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)

	claims, err := ValidateToken(tokens.AccessToken, testSecret)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.True(t, claims.IssuedAt.Before(claims.ExpiresAt.Time))
}

func TestDifferentSecrets(t *testing.T) {
	tokens, err := GenerateTokenPair(
		1,
		"test@example.com",
		"user",
		"secret1",
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)

	// Try to validate with different secret
	claims, err := ValidateToken(tokens.AccessToken, "secret2")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Nil(t, claims)
}
