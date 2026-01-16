package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ikkim/udonggeum-backend/internal/app/model"
	"github.com/ikkim/udonggeum-backend/internal/app/repository"
	"github.com/ikkim/udonggeum-backend/pkg/logger"
	redisClient "github.com/ikkim/udonggeum-backend/pkg/redis"
	"github.com/ikkim/udonggeum-backend/pkg/util"
	"gorm.io/gorm"
)

var (
	ErrEmailAlreadyExists       = errors.New("이미 사용 중인 이메일입니다")
	ErrInvalidCredentials       = errors.New("이메일 또는 비밀번호가 올바르지 않습니다")
	ErrUserNotFound             = errors.New("사용자를 찾을 수 없습니다")
	ErrInvalidToken             = errors.New("유효하지 않은 토큰입니다")
	ErrExpiredToken             = errors.New("토큰이 만료되었습니다")
	ErrTokenRevoked             = errors.New("토큰이 폐기되었습니다")
	ErrNicknameAlreadyExists    = errors.New("이미 사용 중인 닉네임입니다")
	ErrInvalidVerificationCode  = errors.New("유효하지 않거나 만료된 인증 코드입니다")
	ErrEmailAlreadyVerified     = errors.New("이미 인증된 이메일입니다")
	ErrPhoneAlreadyVerified     = errors.New("이미 인증된 휴대폰입니다")
)

type AuthService interface {
	Register(email, password, name, nickname, phone string) (*model.User, *util.TokenPair, error)
	Login(email, password string) (*model.User, *util.TokenPair, error)
	GetUserByID(id uint) (*model.User, error)
	UpdateProfile(userID uint, name, phone, nickname, address, profileImage string) (*model.User, error)
	CheckNickname(nickname string) (bool, error)
	CheckEmailAvailability(email string) (bool, error)
	RefreshToken(refreshToken string) (*util.TokenPair, error)
	RevokeToken(refreshToken string) error
	GetKakaoLoginURL() string
	KakaoLogin(code string) (*model.User, *util.TokenPair, error)

	// 이메일/휴대폰 인증
	SendEmailVerification(email string) error
	VerifyEmail(email, code string) error
	SendPhoneVerification(userID uint, phone string) error
	VerifyPhone(userID uint, phone, code string) error
}

type authService struct {
	userRepo          repository.UserRepository
	jwtSecret         string
	accessExpiry      time.Duration
	refreshExpiry     time.Duration
	kakaoClientID     string
	kakaoClientSecret string
	kakaoRedirectURI  string
}

func NewAuthService(
	userRepo repository.UserRepository,
	jwtSecret string,
	accessExpiry, refreshExpiry time.Duration,
	kakaoClientID, kakaoClientSecret, kakaoRedirectURI string,
) AuthService {
	return &authService{
		userRepo:          userRepo,
		jwtSecret:         jwtSecret,
		accessExpiry:      accessExpiry,
		refreshExpiry:     refreshExpiry,
		kakaoClientID:     kakaoClientID,
		kakaoClientSecret: kakaoClientSecret,
		kakaoRedirectURI:  kakaoRedirectURI,
	}
}

func (s *authService) Register(email, password, name, nickname, phone string) (*model.User, *util.TokenPair, error) {
	logger.Info("Attempting user registration", map[string]interface{}{
		"email":    email,
		"name":     name,
		"nickname": nickname,
	})

	// Check if user already exists
	existingUser, err := s.userRepo.FindByEmail(email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("Failed to check existing user", err, map[string]interface{}{
			"email": email,
		})
		return nil, nil, err
	}
	if existingUser != nil {
		logger.Warn("Registration failed: email already exists", map[string]interface{}{
			"email": email,
		})
		return nil, nil, ErrEmailAlreadyExists
	}

	// Hash password
	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		logger.Error("Failed to hash password", err, map[string]interface{}{
			"email": email,
		})
		return nil, nil, err
	}

	// Use provided nickname or generate unique one
	var finalNickname string
	if nickname == "" {
		generatedNickname, err := s.generateUniqueNickname()
		if err != nil {
			logger.Error("Failed to generate unique nickname", err, map[string]interface{}{
				"email": email,
			})
			return nil, nil, err
		}
		finalNickname = generatedNickname
		logger.Debug("Generated unique nickname", map[string]interface{}{
			"nickname": finalNickname,
		})
	} else {
		// Check if provided nickname already exists
		existingNicknameUser, err := s.userRepo.FindByNickname(nickname)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("Failed to check existing nickname", err, map[string]interface{}{
				"nickname": nickname,
			})
			return nil, nil, err
		}
		if existingNicknameUser != nil {
			logger.Warn("Registration failed: nickname already exists", map[string]interface{}{
				"nickname": nickname,
			})
			return nil, nil, ErrNicknameAlreadyExists
		}
		finalNickname = nickname
		logger.Debug("Using provided nickname", map[string]interface{}{
			"nickname": finalNickname,
		})
	}

	// Create user
	user := &model.User{
		Email:        email,
		PasswordHash: hashedPassword,
		Name:         name,
		Nickname:     finalNickname,
		Phone:        phone,
		Role:         model.RoleUser,
	}

	if err := s.userRepo.Create(user); err != nil {
		logger.Error("Failed to create user in database", err, map[string]interface{}{
			"email": email,
		})
		return nil, nil, err
	}

	// Generate tokens
	tokens, err := util.GenerateTokenPair(
		user.ID,
		user.Email,
		string(user.Role),
		s.jwtSecret,
		s.accessExpiry,
		s.refreshExpiry,
	)
	if err != nil {
		logger.Error("Failed to generate tokens", err, map[string]interface{}{
			"user_id": user.ID,
			"email":   email,
		})
		return nil, nil, err
	}

	logger.Info("User registered successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   email,
		"role":    user.Role,
	})

	return user, tokens, nil
}

func (s *authService) Login(email, password string) (*model.User, *util.TokenPair, error) {
	logger.Info("Login attempt", map[string]interface{}{
		"email": email,
	})

	// Find user by email
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Login failed: user not found", map[string]interface{}{
				"email": email,
			})
			return nil, nil, ErrInvalidCredentials
		}
		logger.Error("Failed to find user", err, map[string]interface{}{
			"email": email,
		})
		return nil, nil, err
	}

	// Verify password
	if !util.VerifyPassword(user.PasswordHash, password) {
		logger.Warn("Login failed: invalid password", map[string]interface{}{
			"email":   email,
			"user_id": user.ID,
		})
		return nil, nil, ErrInvalidCredentials
	}

	// Generate tokens
	tokens, err := util.GenerateTokenPair(
		user.ID,
		user.Email,
		string(user.Role),
		s.jwtSecret,
		s.accessExpiry,
		s.refreshExpiry,
	)
	if err != nil {
		logger.Error("Failed to generate tokens", err, map[string]interface{}{
			"user_id": user.ID,
			"email":   email,
		})
		return nil, nil, err
	}

	logger.Info("User logged in successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   email,
		"role":    user.Role,
	})

	return user, tokens, nil
}

func (s *authService) GetUserByID(id uint) (*model.User, error) {
	logger.Debug("Fetching user by ID", map[string]interface{}{
		"user_id": id,
	})

	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("User not found", map[string]interface{}{
				"user_id": id,
			})
			return nil, ErrUserNotFound
		}
		logger.Error("Failed to fetch user", err, map[string]interface{}{
			"user_id": id,
		})
		return nil, err
	}

	logger.Debug("User fetched successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return user, nil
}

func (s *authService) UpdateProfile(userID uint, name, phone, nickname, address, profileImage string) (*model.User, error) {
	logger.Info("Updating user profile", map[string]interface{}{
		"user_id": userID,
	})

	// Fetch existing user
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("User not found for profile update", map[string]interface{}{
				"user_id": userID,
			})
			return nil, ErrUserNotFound
		}
		logger.Error("Failed to fetch user for profile update", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	// Update fields if provided
	updated := false
	if name != "" && name != user.Name {
		user.Name = name
		updated = true
	}
	if phone != "" && phone != user.Phone {
		user.Phone = phone
		updated = true
	}
	if nickname != "" && nickname != user.Nickname {
		// Admin 사용자는 닉네임을 직접 수정할 수 없음 (매장 이름과 자동 동기화됨)
		if user.Role == model.RoleAdmin {
			logger.Warn("Admin users cannot update nickname directly", map[string]interface{}{
				"user_id": userID,
			})
			return nil, errors.New("관리자는 닉네임을 직접 수정할 수 없습니다 - 매장 이름과 자동으로 동기화됩니다")
		}

		// Check if nickname already exists
		existingUser, err := s.userRepo.FindByNickname(nickname)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("Failed to check existing nickname", err, map[string]interface{}{
				"nickname": nickname,
			})
			return nil, err
		}
		if existingUser != nil && existingUser.ID != userID {
			logger.Warn("Nickname already exists", map[string]interface{}{
				"nickname": nickname,
			})
			return nil, ErrNicknameAlreadyExists
		}
		user.Nickname = nickname
		updated = true
	}
	// Address can be empty (to clear it), so we don't check for empty string
	if address != user.Address {
		user.Address = address
		updated = true

		// Geocode the address to get latitude and longitude
		if address != "" {
			lat, lng, err := util.GeocodeAddress(address)
			if err != nil {
				logger.Warn("Failed to geocode address, continuing without coordinates", map[string]interface{}{
					"address": address,
					"error":   err.Error(),
				})
				// Don't fail the update if geocoding fails
				// User can still use the service, just without location-based features
			} else {
				user.Latitude = lat
				user.Longitude = lng
				logger.Info("Successfully geocoded address", map[string]interface{}{
					"address":   address,
					"latitude":  lat,
					"longitude": lng,
				})
			}
		} else {
			// Clear coordinates if address is cleared
			user.Latitude = nil
			user.Longitude = nil
		}
	}
	// ProfileImage can be empty (to clear it) or update to new URL
	if profileImage != user.ProfileImage {
		user.ProfileImage = profileImage
		updated = true
	}

	// Only update if there are changes
	if !updated {
		logger.Debug("No changes detected for user profile", map[string]interface{}{
			"user_id": userID,
		})
		return user, nil
	}

	// Save changes
	if err := s.userRepo.Update(user); err != nil {
		logger.Error("Failed to update user profile", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, err
	}

	logger.Info("User profile updated successfully", map[string]interface{}{
		"user_id":  user.ID,
		"name":     user.Name,
		"phone":    user.Phone,
		"nickname": user.Nickname,
		"address":  user.Address,
	})

	return user, nil
}

// CheckNickname checks if a nickname is available
func (s *authService) CheckNickname(nickname string) (bool, error) {
	logger.Debug("Checking nickname availability", map[string]interface{}{
		"nickname": nickname,
	})

	// Check if nickname already exists
	existingUser, err := s.userRepo.FindByNickname(nickname)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("Failed to check nickname availability", err, map[string]interface{}{
			"nickname": nickname,
		})
		return false, err
	}

	// If user exists, nickname is not available
	isAvailable := existingUser == nil
	logger.Debug("Nickname availability checked", map[string]interface{}{
		"nickname":    nickname,
		"is_available": isAvailable,
	})

	return isAvailable, nil
}

// CheckEmailAvailability checks if an email is available for registration
func (s *authService) CheckEmailAvailability(email string) (bool, error) {
	logger.Debug("Checking email availability", map[string]interface{}{
		"email": email,
	})

	// Check if email already exists
	existingUser, err := s.userRepo.FindByEmail(email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("Failed to check email availability", err, map[string]interface{}{
			"email": email,
		})
		return false, err
	}

	// If user exists, email is not available
	isAvailable := existingUser == nil
	logger.Debug("Email availability checked", map[string]interface{}{
		"email":        email,
		"is_available": isAvailable,
	})

	return isAvailable, nil
}

// RefreshToken validates a refresh token and generates a new token pair
func (s *authService) RefreshToken(refreshToken string) (*util.TokenPair, error) {
	logger.Debug("Attempting to refresh token")

	// Check if token is blacklisted
	ctx := context.Background()
	isBlacklisted, err := redisClient.IsTokenBlacklisted(ctx, refreshToken)
	if err != nil {
		logger.Error("Failed to check token blacklist", err, nil)
		return nil, err
	}
	if isBlacklisted {
		logger.Warn("Attempted to use revoked refresh token", nil)
		return nil, ErrTokenRevoked
	}

	// Validate the refresh token
	claims, err := util.ValidateToken(refreshToken, s.jwtSecret)
	if err != nil {
		if errors.Is(err, util.ErrExpiredToken) {
			logger.Warn("Refresh token has expired", nil)
			return nil, ErrExpiredToken
		}
		logger.Warn("Invalid refresh token", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, ErrInvalidToken
	}

	// Verify user still exists
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("User not found for token refresh", map[string]interface{}{
				"user_id": claims.UserID,
			})
			return nil, ErrUserNotFound
		}
		logger.Error("Failed to fetch user for token refresh", err, map[string]interface{}{
			"user_id": claims.UserID,
		})
		return nil, err
	}

	// Generate new token pair
	tokens, err := util.GenerateTokenPair(
		user.ID,
		user.Email,
		string(user.Role),
		s.jwtSecret,
		s.accessExpiry,
		s.refreshExpiry,
	)
	if err != nil {
		logger.Error("Failed to generate new token pair", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, err
	}

	// Blacklist the old refresh token (token rotation)
	if err := redisClient.BlacklistToken(ctx, refreshToken, s.refreshExpiry); err != nil {
		logger.Error("Failed to blacklist old refresh token", err, nil)
		// Don't fail the request, just log the error
	}

	logger.Info("Token refreshed successfully", map[string]interface{}{
		"user_id": user.ID,
	})

	return tokens, nil
}

// RevokeToken adds a refresh token to the blacklist
func (s *authService) RevokeToken(refreshToken string) error {
	logger.Debug("Attempting to revoke token")

	// Validate token to get expiry time
	claims, err := util.ValidateToken(refreshToken, s.jwtSecret)
	if err != nil {
		// Even if token is invalid/expired, we still blacklist it
		logger.Warn("Revoking invalid/expired token", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Calculate remaining TTL
	var ttl time.Duration
	if claims != nil && claims.ExpiresAt != nil {
		ttl = time.Until(claims.ExpiresAt.Time)
		if ttl < 0 {
			// Token already expired, no need to blacklist
			logger.Debug("Token already expired, skipping blacklist", nil)
			return nil
		}
	} else {
		// Default to refresh token expiry if we can't determine
		ttl = s.refreshExpiry
	}

	ctx := context.Background()
	if err := redisClient.BlacklistToken(ctx, refreshToken, ttl); err != nil {
		logger.Error("Failed to blacklist token", err, nil)
		return err
	}

	logger.Info("Token revoked successfully", nil)
	return nil
}

// generateUniqueNickname generates a random unique nickname
func (s *authService) generateUniqueNickname() (string, error) {
	const (
		maxRetries = 10
		prefix     = "사용자"
	)

	for i := 0; i < maxRetries; i++ {
		// Generate random 6-digit number
		randomNum := util.GenerateRandomNumber(100000, 999999)
		nickname := fmt.Sprintf("%s%d", prefix, randomNum)

		// Check if nickname already exists
		existingUser, err := s.userRepo.FindByNickname(nickname)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("Failed to check existing nickname", err, map[string]interface{}{
				"nickname": nickname,
			})
			return "", err
		}

		// If nickname doesn't exist, return it
		if existingUser == nil {
			logger.Debug("Generated unique nickname", map[string]interface{}{
				"nickname": nickname,
			})
			return nickname, nil
		}

		logger.Debug("Nickname already exists, retrying", map[string]interface{}{
			"nickname": nickname,
			"attempt":  i + 1,
		})
	}

	return "", errors.New("고유한 닉네임을 생성하는데 실패했습니다. 다시 시도해주세요")
}

// normalizePhoneNumber normalizes phone number from Kakao format to storage format
// Input examples: "+82 10-1234-5678", "+82 1012345678", "+82-10-1234-5678"
// Output: "01012345678" (digits only, +82 replaced with 0)
func normalizePhoneNumber(kakaoPhone string) string {
	if kakaoPhone == "" {
		return ""
	}

	// Remove all spaces and hyphens
	phone := strings.ReplaceAll(kakaoPhone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	// Replace +82 with 0
	if strings.HasPrefix(phone, "+82") {
		phone = "0" + phone[3:]
	}

	// Extract only digits
	re := regexp.MustCompile(`\d+`)
	phone = strings.Join(re.FindAllString(phone, -1), "")

	// Validate Korean phone number format (10-11 digits starting with 0)
	if len(phone) < 10 || len(phone) > 11 || !strings.HasPrefix(phone, "0") {
		logger.Warn("Invalid phone number format after normalization", map[string]interface{}{
			"original": kakaoPhone,
			"normalized": phone,
		})
		return ""
	}

	return phone
}

// getProfileImageURL returns the best available profile image URL from Kakao
func getProfileImageURL(kakaoUserInfo *kakaoUserInfo) string {
	// Priority: kakao_account.profile.profile_image_url > properties.profile_image
	if kakaoUserInfo.KakaoAccount.Profile.ProfileImageURL != "" &&
	   !kakaoUserInfo.KakaoAccount.Profile.IsDefaultImage {
		return kakaoUserInfo.KakaoAccount.Profile.ProfileImageURL
	}

	if kakaoUserInfo.Properties.ProfileImage != "" {
		return kakaoUserInfo.Properties.ProfileImage
	}

	return ""
}

// GetKakaoLoginURL returns the Kakao OAuth login URL
func (s *authService) GetKakaoLoginURL() string {
	return fmt.Sprintf("https://kauth.kakao.com/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code",
		s.kakaoClientID, s.kakaoRedirectURI)
}

// KakaoLogin handles Kakao OAuth login
func (s *authService) KakaoLogin(code string) (*model.User, *util.TokenPair, error) {
	logger.Info("Starting Kakao login", map[string]interface{}{
		"code": code,
	})

	// 1. Get Kakao access token
	kakaoToken, err := s.getKakaoToken(code)
	if err != nil {
		logger.Error("Failed to get Kakao access token", err, nil)
		return nil, nil, fmt.Errorf("failed to get Kakao access token: %w", err)
	}

	// 2. Get user info from Kakao
	kakaoUserInfo, err := s.getKakaoUserInfo(kakaoToken.AccessToken)
	if err != nil {
		logger.Error("Failed to get Kakao user info", err, nil)
		return nil, nil, fmt.Errorf("failed to get Kakao user info: %w", err)
	}

	logger.Debug("Kakao user info retrieved", map[string]interface{}{
		"email": kakaoUserInfo.KakaoAccount.Email,
	})

	// 3. Check if user already exists
	user, err := s.userRepo.FindByEmail(kakaoUserInfo.KakaoAccount.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("Failed to check existing user", err, map[string]interface{}{
			"email": kakaoUserInfo.KakaoAccount.Email,
		})
		return nil, nil, err
	}

	// 4. Create new user if not exists
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Info("Creating new user from Kakao login", map[string]interface{}{
			"email": kakaoUserInfo.KakaoAccount.Email,
		})

		// Generate unique nickname
		nickname, err := s.generateUniqueNickname()
		if err != nil {
			logger.Error("Failed to generate unique nickname", err, nil)
			return nil, nil, err
		}

		// Normalize phone number
		phone := normalizePhoneNumber(kakaoUserInfo.KakaoAccount.PhoneNumber)

		// Get profile image URL
		profileImage := getProfileImageURL(kakaoUserInfo)

		user = &model.User{
			Email:        kakaoUserInfo.KakaoAccount.Email,
			PasswordHash: "", // Kakao login users don't have password
			Name:         kakaoUserInfo.Properties.Nickname,
			Nickname:     nickname,
			Phone:        phone,
			ProfileImage: profileImage,
			Role:         model.RoleUser,
		}

		logger.Debug("Creating user with Kakao info", map[string]interface{}{
			"email":         user.Email,
			"name":          user.Name,
			"phone":         user.Phone,
			"profile_image": user.ProfileImage,
		})

		if err := s.userRepo.Create(user); err != nil {
			logger.Error("Failed to create Kakao user", err, map[string]interface{}{
				"email": kakaoUserInfo.KakaoAccount.Email,
			})
			return nil, nil, err
		}

		logger.Info("New user created from Kakao login", map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
		})
	} else {
		logger.Info("Existing user found for Kakao login", map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
		})
	}

	// 5. Store Kakao access token in Redis
	ctx := context.Background()
	kakaoRedisKey := fmt.Sprintf("kakao_access_token:%d", user.ID)
	if err := redisClient.StoreKakaoToken(ctx, kakaoRedisKey, kakaoToken.AccessToken, 15*time.Minute); err != nil {
		logger.Error("Failed to store Kakao access token in Redis", err, map[string]interface{}{
			"user_id": user.ID,
		})
		// Don't fail the login if Redis storage fails
	}

	// 6. Generate JWT tokens
	tokens, err := util.GenerateTokenPair(
		user.ID,
		user.Email,
		string(user.Role),
		s.jwtSecret,
		s.accessExpiry,
		s.refreshExpiry,
	)
	if err != nil {
		logger.Error("Failed to generate tokens for Kakao login", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, nil, err
	}

	logger.Info("Kakao login successful", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return user, tokens, nil
}

// kakaoTokenResponse represents Kakao token response
type kakaoTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// kakaoUserInfo represents Kakao user information
type kakaoUserInfo struct {
	ID           int64         `json:"id"`
	ConnectedAt  string        `json:"connected_at"`
	Properties   kakaoProperties `json:"properties"`
	KakaoAccount kakaoAccount  `json:"kakao_account"`
}

type kakaoProperties struct {
	Nickname       string `json:"nickname"`
	ProfileImage   string `json:"profile_image"`
	ThumbnailImage string `json:"thumbnail_image"`
}

type kakaoAccount struct {
	Email                 string        `json:"email"`
	ProfileNeedsAgreement bool          `json:"profile_needs_agreement"`
	HasEmail              bool          `json:"has_email"`
	PhoneNumber           string        `json:"phone_number"`
	HasPhoneNumber        bool          `json:"has_phone_number"`
	Profile               kakaoProfile  `json:"profile"`
}

type kakaoProfile struct {
	Nickname         string `json:"nickname"`
	ProfileImageURL  string `json:"profile_image_url"`
	ThumbnailURL     string `json:"thumbnail_image_url"`
	IsDefaultImage   bool   `json:"is_default_image"`
}

// getKakaoToken requests access token from Kakao
func (s *authService) getKakaoToken(code string) (*kakaoTokenResponse, error) {
	logger.Debug("Requesting Kakao token")

	reqBody := fmt.Sprintf("grant_type=authorization_code&client_id=%s&client_secret=%s&redirect_uri=%s&code=%s",
		s.kakaoClientID, s.kakaoClientSecret, s.kakaoRedirectURI, code)

	resp, err := http.Post("https://kauth.kakao.com/oauth/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(reqBody))
	if err != nil {
		logger.Error("Failed to make HTTP request to Kakao token endpoint", err, nil)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read Kakao token response", err, nil)
		return nil, err
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		logger.Error("Kakao token request failed", nil, map[string]interface{}{
			"status_code":   resp.StatusCode,
			"response_body": string(body),
		})
		return nil, fmt.Errorf("kakao token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp kakaoTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		logger.Error("Failed to unmarshal Kakao token response", err, map[string]interface{}{
			"response_body": string(body),
		})
		return nil, err
	}

	// Validate access token exists
	if tokenResp.AccessToken == "" {
		logger.Error("Kakao token response missing access_token", nil, map[string]interface{}{
			"response_body": string(body),
		})
		return nil, fmt.Errorf("kakao token response missing access_token")
	}

	logger.Debug("Kakao token obtained successfully")
	return &tokenResp, nil
}

// getKakaoUserInfo gets user information from Kakao
func (s *authService) getKakaoUserInfo(accessToken string) (*kakaoUserInfo, error) {
	logger.Debug("Requesting Kakao user info")

	req, err := http.NewRequest("GET", "https://kapi.kakao.com/v2/user/me", nil)
	if err != nil {
		logger.Error("Failed to create HTTP request for Kakao user info", err, nil)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to make HTTP request to Kakao user info endpoint", err, nil)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read Kakao user info response", err, nil)
		return nil, err
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		logger.Error("Kakao user info request failed", nil, map[string]interface{}{
			"status_code":   resp.StatusCode,
			"response_body": string(body),
		})
		return nil, fmt.Errorf("kakao user info request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo kakaoUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		logger.Error("Failed to unmarshal Kakao user info response", err, map[string]interface{}{
			"response_body": string(body),
		})
		return nil, err
	}

	// Validate email exists
	if userInfo.KakaoAccount.Email == "" {
		logger.Error("Kakao user info missing email", nil, map[string]interface{}{
			"response_body": string(body),
		})
		return nil, fmt.Errorf("카카오 사용자가 이메일을 제공하지 않았습니다 - 이메일 동의가 필요합니다")
	}

	logger.Debug("Kakao user info obtained successfully", map[string]interface{}{
		"email": userInfo.KakaoAccount.Email,
	})
	return &userInfo, nil
}

// === 이메일/휴대폰 인증 메서드 ===

// SendEmailVerification sends verification code to email
func (s *authService) SendEmailVerification(email string) error {
	// Generate verification code
	code, err := util.GenerateVerificationCode()
	if err != nil {
		return fmt.Errorf("failed to generate verification code: %w", err)
	}

	// Store code
	util.StoreEmailVerificationCode(email, code)

	// Send email
	err = util.SendVerificationEmail(email, code)
	if err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}

// VerifyEmail verifies email with code
func (s *authService) VerifyEmail(email, code string) error {
	// Verify code
	if !util.VerifyEmailCode(email, code) {
		return ErrInvalidVerificationCode
	}

	// Find user by email
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 회원가입 전 이메일 인증인 경우 - 이메일만 검증하고 user 업데이트는 하지 않음
			return nil
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Update user's email_verified status
	now := time.Now()
	user.EmailVerified = true
	user.EmailVerifiedAt = &now

	err = s.userRepo.Update(user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// SendPhoneVerification sends verification code to phone
func (s *authService) SendPhoneVerification(userID uint, phone string) error {
	// Get user
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Check if already verified
	if user.PhoneVerified {
		return ErrPhoneAlreadyVerified
	}

	// Generate verification code
	code, err := util.GenerateVerificationCode()
	if err != nil {
		return fmt.Errorf("failed to generate verification code: %w", err)
	}

	// Store code
	util.StorePhoneVerificationCode(phone, code)

	// Send SMS
	err = util.SendVerificationSMS(phone, code)
	if err != nil {
		return fmt.Errorf("failed to send verification SMS: %w", err)
	}

	return nil
}

// VerifyPhone verifies phone with code
func (s *authService) VerifyPhone(userID uint, phone, code string) error {
	// Verify code
	if !util.VerifyPhoneCode(phone, code) {
		return ErrInvalidVerificationCode
	}

	// Get user
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Update user's phone and phone_verified status
	now := time.Now()
	user.Phone = phone
	user.PhoneVerified = true
	user.PhoneVerifiedAt = &now

	err = s.userRepo.Update(user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}
