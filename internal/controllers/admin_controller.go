package controllers

import (
	"fmt"
	"net/http"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/services"

	"github.com/gin-gonic/gin"
)

type AdminController struct {
	Svc services.BankService
}

func NewAdminController(svc services.BankService) *AdminController {
	return &AdminController{Svc: svc}
}

func (ac *AdminController) GetAdminBankSampah(c *gin.Context) {
	rows, err := ac.Svc.ListAdminBank(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Admin retrieved successfully",
		"data":    rows,
	})
}

func (ac *AdminController) AddAdminBankSampah(c *gin.Context) {
	var req struct {
		UserID  string `json:"user_id" binding:"required"`
		BankID  string `json:"bank_id" binding:"required"`
		Role    string `json:"role" binding:"required"`
		AdminID string `json:"admin_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newAdmin, err := ac.Svc.AddAdminBank(services.AddAdminBankInput{
		UserID:       req.UserID,
		BankID:       req.BankID,
		Role:         req.Role,
		ActorAdminID: req.AdminID,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin created successfully and activation sent",
		"data":    newAdmin,
	})
}

func (ac *AdminController) DeleteStaffBankSampah(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	result, err := ac.Svc.DeleteStaffBank(c.Param("admin_id"), claims.UserID)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Staff %s berhasil dihapus dari %s", result.StaffName, result.BankName),
	})
}
