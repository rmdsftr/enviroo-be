package controllers

import (
	"net/http"

	"enviroo-be/internal/services"

	"github.com/gin-gonic/gin"
)

type NasabahController struct {
	Svc services.NasabahService
	BankSvc services.BankService
}

func NewNasabahController(svc services.NasabahService, bankSvc services.BankService) *NasabahController {
	return &NasabahController{Svc: svc, BankSvc: bankSvc}
}

type AddNasabahRequest struct {
	UserID      string  `json:"user_id" binding:"required"`
	Nama        string  `json:"nama" binding:"required"`
	Email       string  `json:"email" binding:"required,email"`
	NoWhatsapp  string  `json:"no_whatsapp" binding:"required"`
	BankID      string  `json:"bank_id" binding:"required"`
	AdminID     string  `json:"admin_id" binding:"required"`
	NoRekening  string  `json:"no_rekening"`
	SaldoRupiah float64 `json:"saldo_rupiah"`
	SaldoPoin   float64 `json:"saldo_poin"`
}

func (nc *NasabahController) AddNewNasabah(c *gin.Context) {
	var req AddNasabahRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nasabah, err := nc.Svc.DaftarNasabahBaru(services.DaftarNasabahBaruInput{
		UserID:      req.UserID,
		Nama:        req.Nama,
		Email:       req.Email,
		NoWhatsapp:  req.NoWhatsapp,
		BankID:      req.BankID,
		AdminID:     req.AdminID,
		NoRekening:  req.NoRekening,
		SaldoRupiah: req.SaldoRupiah,
		SaldoPoin:   req.SaldoPoin,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Nasabah created successfully. Activation email sent.",
		"data":    nasabah,
	})
}

type AddNasabahOldUserRequest struct {
	UserID      string  `json:"user_id" binding:"required"`
	BankID      string  `json:"bank_id" binding:"required"`
	AdminID     string  `json:"admin_id" binding:"required"`
	NoRekening  string  `json:"no_rekening"`
	SaldoRupiah float64 `json:"saldo_rupiah"`
	SaldoPoin   float64 `json:"saldo_poin"`
}

func (nc *NasabahController) AddNewNasabahOldUser(c *gin.Context) {
	var req AddNasabahOldUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nasabah, err := nc.Svc.DaftarNasabahUserLama(services.DaftarNasabahUserLamaInput{
		UserID:      req.UserID,
		BankID:      req.BankID,
		AdminID:     req.AdminID,
		NoRekening:  req.NoRekening,
		SaldoRupiah: req.SaldoRupiah,
		SaldoPoin:   req.SaldoPoin,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Nasabah created successfully. Activation email sent.",
		"data":    nasabah,
	})
}

func (nc *NasabahController) GetAfiliasi(c *gin.Context) {
	banks, err := nc.BankSvc.ListSemuaBank()
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Afiliasi retrieved successfully",
		"data":    banks,
	})
}

func (nc *NasabahController) DeleteNasabah(c *gin.Context) {
	if err := handleServiceError(c, nc.Svc.HapusNasabah(c.Param("nasabah_id"))); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Nasabah deleted successfully"})
}
