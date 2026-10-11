package controllers

import (
	"net/http"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/utils"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	AuthSvc services.AuthService
	AkunSvc services.AkunService
}

func NewAuthController(authSvc services.AuthService, akunSvc services.AkunService) *AuthController {
	return &AuthController{AuthSvc: authSvc, AkunSvc: akunSvc}
}

func (ac *AuthController) Login(c *gin.Context) {
	var req struct {
		Email    string              `json:"email" binding:"required,email"`
		Password string              `json:"password" binding:"required,min=8"`
		Platform string              `json:"platform" binding:"required"`
		Role     models.RoleUserEnum `json:"role" binding:"required"`
		FCMToken string              `json:"fcm_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := ac.AuthSvc.Login(services.LoginInput{
		Email:    req.Email,
		Password: req.Password,
		Platform: req.Platform,
		Role:     req.Role,
		FCMToken: req.FCMToken,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	if req.Platform == models.PlatformWeb {
		utils.SetTokenCookies(c, result.AccessToken, result.RefreshToken)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login berhasil",
		"data":    result,
	})
}

func (ac *AuthController) RefreshToken(c *gin.Context) {
	refreshStr, _ := c.Cookie(utils.RefreshTokenCookieName)

	tokens, err := ac.AuthSvc.Refresh(refreshStr)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	utils.SetTokenCookies(c, tokens.AccessToken, tokens.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"message":       "Token berhasil diperbarui",
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

func (ac *AuthController) Logout(c *gin.Context) {
	ac.AuthSvc.Logout(utils.ExtractTokenFromRequest(c))

	utils.ClearTokenCookies(c)
	c.JSON(http.StatusOK, gin.H{"message": "Logout berhasil"})
}

func (ac *AuthController) Me(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	result, err := ac.AuthSvc.Me(claims)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (ac *AuthController) SwitchRole(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	var req struct {
		TargetRole models.RoleUserEnum `json:"target_role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := ac.AuthSvc.SwitchRole(claims, req.TargetRole)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Switch role berhasil",
		"data":    result,
	})
}

func (ac *AuthController) CekUserMobile(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := ac.AuthSvc.CekUserMobile(req.Email, req.Password)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User ditemukan",
		"data":    result,
	})
}

func (ac *AuthController) AktivasiAkun(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		OTP      string `json:"otp" binding:"required"`
		Password string `json:"password"`
		Platform string `json:"platform" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pesan, err := ac.AkunSvc.AktivasiAkun(services.AktivasiAkunInput{
		UserID:   req.UserID,
		OTP:      req.OTP,
		Password: req.Password,
		Platform: req.Platform,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": pesan})
}

func (ac *AuthController) GenerateReactivateAkun(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	var req struct {
		UserID  string              `json:"user_id" binding:"required"`
		Role    models.RoleUserEnum `json:"role" binding:"required"`
		RoleID  string              `json:"role_id" binding:"required"`
		AdminID string              `json:"admin_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := ac.AkunSvc.GenerateReaktivasi(services.GenerateReaktivasiInput{
		UserID:      req.UserID,
		Role:        req.Role,
		RoleID:      req.RoleID,
		AdminID:     req.AdminID,
		ActorBankID: claims.BankID,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Permintaan aktivasi ulang berhasil diproses. Informasi telah dikirimkan ke email user terkait.",
		"data":    result,
	})
}

func (ac *AuthController) DeactivateAkun(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	var req struct {
		UserID string              `json:"user_id" binding:"required"`
		Role   models.RoleUserEnum `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ac.AkunSvc.Deaktivasi(services.DeaktivasiInput{
		UserID:      req.UserID,
		Role:        req.Role,
		ActorBankID: claims.BankID,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Akun berhasil dinonaktifkan."})
}

func (ac *AuthController) ChangePassword(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	var req struct {
		PasswordLama           string `json:"password_lama" binding:"required"`
		PasswordBaru           string `json:"password_baru" binding:"required"`
		KonfirmasiPasswordBaru string `json:"konfirmasi_password_baru" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format input tidak valid"})
		return
	}

	err := ac.AkunSvc.ChangePassword(claims.UserID, services.ChangePasswordInput{
		PasswordLama:           req.PasswordLama,
		PasswordBaru:           req.PasswordBaru,
		KonfirmasiPasswordBaru: req.KonfirmasiPasswordBaru,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diubah"})
}

func (ac *AuthController) SendEmailForgetPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := ac.AkunSvc.KirimOTPResetPassword(req.Email)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Email OTP untuk reset password telah berhasil dikirimkan",
		"data":    result,
	})
}

func (ac *AuthController) VerifikasiOTP(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		OTP   string `json:"otp" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := ac.AkunSvc.VerifikasiOTPReset(req.Email, req.OTP)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Verifikasi OTP berhasil",
		"data":    result,
	})
}

func (ac *AuthController) ResetPassword(c *gin.Context) {
	var req struct {
		Email                  string `json:"email" binding:"required,email"`
		OTP                    string `json:"otp" binding:"required"`
		PasswordBaru           string `json:"password_baru" binding:"required"`
		KonfirmasiPasswordBaru string `json:"konfirmasi_password_baru" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ac.AkunSvc.ResetPassword(services.ResetPasswordInput{
		Email:                  req.Email,
		OTP:                    req.OTP,
		PasswordBaru:           req.PasswordBaru,
		KonfirmasiPasswordBaru: req.KonfirmasiPasswordBaru,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password berhasil diubah. Silakan login menggunakan password baru Anda.",
	})
}
