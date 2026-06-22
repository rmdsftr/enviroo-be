package controllers

import (
	"net/http"
	"strconv"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"

	"github.com/gin-gonic/gin"
)

type NotifikasiController struct {
	Svc services.NotifikasiService
}

func NewNotifikasiController(svc services.NotifikasiService) *NotifikasiController {
	return &NotifikasiController{Svc: svc}
}

// roleTarget menentukan role_target berdasarkan role JWT:
// nasabah → "nasabah", semua admin/petugas → "admin"
func roleTarget(role models.RoleAdmin) string {
	if role == models.RoleNasabah {
		return "nasabah"
	}
	return "admin"
}

// GET /notifikasi/list?page=1&limit=20
func (nc *NotifikasiController) GetList(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := nc.Svc.GetByUser(c.Request.Context(), claims.UserID, roleTarget(claims.Role), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil notifikasi",
		"data":    result.Data,
		"meta": gin.H{
			"total":         result.Total,
			"page":          result.Page,
			"limit":         result.Limit,
			"total_halaman": result.TotalHalaman,
		},
	})
}

// PATCH /notifikasi/read/:notifikasi_id
func (nc *NotifikasiController) MarkAsRead(c *gin.Context) {
	notifikasiID := c.Param("notifikasi_id")

	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	if err := nc.Svc.BacaNotif(c.Request.Context(), notifikasiID, claims.UserID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notifikasi ditandai sudah dibaca"})
}

// PATCH /notifikasi/read-all
func (nc *NotifikasiController) MarkAllAsRead(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	if err := nc.Svc.BacaSemua(c.Request.Context(), claims.UserID, roleTarget(claims.Role)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Semua notifikasi ditandai sudah dibaca"})
}

// GET /notifikasi/unread-count
func (nc *NotifikasiController) UnreadCount(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	count, err := nc.Svc.JumlahBelumDibaca(c.Request.Context(), claims.UserID, roleTarget(claims.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"unread_count": count}})
}
