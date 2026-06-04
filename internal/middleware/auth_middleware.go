package middleware

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const ClaimsKey = "claims"

// RequireAuth memvalidasi access_token lalu memverifikasi status akun ke database.
// Jika akun nonaktif/pending, request ditolak dengan 403 dan code ACCOUNT_INACTIVE.
func RequireAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Coba ambil dari cookie (untuk Web)
		tokenStr, err := c.Cookie(utils.AccessTokenCookieName)

		// 2. Jika tidak ada di cookie, coba ambil dari Authorization header (untuk Mobile)
		if err != nil || tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenStr = authHeader[7:]
			}
		}

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Sesi tidak ditemukan. Silakan login terlebih dahulu.",
			})
			return
		}

		claims, err := utils.ParseJWT(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Sesi tidak valid atau sudah berakhir. Silakan login kembali.",
			})
			return
		}

		// 3. Cek status akun di database
		if claims.Role == models.RoleNasabah {
			var count int64
			db.Model(&models.Nasabah{}).
				Where("user_id = ? AND status_nasabah = ?", claims.UserID, models.Aktif).
				Count(&count)
			if count == 0 {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Akun Anda tidak aktif. Silakan hubungi administrator.",
					"code":  "ACCOUNT_INACTIVE",
				})
				return
			}
		} else {
			var count int64
			db.Model(&models.Admin{}).
				Where("user_id = ? AND role = ? AND status_admin = ?", claims.UserID, claims.Role, models.Aktif).
				Count(&count)
			if count == 0 {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Akun Anda tidak aktif. Silakan hubungi administrator.",
					"code":  "ACCOUNT_INACTIVE",
				})
				return
			}
		}

		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

// RequireRole memastikan user yang sudah terautentikasi memiliki salah satu role yang diizinkan.
// Harus dipasang setelah RequireAuth.
func RequireRole(roles ...models.RoleAdmin) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, exists := c.Get(ClaimsKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
			return
		}

		claims, ok := raw.(*utils.JWTClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		for _, allowed := range roles {
			if claims.Role == allowed {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Akses ditolak. Anda tidak memiliki izin untuk endpoint ini.",
		})
	}
}

// GetClaims adalah helper untuk mengambil claims dari context Gin.
func GetClaims(c *gin.Context) (*utils.JWTClaims, bool) {
	raw, exists := c.Get(ClaimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := raw.(*utils.JWTClaims)
	return claims, ok
}
