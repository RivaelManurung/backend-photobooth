package handlers

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/middleware"
	"backendphotobooth/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TwoFAHandler struct {
	config *config.Config
}

func NewTwoFAHandler(cfg *config.Config) *TwoFAHandler {
	return &TwoFAHandler{config: cfg}
}

// SetupTwoFA initiates 2FA setup for user
func (h *TwoFAHandler) SetupTwoFA(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if 2FA already exists
	existingTwoFA, _ := models.GetUserTwoFactorAuth(database.DB, user.ID)
	if existingTwoFA != nil && existingTwoFA.IsEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is already enabled"})
		return
	}

	// Create or update 2FA settings
	var twoFA *models.TwoFactorAuth
	if existingTwoFA != nil {
		twoFA = existingTwoFA
	} else {
		twoFA = &models.TwoFactorAuth{
			UserID: user.ID,
		}
	}

	// Generate secret
	if err := twoFA.GenerateSecret(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate secret"})
		return
	}

	// Generate QR code URL
	issuer := "Photo Booth"
	accountName := user.Email
	qrCodeURL, err := twoFA.GenerateQRCode(issuer, accountName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
		return
	}

	// Generate backup codes
	backupCodes, err := twoFA.GenerateBackupCodes(8)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate backup codes"})
		return
	}

	// Save to database
	if existingTwoFA == nil {
		if err := models.CreateTwoFactorAuth(database.DB, twoFA); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save 2FA settings"})
			return
		}
	} else {
		database.DB.Save(twoFA)
	}

	c.JSON(http.StatusOK, gin.H{
		"secret":       twoFA.Secret,
		"qr_code_url":  qrCodeURL,
		"backup_codes": backupCodes,
		"message":      "Scan QR code with your authenticator app and verify with a code",
	})
}

// VerifyAndEnableTwoFA verifies code and enables 2FA
func (h *TwoFAHandler) VerifyAndEnableTwoFA(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get 2FA settings
	twoFA, err := models.GetUserTwoFactorAuth(database.DB, user.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "2FA not set up"})
		return
	}

	// Verify code
	if !twoFA.VerifyCode(req.Code) {
		// Log failed attempt
		models.CreateTwoFactorLog(database.DB, &models.TwoFactorLog{
			UserID:    user.ID,
			Action:    "enable",
			Status:    "failed",
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			ErrorMessage: "Invalid code",
		})

		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification code"})
		return
	}

	// Enable 2FA
	if err := twoFA.Enable(database.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable 2FA"})
		return
	}

	// Log success
	models.CreateTwoFactorLog(database.DB, &models.TwoFactorLog{
		UserID:    user.ID,
		Action:    "enable",
		Status:    "success",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "2FA enabled successfully",
		"enabled": true,
	})
}

// DisableTwoFA disables 2FA for user
func (h *TwoFAHandler) DisableTwoFA(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify password
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// Get 2FA settings
	twoFA, err := models.GetUserTwoFactorAuth(database.DB, user.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "2FA not enabled"})
		return
	}

	// Verify code
	if !twoFA.VerifyCode(req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification code"})
		return
	}

	// Disable 2FA
	if err := twoFA.Disable(database.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable 2FA"})
		return
	}

	// Log action
	models.CreateTwoFactorLog(database.DB, &models.TwoFactorLog{
		UserID:    user.ID,
		Action:    "disable",
		Status:    "success",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "2FA disabled successfully",
		"enabled": false,
	})
}

// VerifyTwoFA verifies 2FA code during login
func (h *TwoFAHandler) VerifyTwoFA(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Code  string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Get 2FA settings
	twoFA, err := models.GetUserTwoFactorAuth(database.DB, user.ID)
	if err != nil || !twoFA.IsEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA not enabled"})
		return
	}

	// Check if locked
	if twoFA.IsLocked() {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Too many failed attempts. Please try again later.",
		})
		return
	}

	// Verify code
	if !twoFA.VerifyCode(req.Code) {
		// Increment failed attempts
		twoFA.IncrementFailedAttempts(database.DB)

		// Log failed attempt
		models.CreateTwoFactorLog(database.DB, &models.TwoFactorLog{
			UserID:       user.ID,
			Action:       "verify",
			Status:       "failed",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			ErrorMessage: "Invalid code",
		})

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid verification code"})
		return
	}

	// Reset failed attempts
	twoFA.ResetFailedAttempts(database.DB)
	twoFA.UpdateLastUsed(database.DB)

	// Log success
	models.CreateTwoFactorLog(database.DB, &models.TwoFactorLog{
		UserID:    user.ID,
		Action:    "verify",
		Status:    "success",
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})

	// Generate tokens
	accessToken, err := middleware.GenerateToken(&user, h.config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken, err := middleware.GenerateRefreshToken(&user, h.config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

// GetTwoFAStatus returns 2FA status for user
func (h *TwoFAHandler) GetTwoFAStatus(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	twoFA, err := models.GetUserTwoFactorAuth(database.DB, user.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled":    false,
			"configured": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":     twoFA.IsEnabled,
		"configured":  true,
		"verified_at": twoFA.VerifiedAt,
		"last_used":   twoFA.LastUsedAt,
	})
}

// RegenerateBackupCodes regenerates backup codes
func (h *TwoFAHandler) RegenerateBackupCodes(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get 2FA settings
	twoFA, err := models.GetUserTwoFactorAuth(database.DB, user.ID)
	if err != nil || !twoFA.IsEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA not enabled"})
		return
	}

	// Verify code
	if !twoFA.VerifyCode(req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid verification code"})
		return
	}

	// Generate new backup codes
	backupCodes, err := twoFA.GenerateBackupCodes(8)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate backup codes"})
		return
	}

	database.DB.Save(twoFA)

	c.JSON(http.StatusOK, gin.H{
		"backup_codes": backupCodes,
		"message":      "Backup codes regenerated. Store them safely!",
	})
}
