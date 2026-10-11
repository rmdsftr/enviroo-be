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

type JWTClaims struct {
	UserID    string           `json:"user_id"`
	Role      models.RoleAdmin `json:"role"`
	BankID    string           `json:"bank_id"`
	Platform  string           `json:"platform"`
	SessionID string           `json:"session_id"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID string, role models.RoleAdmin, bankID, platform, sessionID string) (string, error) {
	return generateToken(userID, role, bankID, platform, sessionID, AccessTokenDuration)
}

func GenerateRefreshToken(userID string, role models.RoleAdmin, bankID, platform, sessionID string) (string, error) {
	return generateToken(userID, role, bankID, platform, sessionID, RefreshTokenDuration)
}

func generateToken(userID string, role models.RoleAdmin, bankID, platform, sessionID string, duration time.Duration) (string, error) {
	secret := getSecret()
	claims := JWTClaims{
		UserID:    userID,
		Role:      role,
		BankID:    bankID,
		Platform:  platform,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "enviroo",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}


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

func SetTokenCookies(c *gin.Context, accessToken, refreshToken string) {
	isProduction := os.Getenv("ENVIRONMENT") == "production"
	sameSite := http.SameSiteNoneMode
	if !isProduction {
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
		true,
	)

	c.SetSameSite(sameSite)
	c.SetCookie(
		RefreshTokenCookieName,
		refreshToken,
		int(RefreshTokenDuration.Seconds()),
		"/auth/refresh",
		"",
		isProduction,
		true,
	)
}

func ClearTokenCookies(c *gin.Context) {
	c.SetCookie(AccessTokenCookieName, "", -1, "/", "", false, true)
	c.SetCookie(RefreshTokenCookieName, "", -1, "/auth/refresh", "", false, true)
}

func ExtractTokenFromRequest(c *gin.Context) string {
	tokenStr, err := c.Cookie(AccessTokenCookieName)
	if err == nil && tokenStr != "" {
		return tokenStr
	}

	authHeader := c.GetHeader("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}

	return ""
}

func getSecret() string {
	return os.Getenv("JWT_SECRET")
}
