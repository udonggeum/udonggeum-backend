package repository

import (
	"testing"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUserTest(t *testing.T) (*gorm.DB, UserRepository) {
	testDB, err := db.SetupTestDB()
	require.NoError(t, err)

	repo := NewUserRepository(testDB)
	return testDB, repo
}

func TestUserRepository_Create(t *testing.T) {
	testDB, repo := setupUserTest(t)
	defer db.CleanupTestDB(testDB)

	tests := []struct {
		name    string
		user    *model.User
		wantErr bool
	}{
		{
			name: "Valid user",
			user: &model.User{
				Email:        "test@example.com",
				PasswordHash: "hashedpassword",
				Name:         "Test User",
				Phone:        "010-1234-5678",
				Role:         model.RoleUser,
			},
			wantErr: false,
		},
		{
			name: "Duplicate email",
			user: &model.User{
				Email:        "test@example.com",
				PasswordHash: "hashedpassword",
				Name:         "Another User",
				Phone:        "010-8765-4321",
				Role:         model.RoleUser,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(tt.user)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.user.ID)
			}
		})
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	testDB, repo := setupUserTest(t)
	defer db.CleanupTestDB(testDB)

	// Create test user
	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Name:         "Test User",
		Phone:        "010-1234-5678",
		Role:         model.RoleUser,
	}
	err := repo.Create(user)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      uint
		wantErr bool
	}{
		{
			name:    "Existing user",
			id:      user.ID,
			wantErr: false,
		},
		{
			name:    "Non-existing user",
			id:      9999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := repo.FindByID(tt.id)

			if tt.wantErr {
				assert.Error(t, err)
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

func TestUserRepository_FindByEmail(t *testing.T) {
	testDB, repo := setupUserTest(t)
	defer db.CleanupTestDB(testDB)

	// Create test user
	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Name:         "Test User",
		Phone:        "010-1234-5678",
		Role:         model.RoleUser,
	}
	err := repo.Create(user)
	require.NoError(t, err)

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "Existing email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "Non-existing email",
			email:   "notfound@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := repo.FindByEmail(tt.email)

			if tt.wantErr {
				assert.Error(t, err)
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

func TestUserRepository_Update(t *testing.T) {
	testDB, repo := setupUserTest(t)
	defer db.CleanupTestDB(testDB)

	// Create test user
	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Name:         "Test User",
		Phone:        "010-1234-5678",
		Role:         model.RoleUser,
	}
	err := repo.Create(user)
	require.NoError(t, err)

	// Update user
	user.Name = "Updated Name"
	user.Phone = "010-9999-9999"

	err = repo.Update(user)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "010-9999-9999", updated.Phone)
}

func TestUserRepository_Delete(t *testing.T) {
	testDB, repo := setupUserTest(t)
	defer db.CleanupTestDB(testDB)

	// Create test user
	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Name:         "Test User",
		Phone:        "010-1234-5678",
		Role:         model.RoleUser,
	}
	err := repo.Create(user)
	require.NoError(t, err)

	// Delete user
	err = repo.Delete(user.ID)
	assert.NoError(t, err)

	// Verify deletion (soft delete)
	_, err = repo.FindByID(user.ID)
	assert.Error(t, err)
}
