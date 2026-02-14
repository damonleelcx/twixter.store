package controller

import (
	"backend/entity"
	"backend/middleware"
	"backend/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthController Authentication controller
type AuthController struct {
	authService service.AuthService
}

// NewAuthController Create authentication controller instance
func NewAuthController(authService service.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

// RegisterRequest Registration request
type RegisterRequest struct {
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required,min=8"`
	Username         string `json:"username,omitempty"`           // 用户名（可选）
	ReferralCode     string `json:"referral_code,omitempty"`      // 推荐码（可选）
	Viewing          string `json:"viewing,omitempty"`             // 明文查看类型（可选，与 viewing_token 二选一）
	ViewingToken     string `json:"viewing_token,omitempty"`      // 加密的 viewing token（可选，解密后用于账户类型）
	PromoFreeCredits bool   `json:"promo_freecredits,omitempty"`   // 是否通过 promo=freecredits 注册（注册即送 20 积分）
}

// LoginRequest Login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest Refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RequestPasswordResetRequest Password reset request
type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest Reset password request
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePasswordRequest Change password request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// Register Register a new user
// @Summary Register a new user
// @Description Register a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /api/auth/register [post]
func (ac *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	ipAddress := c.ClientIP()

	// 根据 viewing_token（解密）或 viewing（明文）确定账户类型
	viewing := req.Viewing
	if req.ViewingToken != "" {
		viewing = middleware.DecryptViewingToken(req.ViewingToken)
	}
	if viewing != "viewing_dark" && viewing != "viewing_light" {
		viewing = "viewing_light"
	}
	var accountType entity.AccountType
	if viewing == "viewing_dark" {
		accountType = entity.AccountTypeDark
	} else {
		accountType = entity.AccountTypeLight
	}

	user, session, err := ac.authService.Register(req.Email, req.Password, ipAddress, req.Username, req.ReferralCode, accountType, req.PromoFreeCredits)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "email already registered" {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user": gin.H{
			"id":            user.ID,
			"email":         user.Email,
			"username":      user.Username,
			"account_type":  user.AccountType,
			"referral_code": user.ReferralCode,
		},
		"session": gin.H{
			"access_token":             session.AccessToken,
			"refresh_token":            session.RefreshToken,
			"token_type":               session.TokenType,
			"access_token_expires_at":  session.AccessTokenExpiresAt,
			"refresh_token_expires_at": session.RefreshTokenExpiresAt,
		},
	})
}

// Login User login
// @Summary User login
// @Description Authenticate user and return session tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 423 {object} map[string]interface{}
// @Router /api/auth/login [post]
func (ac *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	deviceInfo := c.GetHeader("X-Device-Info")

	user, session, err := ac.authService.Login(req.Email, req.Password, ipAddress, userAgent, deviceInfo)
	if err != nil {
		statusCode := http.StatusUnauthorized
		if err.Error() == "account is locked, please try again later" {
			statusCode = http.StatusLocked
		}
		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user": gin.H{
			"id":            user.ID,
			"email":         user.Email,
			"username":      user.Username,
			"account_type":  user.AccountType,
			"referral_code": user.ReferralCode,
		},
		"session": gin.H{
			"access_token":             session.AccessToken,
			"refresh_token":            session.RefreshToken,
			"token_type":               session.TokenType,
			"access_token_expires_at":  session.AccessTokenExpiresAt,
			"refresh_token_expires_at": session.RefreshTokenExpiresAt,
		},
	})
}

// Logout Logout (revoke current session)
// @Summary Logout current session
// @Description Revoke the current user session
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/auth/logout [post]
func (ac *AuthController) Logout(c *gin.Context) {
	session, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	sess := session.(*entity.Session)
	if err := ac.authService.Logout(sess.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to logout",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

// LogoutAll Logout all sessions
// @Summary Logout all user sessions
// @Description Revoke all sessions for the current user
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/auth/logout-all [post]
func (ac *AuthController) LogoutAll(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	userBase := user.(*entity.UserBase)
	if err := ac.authService.LogoutAll(userBase.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to logout all sessions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All sessions logged out successfully",
	})
}

// RefreshToken Refresh access token
// @Summary Refresh access token
// @Description Refresh the access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/auth/refresh [post]
func (ac *AuthController) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	session, err := ac.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		statusCode := http.StatusUnauthorized
		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"session": gin.H{
			"access_token":             session.AccessToken,
			"refresh_token":            session.RefreshToken,
			"token_type":               session.TokenType,
			"access_token_expires_at":  session.AccessTokenExpiresAt,
			"refresh_token_expires_at": session.RefreshTokenExpiresAt,
		},
	})
}

// RequestPasswordReset Request password reset
// @Summary Request password reset
// @Description Request a password reset token via email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RequestPasswordResetRequest true "Password reset request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/auth/password/reset/request [post]
func (ac *AuthController) RequestPasswordReset(c *gin.Context) {
	var req RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	if err := ac.authService.RequestPasswordReset(req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to request password reset",
		})
		return
	}

	// Always return success for security (don't reveal if email exists)
	c.JSON(http.StatusOK, gin.H{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword Reset password
// @Summary Reset password
// @Description Reset password using reset token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Reset password request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/auth/password/reset [post]
func (ac *AuthController) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	if err := ac.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "reset token invalid" || err.Error() == "reset token expired" {
			statusCode = http.StatusUnauthorized
		}
		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
	})
}

// ChangePassword Change password
// @Summary Change password
// @Description Change user password (requires authentication)
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequest true "Change password request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/auth/password/change [post]
func (ac *AuthController) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	userBase := user.(*entity.UserBase)
	if err := ac.authService.ChangePassword(userBase.ID, req.OldPassword, req.NewPassword); err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "old password incorrect" {
			statusCode = http.StatusUnauthorized
		}
		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// GetCurrentUser Get current user information
// @Summary Get current user
// @Description Get current authenticated user information
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/auth/me [get]
func (ac *AuthController) GetCurrentUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	userBase := user.(*entity.UserBase)
	permissionNames, _ := ac.authService.GetUserPermissionNames(userBase.ID)
	walletBalance, _ := ac.authService.GetWalletBalance(userBase.ID)
	membershipStatus, membershipExpiresAt := ac.authService.GetMembershipStatus(userBase.ID)
	userPayload := gin.H{
		"id":                userBase.ID,
		"email":             userBase.Email,
		"username":          userBase.Username,
		"account_type":      userBase.AccountType,
		"email_verified":    userBase.EmailVerified,
		"referral_code":     userBase.ReferralCode,
		"created_at":        userBase.CreatedAt,
		"permissions":       permissionNames,
		"wallet_balance":    walletBalance,
		"membership_status": membershipStatus,
	}
	if membershipExpiresAt != nil {
		userPayload["membership_expires_at"] = membershipExpiresAt.Format("2006-01-02T15:04:05Z07:00")
	} else {
		userPayload["membership_expires_at"] = nil
	}
	userPayload["shareable_link_claimed"] = userBase.ShareableLinkClaimedAt != nil
	c.JSON(http.StatusOK, gin.H{"user": userPayload})
}

// GetViewingTokenForReferral 为推荐链接领取 viewing token（每人仅可领取一次，需登录）
func (ac *AuthController) GetViewingTokenForReferral(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userBase := user.(*entity.UserBase)
	mode := c.DefaultQuery("mode", string(userBase.AccountType))
	if mode != "dark" {
		mode = "light"
	}
	token, alreadyClaimed, err := ac.authService.ClaimViewingTokenForReferral(userBase.ID, mode)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	if alreadyClaimed {
		c.JSON(http.StatusForbidden, gin.H{"error": "already_shared", "message": "You can only share your link once"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}
