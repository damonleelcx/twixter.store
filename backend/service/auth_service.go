package service

import (
	"backend/entity"
	"backend/repository"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService Authentication service interface
type AuthService interface {
	// Registration and login
	Register(email, password, ipAddress string) (*entity.UserBase, *entity.Session, error)
	Login(email, password, ipAddress, userAgent, deviceInfo string) (*entity.UserBase, *entity.Session, error)
	Logout(sessionID uint) error
	LogoutAll(userID uint) error

	// Token management
	RefreshToken(refreshToken string) (*entity.Session, error)
	ValidateToken(accessToken string) (*entity.Session, *entity.UserBase, error)

	// Password management
	RequestPasswordReset(email string) error
	ResetPassword(token, newPassword string) error
	ChangePassword(userID uint, oldPassword, newPassword string) error

	// Account management
	LockAccount(userID uint, duration time.Duration, reason string) error
	UnlockAccount(userID uint) error
	CheckAccountLocked(userID uint) (bool, error)
}

// authService Authentication service implementation
type authService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	db          *gorm.DB
}

// NewAuthService Create authentication service instance
func NewAuthService(db *gorm.DB, userRepo repository.UserRepository, sessionRepo repository.SessionRepository) AuthService {
	return &authService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		db:          db,
	}
}

// Configuration constants
const (
	// Token expiration duration
	AccessTokenDuration  = 15 * time.Minute   // Access token 15 minutes
	RefreshTokenDuration = 7 * 24 * time.Hour // Refresh token 7 days

	// Account lock configuration
	MaxFailedLoginAttempts = 5                // Maximum failed login attempts
	AccountLockDuration    = 30 * time.Minute // Account lock duration

	// Password reset token expiration duration
	PasswordResetTokenDuration = 1 * time.Hour
)

// hashPassword Hash password
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// comparePassword Compare password
func comparePassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// generateToken Generate random token
func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// getJWTSecret Get JWT secret (if using JWT, can be obtained from environment variables)
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-key-change-in-production"
	}
	return secret
}

// Register User registration
func (s *authService) Register(email, password, ipAddress string) (*entity.UserBase, *entity.Session, error) {
	// Check if email already exists
	existingUser, err := s.userRepo.GetByEmailFromShard(email)
	if err == nil && existingUser != nil {
		return nil, nil, errors.New("email already registered")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Hash password
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, nil, err
	}

	// Create user
	user := &entity.UserBase{
		Email:       email,
		Password:    hashedPassword,
		AccountType: entity.AccountTypeLight,
		IPAddress:   ipAddress,
	}

	// Create user and session using transaction
	var createdUser *entity.UserBase
	var session *entity.Session

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Create user (use default table first, then shard by ID)
		userEntity := &entity.User{
			UserBase: *user,
		}
		if err := tx.Create(userEntity).Error; err != nil {
			return err
		}

		// Copy user to shard table
		createdUser = &entity.UserBase{
			ID:          userEntity.ID,
			Email:       userEntity.Email,
			Password:    userEntity.Password,
			AccountType: userEntity.AccountType,
			IPAddress:   userEntity.IPAddress,
			CreatedAt:   userEntity.CreatedAt,
			UpdatedAt:   userEntity.UpdatedAt,
		}

		// Create sharded user
		if err := s.userRepo.CreateInShard(userEntity.ID, createdUser); err != nil {
			return err
		}

		// Generate tokens
		accessToken, err := generateToken(32)
		if err != nil {
			return err
		}
		refreshToken, err := generateToken(32)
		if err != nil {
			return err
		}

		// Create session
		now := time.Now()
		session = &entity.Session{
			UserID:                userEntity.ID,
			AccessToken:           accessToken,
			RefreshToken:          refreshToken,
			TokenType:             "Bearer",
			AccessTokenExpiresAt:  now.Add(AccessTokenDuration),
			RefreshTokenExpiresAt: now.Add(RefreshTokenDuration),
			IPAddress:             ipAddress,
			IsActive:              true,
			LastUsedAt:            now,
		}

		if err := tx.Create(session).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("registration failed: %w", err)
	}

	return createdUser, session, nil
}

// Login User login
func (s *authService) Login(email, password, ipAddress, userAgent, deviceInfo string) (*entity.UserBase, *entity.Session, error) {
	// Get user
	user, err := s.userRepo.GetByEmailFromShard(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("email or password incorrect")
		}
		return nil, nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Check if account is locked
	if user.AccountLocked {
		if user.AccountLockedUntil != nil && time.Now().Before(*user.AccountLockedUntil) {
			return nil, nil, errors.New("account is locked, please try again later")
		}
		// Lock expired, unlock account
		user.AccountLocked = false
		user.AccountLockedUntil = nil
		user.FailedLoginAttempts = 0
		if err := s.userRepo.UpdateInShard(user.ID, user); err != nil {
			return nil, nil, fmt.Errorf("failed to unlock account: %w", err)
		}
	}

	// Verify password
	if !comparePassword(user.Password, password) {
		// Increment failed login attempts
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= MaxFailedLoginAttempts {
			lockUntil := time.Now().Add(AccountLockDuration)
			user.AccountLocked = true
			user.AccountLockedUntil = &lockUntil
		}
		if err := s.userRepo.UpdateInShard(user.ID, user); err != nil {
			return nil, nil, fmt.Errorf("failed to update failed login attempts: %w", err)
		}
		return nil, nil, errors.New("email or password incorrect")
	}

	// Login successful, reset failed login attempts
	now := time.Now()
	user.FailedLoginAttempts = 0
	user.AccountLocked = false
	user.AccountLockedUntil = nil
	user.LastLoginAt = &now
	user.LastIPAddress = ipAddress

	if err := s.userRepo.UpdateInShard(user.ID, user); err != nil {
		return nil, nil, fmt.Errorf("failed to update user info: %w", err)
	}

	// Generate tokens
	accessToken, err := generateToken(32)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	refreshToken, err := generateToken(32)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session
	session := &entity.Session{
		UserID:                user.ID,
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		TokenType:             "Bearer",
		AccessTokenExpiresAt:  now.Add(AccessTokenDuration),
		RefreshTokenExpiresAt: now.Add(RefreshTokenDuration),
		IPAddress:             ipAddress,
		UserAgent:             userAgent,
		DeviceInfo:            deviceInfo,
		IsActive:              true,
		LastUsedAt:            now,
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	return user, session, nil
}

// Logout Logout (revoke single session)
func (s *authService) Logout(sessionID uint) error {
	return s.sessionRepo.RevokeSession(sessionID, "user initiated logout")
}

// LogoutAll Logout all sessions
func (s *authService) LogoutAll(userID uint) error {
	return s.sessionRepo.RevokeAllUserSessions(userID, "user initiated logout from all devices")
}

// RefreshToken Refresh access token
func (s *authService) RefreshToken(refreshToken string) (*entity.Session, error) {
	// Get session
	session, err := s.sessionRepo.GetByRefreshToken(refreshToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("refresh token invalid")
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	// Check if refresh token is expired
	if session.IsRefreshTokenExpired() {
		return nil, errors.New("refresh token expired")
	}

	// Check if session is valid
	if !session.IsActive || session.RevokedAt != nil {
		return nil, errors.New("session revoked")
	}

	// Generate new access token
	newAccessToken, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Update session
	now := time.Now()
	session.AccessToken = newAccessToken
	session.AccessTokenExpiresAt = now.Add(AccessTokenDuration)
	session.LastUsedAt = now

	if err := s.sessionRepo.Update(session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return session, nil
}

// ValidateToken Validate access token
func (s *authService) ValidateToken(accessToken string) (*entity.Session, *entity.UserBase, error) {
	// Get session
	session, err := s.sessionRepo.GetByAccessToken(accessToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("access token invalid")
		}
		return nil, nil, fmt.Errorf("failed to query session: %w", err)
	}

	// Check if session is valid
	if !session.IsValid() {
		return nil, nil, errors.New("session expired or revoked")
	}

	// Update last used time
	now := time.Now()
	if now.Sub(session.LastUsedAt) > 5*time.Minute { // Update every 5 minutes
		session.LastUsedAt = now
		if err := s.sessionRepo.Update(session); err != nil {
			// Log error but don't affect validation result
			fmt.Printf("failed to update session last used time: %v\n", err)
		}
	}

	// Get user information
	user, err := s.userRepo.GetByIDFromShard(session.UserID, session.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query user: %w", err)
	}

	return session, user, nil
}

// RequestPasswordReset Request password reset
func (s *authService) RequestPasswordReset(email string) error {
	// Get user
	user, err := s.userRepo.GetByEmailFromShard(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// For security, return success even if user doesn't exist
			return nil
		}
		return fmt.Errorf("failed to query user: %w", err)
	}

	// Generate reset token
	token, err := generateToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Set token and expiration time
	now := time.Now()
	expiresAt := now.Add(PasswordResetTokenDuration)
	user.PasswordResetToken = token
	user.PasswordResetExpires = &expiresAt

	if err := s.userRepo.UpdateInShard(user.ID, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// TODO: Send password reset email
	// Should call email service to send reset link
	fmt.Printf("Password reset token: %s (for testing only, should be sent via email in production)\n", token)

	return nil
}

// ResetPassword Reset password
func (s *authService) ResetPassword(token, newPassword string) error {
	// Find user with reset token (need to search all shards)
	var user *entity.UserBase

	// Search shard 0 first
	var user0 entity.UserShard0
	if err := s.db.Where("password_reset_token = ?", token).First(&user0).Error; err == nil {
		user = &user0.UserBase
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to query user: %w", err)
	} else {
		// Then search shard 1
		var user1 entity.UserShard1
		if err := s.db.Where("password_reset_token = ?", token).First(&user1).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("reset token invalid")
			}
			return fmt.Errorf("failed to query user: %w", err)
		}
		user = &user1.UserBase
	}

	if user == nil {
		return errors.New("reset token invalid")
	}

	// Check if token is expired
	if user.PasswordResetExpires == nil || time.Now().After(*user.PasswordResetExpires) {
		return errors.New("reset token expired")
	}

	// Hash new password
	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and clear reset token
	user.Password = hashedPassword
	user.PasswordResetToken = ""
	user.PasswordResetExpires = nil

	if err := s.userRepo.UpdateInShard(user.ID, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Revoke all user sessions (security measure)
	if err := s.LogoutAll(user.ID); err != nil {
		fmt.Printf("failed to revoke user sessions: %v\n", err)
	}

	return nil
}

// ChangePassword Change password
func (s *authService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	// Get user
	user, err := s.userRepo.GetByIDFromShard(userID, userID)
	if err != nil {
		return fmt.Errorf("failed to query user: %w", err)
	}

	// Verify old password
	if !comparePassword(user.Password, oldPassword) {
		return errors.New("old password incorrect")
	}

	// Hash new password
	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	user.Password = hashedPassword
	if err := s.userRepo.UpdateInShard(user.ID, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// LockAccount Lock account
func (s *authService) LockAccount(userID uint, duration time.Duration, reason string) error {
	user, err := s.userRepo.GetByIDFromShard(userID, userID)
	if err != nil {
		return fmt.Errorf("failed to query user: %w", err)
	}

	lockUntil := time.Now().Add(duration)
	user.AccountLocked = true
	user.AccountLockedUntil = &lockUntil

	if err := s.userRepo.UpdateInShard(user.ID, user); err != nil {
		return fmt.Errorf("failed to lock account: %w", err)
	}

	// Revoke all user sessions
	if err := s.LogoutAll(userID); err != nil {
		fmt.Printf("failed to revoke user sessions: %v\n", err)
	}

	return nil
}

// UnlockAccount Unlock account
func (s *authService) UnlockAccount(userID uint) error {
	user, err := s.userRepo.GetByIDFromShard(userID, userID)
	if err != nil {
		return fmt.Errorf("failed to query user: %w", err)
	}

	user.AccountLocked = false
	user.AccountLockedUntil = nil
	user.FailedLoginAttempts = 0

	if err := s.userRepo.UpdateInShard(user.ID, user); err != nil {
		return fmt.Errorf("failed to unlock account: %w", err)
	}

	return nil
}

// CheckAccountLocked Check if account is locked
func (s *authService) CheckAccountLocked(userID uint) (bool, error) {
	user, err := s.userRepo.GetByIDFromShard(userID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to query user: %w", err)
	}

	if !user.AccountLocked {
		return false, nil
	}

	// Check if lock is expired
	if user.AccountLockedUntil != nil && time.Now().After(*user.AccountLockedUntil) {
		// Auto unlock
		if err := s.UnlockAccount(userID); err != nil {
			return false, err
		}
		return false, nil
	}

	return true, nil
}
