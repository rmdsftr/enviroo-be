package utils

import (
	"enviroo-be/internal/models"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"

	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
)

// JWTClaims berisi payload yang disimpan di dalam JWT.
type JWTClaims struct {
	UserID string           `json:"user_id"`
	Role   models.RoleAdmin `json:"role"`
	BankID string           `json:"bank_id"` // kosong untuk superadmin
	jwt.RegisteredClaims
}

// ─── Generate ────────────────────────────────────────────────────────────────

// GenerateJWT membuat short-lived access token (15 menit).
func GenerateJWT(userID string, role models.RoleAdmin, bankID string) (string, error) {
	return generateToken(userID, role, bankID, AccessTokenDuration)
}

// GenerateRefreshToken membuat long-lived refresh token (7 hari).
func GenerateRefreshToken(userID string, role models.RoleAdmin, bankID string) (string, error) {
	return generateToken(userID, role, bankID, RefreshTokenDuration)
}

func generateToken(userID string, role models.RoleAdmin, bankID string, duration time.Duration) (string, error) {
	secret := getSecret()
	claims := JWTClaims{
		UserID: userID,
		Role:   role,
		BankID: bankID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "enviroo",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ─── Parse ───────────────────────────────────────────────────────────────────

// ParseJWT memvalidasi token string dan mengembalikan claims-nya.
func ParseJWT(tokenStr string) (*JWTClaims, error) {
	secret := getSecret()
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token tidak valid")
	}
	return claims, nil
}

// ─── Cookie helpers (Gin) ────────────────────────────────────────────────────

// SetTokenCookies menulis access_token dan refresh_token sebagai HttpOnly cookie
// via Gin context.
func SetTokenCookies(c *gin.Context, accessToken, refreshToken string) {
	isProduction := os.Getenv("ENVIRONMENT") == "production"
	sameSite := http.SameSiteNoneMode
	if !isProduction {
		// Di local dev (cross-origin FE:5173 → BE:8080) pakai Lax agar cookie dikirim.
		sameSite = http.SameSiteLaxMode
	}
	c.SetSameSite(sameSite)

	c.SetCookie(
		AccessTokenCookieName,
		accessToken,
		int(AccessTokenDuration.Seconds()),
		"/",
		"",
		isProduction,
		true, // HttpOnly
	)

	c.SetSameSite(sameSite)
	c.SetCookie(
		RefreshTokenCookieName,
		refreshToken,
		int(RefreshTokenDuration.Seconds()),
		"/auth/refresh",
		"",
		isProduction,
		true, // HttpOnly
	)
}

// ClearTokenCookies menghapus kedua cookie dari browser.
func ClearTokenCookies(c *gin.Context) {
	c.SetCookie(AccessTokenCookieName, "", -1, "/", "", false, true)
	c.SetCookie(RefreshTokenCookieName, "", -1, "/auth/refresh", "", false, true)
}

// ─── Internal ────────────────────────────────────────────────────────────────

// getSecret mengambil JWT secret dari environment.
// Kehadiran & kekuatan secret sudah divalidasi saat startup di main.go,
// sehingga di sini tidak ada fallback lemah.
func getSecret() string {
	return os.Getenv("JWT_SECRET")
}
