package middleware

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const ClaimsKey = "claims"

func RequireAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		tokenStr := utils.ExtractTokenFromRequest(c)

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

		var sessionUser models.User
		if err := db.Select("device_session_id", "web_session_id").
			Where("user_id = ?", claims.UserID).
			First(&sessionUser).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Sesi tidak valid. Silakan login kembali.",
			})
			return
		}
		sessionID := sessionUser.SessionIDFor(claims.Platform)
		if sessionID == nil || *sessionID == "" || *sessionID != claims.SessionID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Sesi Anda telah berakhir karena akun digunakan di perangkat lain. Silakan login kembali.",
				"code":  "SESSION_REVOKED",
			})
			return
		}

		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

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

func RequireSameBankParam(paramName string, bypassRoles ...models.RoleAdmin) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
			return
		}
		if claims.BankID == "" {
			c.Next()
			return
		}
		for _, r := range bypassRoles {
			if claims.Role == r {
				c.Next()
				return
			}
		}
		if c.Param(paramName) != claims.BankID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Akses ditolak. Anda tidak memiliki akses ke bank sampah ini."})
			return
		}
		c.Next()
	}
}

func RequireSameUser(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
			return
		}
		if c.Param(paramName) != claims.UserID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Akses ditolak. Anda hanya dapat mengakses akun Anda sendiri."})
			return
		}
		c.Next()
	}
}

func RequireSameNasabah(db *gorm.DB, paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
			return
		}

		if claims.Role != models.RoleNasabah {
			c.Next()
			return
		}
		var count int64
		db.Model(&models.Nasabah{}).
			Where("nasabah_id = ? AND user_id = ?", c.Param(paramName), claims.UserID).
			Count(&count)
		if count == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Akses ditolak. Anda tidak memiliki akses ke data nasabah ini."})
			return
		}
		c.Next()
	}
}

func RequireNasabahBankAccess(db *gorm.DB, paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
			return
		}

		var nasabah models.Nasabah
		if err := db.Select("nasabah_id", "user_id", "bank_id").
			Where("nasabah_id = ?", c.Param(paramName)).First(&nasabah).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
			return
		}

		if claims.Role == models.RoleNasabah {
			if nasabah.UserID != claims.UserID {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Akses ditolak. Anda tidak memiliki akses ke data nasabah ini."})
				return
			}
			c.Next()
			return
		}

		if nasabah.BankID != claims.BankID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Akses ditolak. Anda tidak memiliki akses ke data nasabah ini."})
			return
		}
		c.Next()
	}
}

func GetClaims(c *gin.Context) (*utils.JWTClaims, bool) {
	raw, exists := c.Get(ClaimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := raw.(*utils.JWTClaims)
	return claims, ok
}
