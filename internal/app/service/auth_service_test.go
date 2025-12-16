package service

import (
	"testing"
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthServiceTest(t *testing.T) (AuthService, *repository.UserRepository) {
	testDB, err := db.SetupTestDB(t)
	require.NoError(t, err)

	userRepo := repository.NewUserRepository(testDB)
	authService := NewAuthService(
		userRepo,
		"test-jwt-secret",
		15*time.Minute,
		7*24*time.Hour,
		"test-kakao-client-id",
		"test-kakao-client-secret",
		"http://localhost:8080/api/v1/auth/kakao/callback",
	)

	return authService, &userRepo
}

func TestAuthService_Register(t *testing.T) {
	authService, _ := setupAuthServiceTest(t)

	tests := []struct {
		name     string
		email    string
		password string
		userName string
		phone    string
		wantErr  error
	}{
		{
			name:     "Valid registration",
			email:    "test@example.com",
			password: "password123",
			userName: "Test User",
			phone:    "010-1234-5678",
			wantErr:  nil,
		},
		{
			name:     "Duplicate email",
			email:    "test@example.com",
			password: "password456",
			userName: "Another User",
			phone:    "010-8765-4321",
			wantErr:  ErrEmailAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, tokens, err := authService.Register(
				tt.email,
				tt.password,
				tt.userName,
				tt.phone,
			)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
				assert.Nil(t, tokens)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				require.NotNil(t, tokens)
				assert.Equal(t, tt.email, user.Email)
				assert.Equal(t, tt.userName, user.Name)
				assert.Equal(t, model.RoleUser, user.Role)
				assert.NotEmpty(t, tokens.AccessToken)
				assert.NotEmpty(t, tokens.RefreshToken)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	authService, _ := setupAuthServiceTest(t)

	// Register a user first
	email := "test@example.com"
	password := "password123"
	_, _, err := authService.Register(email, password, "Test User", "010-1234-5678")
	require.NoError(t, err)

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{
			name:     "Valid login",
			email:    email,
			password: password,
			wantErr:  nil,
		},
		{
			name:     "Wrong password",
			email:    email,
			password: "wrongpassword",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "Non-existing user",
			email:    "notfound@example.com",
			password: "password123",
			wantErr:  ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, tokens, err := authService.Login(tt.email, tt.password)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
				assert.Nil(t, tokens)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				require.NotNil(t, tokens)
				assert.Equal(t, tt.email, user.Email)
				assert.NotEmpty(t, tokens.AccessToken)
				assert.NotEmpty(t, tokens.RefreshToken)
			}
		})
	}
}

func TestAuthService_GetUserByID(t *testing.T) {
	authService, _ := setupAuthServiceTest(t)

	// Register a user
	user, _, err := authService.Register(
		"test@example.com",
		"password123",
		"Test User",
		"010-1234-5678",
	)
	require.NoError(t, err)

	tests := []struct {
		name    string
		userID  uint
		wantErr error
	}{
		{
			name:    "Existing user",
			userID:  user.ID,
			wantErr: nil,
		},
		{
			name:    "Non-existing user",
			userID:  9999,
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := authService.GetUserByID(tt.userID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, found)
			} else {
				require.NoError(t, err)
				require.NotNil(t, found)
				assert.Equal(t, user.Email, found.Email)
				assert.Equal(t, user.Name, found.Name)
			}
		})
	}
}

func TestAuthService_PasswordSecurity(t *testing.T) {
	authService, _ := setupAuthServiceTest(t)

	password := "mySecretPassword123"
	user, _, err := authService.Register(
		"test@example.com",
		password,
		"Test User",
		"010-1234-5678",
	)
	require.NoError(t, err)

	// Password should be hashed
	assert.NotEqual(t, password, user.PasswordHash)
	assert.Contains(t, user.PasswordHash, "$2a$")
}

func TestAuthService_TokenGeneration(t *testing.T) {
	authService, _ := setupAuthServiceTest(t)

	user, tokens, err := authService.Register(
		"test@example.com",
		"password123",
		"Test User",
		"010-1234-5678",
	)
	require.NoError(t, err)

	// Tokens should be different
	assert.NotEqual(t, tokens.AccessToken, tokens.RefreshToken)

	// Tokens should be valid JWT format
	assert.Contains(t, tokens.AccessToken, ".")
	assert.Contains(t, tokens.RefreshToken, ".")

    // Login should generate new tokens
    _, newTokens, err := authService.Login("test@example.com", "password123")
    require.NoError(t, err)
    assert.NotEmpty(t, newTokens.AccessToken)
    assert.NotEmpty(t, newTokens.RefreshToken)

	_ = user
}
