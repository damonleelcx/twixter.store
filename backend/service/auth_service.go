package service

import (
	"backend/entity"
	"backend/repository"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService Authentication service interface
type AuthService interface {
	// Registration and login
	Register(email, password, ipAddress, username, referralCode string, accountType entity.AccountType, promoFreeCredits bool) (*entity.UserBase, *entity.Session, error)
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

	// GetUserPermissionNames 获取用户权限名称列表（用于前端权限判断）
	GetUserPermissionNames(userID uint) ([]string, error)
	// GetWalletBalance 获取用户钱包余额（积分）
	GetWalletBalance(userID uint) (int64, error)
	// GetMembershipStatus 获取用户会员状态：active 表示有效会员并返回到期时间，none 表示无有效会员
	GetMembershipStatus(userID uint) (status string, expiresAt *time.Time)
	// EnsureAdminSeed 若不存在任何 admin 账户则根据环境变量创建种子 admin（ADMIN_EMAIL、ADMIN_PASSWORD）
	EnsureAdminSeed(adminEmail, adminPassword string) error
}

// authService Authentication service implementation
type authService struct {
	userRepo           repository.UserRepository
	sessionRepo        repository.SessionRepository
	walletRepo         repository.WalletRepository
	purchaseRepo       repository.PurchaseRepository
	userPermissionRepo repository.UserPermissionRepository
	permissionRepo     repository.PermissionRepository
	emailService       EmailService // optional; nil 时不发邮件
	db                 *gorm.DB
}

// NewAuthService Create authentication service instance.
// emailService 可为 nil，为 nil 时密码重置仅打印 token（开发用），不发送邮件。
func NewAuthService(db *gorm.DB, userRepo repository.UserRepository, sessionRepo repository.SessionRepository, walletRepo repository.WalletRepository, purchaseRepo repository.PurchaseRepository, userPermissionRepo repository.UserPermissionRepository, permissionRepo repository.PermissionRepository, emailService EmailService) AuthService {
	return &authService{
		userRepo:           userRepo,
		sessionRepo:        sessionRepo,
		walletRepo:         walletRepo,
		purchaseRepo:       purchaseRepo,
		userPermissionRepo: userPermissionRepo,
		permissionRepo:     permissionRepo,
		emailService:       emailService,
		db:                 db,
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

	// Referral configuration
	ReferralRewardCredits = 20 // 推荐奖励积分数量
	ReferralCodeLength    = 8  // 推荐码长度

	// Signup promo (promo=freecredits)
	SignupPromoCredits = 20 // 通过 promo=freecredits 注册赠送的积分数量
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

// generateReferralCode Generate unique referral code
func (s *authService) generateReferralCode() (string, error) {
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		// 生成一个简短的推荐码（使用大写字母和数字）
		const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		bytes := make([]byte, ReferralCodeLength)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("failed to generate referral code: %w", err)
		}
		for i := range bytes {
			bytes[i] = charset[bytes[i]%byte(len(charset))]
		}
		code := string(bytes)

		// 检查推荐码是否已存在
		_, err := s.userRepo.GetByReferralCode(code)
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			return code, nil // 推荐码不存在，可以使用
		} else if err != nil {
			return "", fmt.Errorf("failed to check referral code: %w", err)
		}
		// 如果推荐码已存在，继续尝试生成新的
	}
	return "", fmt.Errorf("failed to generate unique referral code after %d attempts", maxAttempts)
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
func (s *authService) Register(email, password, ipAddress, username, referralCode string, accountType entity.AccountType, promoFreeCredits bool) (*entity.UserBase, *entity.Session, error) {
	// Check if email already exists
	existingUser, err := s.userRepo.GetByEmailFromShard(email)
	if err == nil && existingUser != nil {
		return nil, nil, errors.New("email already registered")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Check if username already exists (if provided)
	if username != "" {
		// Search for existing username in both shards
		var existingUser0 entity.UserShard0
		var existingUser1 entity.UserShard1
		if err := s.db.Where("username = ?", username).First(&existingUser0).Error; err == nil {
			return nil, nil, errors.New("username already taken")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("failed to check username: %w", err)
		}
		if err := s.db.Where("username = ?", username).First(&existingUser1).Error; err == nil {
			return nil, nil, errors.New("username already taken")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("failed to check username: %w", err)
		}
	}

	// Hash password
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, nil, err
	}

	// Generate referral code for new user
	newReferralCode, err := s.generateReferralCode()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate referral code: %w", err)
	}

	// Validate referral code if provided
	var referredBy *uint
	if referralCode != "" {
		referrer, err := s.userRepo.GetByReferralCode(referralCode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil, errors.New("invalid referral code")
			}
			return nil, nil, fmt.Errorf("failed to validate referral code: %w", err)
		}
		referredBy = &referrer.ID
	}

	// Create user
	user := &entity.UserBase{
		Email:        email,
		Username:     username, // Username from request (can be empty)
		Password:     hashedPassword,
		AccountType:  accountType, // 根据注册来源设置账户类型
		IPAddress:    ipAddress,
		ReferralCode: newReferralCode,
		ReferredBy:   referredBy,
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
			ID:           userEntity.ID,
			Email:        userEntity.Email,
			Username:     userEntity.Username,
			Password:     userEntity.Password,
			AccountType:  userEntity.AccountType,
			IPAddress:    userEntity.IPAddress,
			ReferralCode: userEntity.ReferralCode,
			ReferredBy:   userEntity.ReferredBy,
			CreatedAt:    userEntity.CreatedAt,
			UpdatedAt:    userEntity.UpdatedAt,
		}

		// Create sharded user
		if err := s.userRepo.CreateInShard(userEntity.ID, createdUser); err != nil {
			return err
		}

		// Remove from users table permanently (Unscoped = hard delete; source of truth is the shard only)
		if err := tx.Unscoped().Delete(userEntity).Error; err != nil {
			return err
		}

		// Create wallet for new user
		wallet := &entity.Wallet{
			UserID:      userEntity.ID,
			Balance:     0,
			TotalEarned: 0,
			TotalSpent:  0,
		}
		if err := tx.Create(wallet).Error; err != nil {
			return fmt.Errorf("failed to create wallet: %w", err)
		}

		// If user was referred, give referral reward to referrer
		if referredBy != nil {
			// Add credits to referrer's wallet
			if err := s.walletRepo.AddCredits(*referredBy, ReferralRewardCredits); err != nil {
				return fmt.Errorf("failed to add referral reward: %w", err)
			}

			// Create purchase record for referral reward
			referrerPurchase := &entity.PurchaseBase{
				UserID:       *referredBy,
				PurchaseType: entity.PurchaseTypeReferralReward,
				Status:       entity.PurchaseStatusCompleted,
				Credits:      ReferralRewardCredits,
				Notes:        fmt.Sprintf("Referral reward for user %d", userEntity.ID),
			}
			if err := s.purchaseRepo.CreateInShard(*referredBy, referrerPurchase); err != nil {
				return fmt.Errorf("failed to create referral purchase record: %w", err)
			}
		}

		// If user signed up with promo=freecredits, grant 20 credits (update wallet in same tx; AddCredits would use a new tx and not see the just-created wallet)
		if promoFreeCredits {
			wallet.Balance += SignupPromoCredits
			wallet.TotalEarned += SignupPromoCredits
			if err := tx.Save(wallet).Error; err != nil {
				return fmt.Errorf("failed to add signup promo credits: %w", err)
			}
			promoPurchase := &entity.PurchaseBase{
				UserID:       userEntity.ID,
				PurchaseType: entity.PurchaseTypeSignupPromo,
				Status:       entity.PurchaseStatusCompleted,
				Credits:      SignupPromoCredits,
				Notes:        "Signup promo free credits",
			}
			if err := s.purchaseRepo.CreateInShard(userEntity.ID, promoPurchase); err != nil {
				return fmt.Errorf("failed to create signup promo purchase record: %w", err)
			}
		}

		// Grant permissions based on account type
		if accountType == entity.AccountTypeDark {
			// Dark account: grant can_view_nsfw and can_search_tags permissions
			for _, permName := range []string{"can_view_nsfw", "can_search_tags"} {
				var permission entity.Permission
				if err := tx.Where("name = ?", permName).First(&permission).Error; err != nil {
					return fmt.Errorf("failed to get %s permission: %w", permName, err)
				}
				var existingUserPermission entity.UserPermission
				err := tx.Where("user_id = ? AND permission_id = ?", userEntity.ID, permission.ID).
					First(&existingUserPermission).Error
				if err != nil && err != gorm.ErrRecordNotFound {
					return fmt.Errorf("failed to check existing permission: %w", err)
				}
				if err == gorm.ErrRecordNotFound {
					userPermission := &entity.UserPermission{
						UserID:       userEntity.ID,
						PermissionID: permission.ID,
						ShardNumber:  entity.GetShardNumber(userEntity.ID),
					}
					if err := tx.Create(userPermission).Error; err != nil {
						return fmt.Errorf("failed to grant %s permission: %w", permName, err)
					}
				}
			}
		}
		// Light account: no additional permissions needed

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

	// If user is admin, ensure they have all permission types
	if user.AccountType == entity.AccountTypeAdmin {
		allPermissions, err := s.permissionRepo.GetSystemPermissions()
		if err != nil {
			fmt.Printf("Warning: failed to get system permissions for admin user %d: %v\n", user.ID, err)
		} else {
			for _, permission := range allPermissions {
				if err := s.userPermissionRepo.GrantPermission(user.ID, permission.ID); err != nil {
					fmt.Printf("Warning: failed to grant permission %s to admin user %d: %v\n", permission.Name, user.ID, err)
				}
			}
		}
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

	if s.emailService != nil {
		resetLink := getFrontendResetPasswordURL(token)
		if err := s.emailService.SendPasswordResetEmail(user.Email, resetLink); err != nil {
			log.Printf("Failed to send password reset email to %s: %v", user.Email, err)
			// 不向调用方返回错误，避免泄露该邮箱是否已注册
		}
	} else {
		fmt.Printf("Password reset token: %s (for testing only, set SMTP_* env to send email)\n", token)
	}

	return nil
}

// getFrontendResetPasswordURL 根据 FRONTEND_URL 生成重置密码链接（与 Stripe 等共用 FRONTEND_URL）
func getFrontendResetPasswordURL(token string) string {
	base := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
	if base == "" {
		base = "http://localhost:3000"
	}
	base = strings.TrimSuffix(base, "/")
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	return base + "/en/auth/reset-password?token=" + token
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

// GetUserPermissionNames 获取用户权限名称列表（仅来自 user_permissions 表）
func (s *authService) GetUserPermissionNames(userID uint) ([]string, error) {
	permissions, err := s.userPermissionRepo.GetUserPermissions(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	names := make([]string, 0, len(permissions))
	for _, p := range permissions {
		names = append(names, p.Name)
	}
	return names, nil
}

// GetWalletBalance 获取用户钱包余额（积分）
func (s *authService) GetWalletBalance(userID uint) (int64, error) {
	return s.walletRepo.GetBalance(userID)
}

// GetMembershipStatus 获取用户会员状态
func (s *authService) GetMembershipStatus(userID uint) (status string, expiresAt *time.Time) {
	purchase, err := s.purchaseRepo.GetLatestActiveMembership(userID)
	if err != nil || purchase == nil || purchase.ExpiresAt == nil {
		return "none", nil
	}
	return "active", purchase.ExpiresAt
}

// EnsureAdminSeed 若不存在任何 admin 账户则创建种子 admin；adminEmail/adminPassword 为空时跳过
func (s *authService) EnsureAdminSeed(adminEmail, adminPassword string) error {
	if adminEmail == "" || adminPassword == "" {
		return nil
	}
	count, err := s.userRepo.CountByAccountTypeInShards(entity.AccountTypeAdmin)
	if err != nil {
		return fmt.Errorf("count admin: %w", err)
	}
	if count > 0 {
		return nil
	}
	// 检查邮箱是否已被使用
	existing, _ := s.userRepo.GetByEmailFromShard(adminEmail)
	if existing != nil {
		log.Printf("EnsureAdminSeed: email %s already registered, skipping seed", adminEmail)
		return nil
	}
	hashedPassword, err := hashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	newReferralCode, err := s.generateReferralCode()
	if err != nil {
		return fmt.Errorf("generate referral code: %w", err)
	}
	user := &entity.UserBase{
		Email:        adminEmail,
		Username:     "twixter_user",
		Password:     hashedPassword,
		AccountType:  entity.AccountTypeAdmin,
		ReferralCode: newReferralCode,
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		userEntity := &entity.User{UserBase: *user}
		if err := tx.Create(userEntity).Error; err != nil {
			return err
		}
		createdUser := &entity.UserBase{
			ID:           userEntity.ID,
			Email:        userEntity.Email,
			Username:     userEntity.Username,
			Password:     userEntity.Password,
			AccountType:  userEntity.AccountType,
			IPAddress:    userEntity.IPAddress,
			ReferralCode: userEntity.ReferralCode,
			ReferredBy:   userEntity.ReferredBy,
			CreatedAt:    userEntity.CreatedAt,
			UpdatedAt:    userEntity.UpdatedAt,
		}
		if err := s.userRepo.CreateInShard(userEntity.ID, createdUser); err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(userEntity).Error; err != nil {
			return err
		}
		wallet := &entity.Wallet{
			UserID:      userEntity.ID,
			Balance:     0,
			TotalEarned: 0,
			TotalSpent:  0,
		}
		return tx.Create(wallet).Error
	})
	if err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}
	log.Printf("Seeded admin account: %s", adminEmail)
	return nil
}
