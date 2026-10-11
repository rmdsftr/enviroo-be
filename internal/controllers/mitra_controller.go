package controllers

import (
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MitraController struct {
	Svc *services.MitraService
}

func NewMitraController(db *gorm.DB) *MitraController {
	repo := repositories.NewMitraRepo(db)
	svc := services.NewMitraService(db, repo)
	return &MitraController{Svc: svc}
}

func (mc *MitraController) ListMitra(c *gin.Context) {
	mitra, err := mc.Svc.ListMitra(c.Param("bank_id"))
	if err != nil {
		svcErr2(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "List mitra berhasil diambil", "data": mitra})
}

func (mc *MitraController) CreateMitra(c *gin.Context) {
	var req struct {
		NamaMitra string `json:"nama_mitra" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama_mitra wajib diisi"})
		return
	}

	mitra, err := mc.Svc.CreateMitra(c.Param("bank_id"), req.NamaMitra)
	if err != nil {
		svcErr2(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Mitra berhasil dibuat", "data": mitra})
}

func (mc *MitraController) UpdateMitra(c *gin.Context) {
	var req struct {
		NamaMitra *string `json:"nama_mitra"`
		IsActive  *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mitra, err := mc.Svc.UpdateMitra(c.Param("bank_id"), c.Param("mitra_id"), req.NamaMitra, req.IsActive)
	if err != nil {
		svcErr2(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Mitra berhasil diupdate", "data": mitra})
}

func (mc *MitraController) DeleteMitra(c *gin.Context) {
	if err := mc.Svc.DeleteMitra(c.Param("bank_id"), c.Param("mitra_id")); err != nil {
		svcErr2(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Mitra berhasil dihapus"})
}
