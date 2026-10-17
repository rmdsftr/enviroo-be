package utils_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const secretUji = "rahasia-jwt-uji"

func konteksUji(req *http.Request) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if req == nil {
		req = httptest.NewRequest(http.MethodGet, "/", nil)
	}
	c.Request = req
	return c, w
}

func cookieDari(t *testing.T, w *httptest.ResponseRecorder, nama string) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == nama {
			return c
		}
	}
	t.Fatalf("cookie %s tidak dipasang", nama)
	return nil
}

func tandaTangani(t *testing.T, metode jwt.SigningMethod, kunci interface{}, klaim utils.JWTClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(metode, klaim).SignedString(kunci)
	if err != nil {
		t.Fatalf("gagal membuat token uji: %v", err)
	}
	return token
}

// ===== Generate & Parse =====

func TestGenerateJWT_ClaimsTerbaca(t *testing.T) {
	t.Setenv("JWT_SECRET", secretUji)

	token, err := utils.GenerateJWT("USR-001", models.AdminBSI, "BSI-001", "web", "SES-001")
	if !assert.NoError(t, err) {
		return
	}
	klaim, err := utils.ParseJWT(token)

	assert.NoError(t, err)
	if assert.NotNil(t, klaim) {
		assert.Equal(t, "USR-001", klaim.UserID)
		assert.Equal(t, models.AdminBSI, klaim.Role)
		assert.Equal(t, "BSI-001", klaim.BankID)
		assert.Equal(t, "web", klaim.Platform)
		assert.Equal(t, "SES-001", klaim.SessionID)
		assert.Equal(t, "enviroo", klaim.Issuer)
		assert.WithinDuration(t, time.Now(), klaim.IssuedAt.Time, 5*time.Second)
		assert.WithinDuration(t, time.Now().Add(15*time.Minute), klaim.ExpiresAt.Time, 5*time.Second)
	}
}

func TestGenerateRefreshToken_Berlaku7Hari(t *testing.T) {
	t.Setenv("JWT_SECRET", secretUji)

	token, err := utils.GenerateRefreshToken("USR-001", models.RoleNasabah, "BSM-001", "mobile", "SES-002")
	if !assert.NoError(t, err) {
		return
	}
	klaim, err := utils.ParseJWT(token)

	assert.NoError(t, err)
	if assert.NotNil(t, klaim) {
		assert.Equal(t, models.RoleNasabah, klaim.Role)
		assert.Equal(t, "mobile", klaim.Platform)
		assert.WithinDuration(t, time.Now().Add(7*24*time.Hour), klaim.ExpiresAt.Time, 5*time.Second)
	}
}

func TestParseJWT_SecretBeda(t *testing.T) {
	t.Setenv("JWT_SECRET", secretUji)
	token, err := utils.GenerateJWT("USR-001", models.AdminBSI, "BSI-001", "web", "SES-001")
	if !assert.NoError(t, err) {
		return
	}

	t.Setenv("JWT_SECRET", "secret-lain")
	klaim, err := utils.ParseJWT(token)

	assert.Nil(t, klaim)
	assert.ErrorIs(t, err, jwt.ErrTokenSignatureInvalid)
}

func TestParseJWT_Kadaluarsa(t *testing.T) {
	t.Setenv("JWT_SECRET", secretUji)
	token := tandaTangani(t, jwt.SigningMethodHS256, []byte(secretUji), utils.JWTClaims{
		UserID: "USR-001",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})

	klaim, err := utils.ParseJWT(token)

	assert.Nil(t, klaim)
	assert.ErrorIs(t, err, jwt.ErrTokenExpired)
}

func TestParseJWT_AlgoritmaBukanHMAC(t *testing.T) {
	t.Setenv("JWT_SECRET", secretUji)
	token := tandaTangani(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, utils.JWTClaims{
		UserID: "USR-001",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	klaim, err := utils.ParseJWT(token)

	assert.Nil(t, klaim)
	assert.ErrorContains(t, err, "unexpected signing method", "token tanpa tanda tangan harus ditolak")
}

func TestParseJWT_FormatRusak(t *testing.T) {
	t.Setenv("JWT_SECRET", secretUji)

	for _, token := range []string{"", "bukan-jwt", "a.b.c"} {
		klaim, err := utils.ParseJWT(token)

		assert.Nil(t, klaim, token)
		assert.Error(t, err, token)
	}
}

// ===== Cookie =====

func TestSetTokenCookies_Development(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")
	c, w := konteksUji(nil)

	utils.SetTokenCookies(c, "akses-123", "refresh-456")

	akses := cookieDari(t, w, utils.AccessTokenCookieName)
	assert.Equal(t, "akses-123", akses.Value)
	assert.Equal(t, "/", akses.Path)
	assert.Equal(t, 900, akses.MaxAge)
	assert.True(t, akses.HttpOnly)
	assert.False(t, akses.Secure)
	assert.Equal(t, http.SameSiteLaxMode, akses.SameSite)

	refresh := cookieDari(t, w, utils.RefreshTokenCookieName)
	assert.Equal(t, "refresh-456", refresh.Value)
	assert.Equal(t, "/auth/refresh", refresh.Path, "refresh token hanya dikirim ke endpoint refresh")
	assert.Equal(t, 604800, refresh.MaxAge)
	assert.True(t, refresh.HttpOnly)
	assert.False(t, refresh.Secure)
	assert.Equal(t, http.SameSiteLaxMode, refresh.SameSite)
}

func TestSetTokenCookies_Production(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	c, w := konteksUji(nil)

	utils.SetTokenCookies(c, "akses-123", "refresh-456")

	for _, nama := range []string{utils.AccessTokenCookieName, utils.RefreshTokenCookieName} {
		cookie := cookieDari(t, w, nama)
		assert.True(t, cookie.Secure, nama)
		assert.True(t, cookie.HttpOnly, nama)
		assert.Equal(t, http.SameSiteNoneMode, cookie.SameSite, nama)
	}
}

func TestClearTokenCookies(t *testing.T) {
	c, w := konteksUji(nil)

	utils.ClearTokenCookies(c)

	akses := cookieDari(t, w, utils.AccessTokenCookieName)
	assert.Empty(t, akses.Value)
	assert.Less(t, akses.MaxAge, 0, "cookie langsung kadaluarsa")
	assert.Equal(t, "/", akses.Path)

	refresh := cookieDari(t, w, utils.RefreshTokenCookieName)
	assert.Empty(t, refresh.Value)
	assert.Less(t, refresh.MaxAge, 0, "cookie langsung kadaluarsa")
	assert.Equal(t, "/auth/refresh", refresh.Path)
}

// ===== ExtractTokenFromRequest =====

func TestExtractTokenFromRequest_DariCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: utils.AccessTokenCookieName, Value: "akses-123"})
	c, _ := konteksUji(req)

	assert.Equal(t, "akses-123", utils.ExtractTokenFromRequest(c))
}

func TestExtractTokenFromRequest_CookieDiutamakan(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: utils.AccessTokenCookieName, Value: "dari-cookie"})
	req.Header.Set("Authorization", "Bearer dari-header")
	c, _ := konteksUji(req)

	assert.Equal(t, "dari-cookie", utils.ExtractTokenFromRequest(c))
}

func TestExtractTokenFromRequest_DariHeaderBearer(t *testing.T) {
	t.Run("tanpa cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer xyz")
		c, _ := konteksUji(req)

		assert.Equal(t, "xyz", utils.ExtractTokenFromRequest(c))
	})

	t.Run("cookie kosong", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: utils.AccessTokenCookieName, Value: ""})
		req.Header.Set("Authorization", "Bearer xyz")
		c, _ := konteksUji(req)

		assert.Equal(t, "xyz", utils.ExtractTokenFromRequest(c))
	})
}

func TestExtractTokenFromRequest_TidakAda(t *testing.T) {
	kasus := map[string]string{
		"tanpa cookie dan header": "",
		"bearer tanpa token":      "Bearer ",
		"skema bukan bearer":      "Token xyz",
		"bearer huruf kecil":      "bearer xyz",
	}
	for nama, header := range kasus {
		t.Run(nama, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			c, _ := konteksUji(req)

			assert.Equal(t, "", utils.ExtractTokenFromRequest(c))
		})
	}
}
