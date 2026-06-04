package controllers

import (
	"net/http"
	"strconv"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/services"

	"github.com/gin-gonic/gin"
)

type NotifikasiController struct {
	Svc services.NotifikasiService
}

func NewNotifikasiController(svc services.NotifikasiService) *NotifikasiController {
	return &NotifikasiController{Svc: svc}
}

// GET /notifikasi/list/:user_id?page=1&limit=20
func (nc *NotifikasiController) GetList(c *gin.Context) {
	userID := c.Param("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := nc.Svc.GetByUser(c.Request.Context(), userID, page, limit)
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
// Butuh RequireAuth — user_id diambil dari JWT claims, bukan dari request body/query.
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

// PATCH /notifikasi/read-all/:user_id
func (nc *NotifikasiController) MarkAllAsRead(c *gin.Context) {
	userID := c.Param("user_id")

	if err := nc.Svc.BacaSemua(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Semua notifikasi ditandai sudah dibaca"})
}

// GET /notifikasi/unread-count/:user_id
func (nc *NotifikasiController) UnreadCount(c *gin.Context) {
	userID := c.Param("user_id")

	count, err := nc.Svc.JumlahBelumDibaca(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"unread_count": count}})
}
