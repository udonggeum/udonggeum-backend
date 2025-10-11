package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "Valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "Empty password",
			password: "",
			wantErr:  false, // bcrypt can hash empty strings
		},
		{
			name:     "Long password",
			password: "this-is-a-very-long-password-with-special-chars!@#$%^&*()",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, hash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				assert.NotEqual(t, tt.password, hash)

				// Verify that the hash starts with bcrypt prefix
				assert.Contains(t, hash, "$2a$")
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "mySecurePassword123"
	hash, err := HashPassword(password)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		hashedPassword string
		password       string
		want           bool
	}{
		{
			name:           "Correct password",
			hashedPassword: hash,
			password:       password,
			want:           true,
		},
		{
			name:           "Incorrect password",
			hashedPassword: hash,
			password:       "wrongPassword",
			want:           false,
		},
		{
			name:           "Empty password",
			hashedPassword: hash,
			password:       "",
			want:           false,
		},
		{
			name:           "Invalid hash",
			hashedPassword: "invalid-hash",
			password:       password,
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyPassword(tt.hashedPassword, tt.password)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestHashPasswordConsistency(t *testing.T) {
	password := "testPassword"

	// Hash the same password twice
	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Hashes should be different (bcrypt uses salt)
	assert.NotEqual(t, hash1, hash2)

	// But both should verify successfully
	assert.True(t, VerifyPassword(hash1, password))
	assert.True(t, VerifyPassword(hash2, password))
}
