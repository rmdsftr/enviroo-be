package controllers

import (
	"log"
	"net/http"
	"time"

	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DistribusiSisaController struct {
	Svc      *services.DistribusiSisaService
	NotifSvc services.NotifikasiService
}

func NewDistribusiSisa(db *gorm.DB, notifSvc services.NotifikasiService) *DistribusiSisaController {
	repo := repositories.NewDistribusiSisaRepo(db)
	svc := services.NewDistribusiSisaService(db, repo)
	return &DistribusiSisaController{Svc: svc, NotifSvc: notifSvc}
}

func svcErr(c *gin.Context, err error) {
	if se, ok := err.(*services.ServiceError); ok {
		resp := gin.H{"error": se.Message}
		for k, v := range se.Data {
			resp[k] = v
		}
		c.JSON(se.Code, resp)
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

// GET /distribusi-sisa/konfigurasi/:bank_id
func (dsc *DistribusiSisaController) GetKonfigurasi(c *gin.Context) {
	result, err := dsc.Svc.GetKonfigurasi(c.Param("bank_id"))
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /distribusi-sisa/konfigurasi/:bank_id
func (dsc *DistribusiSisaController) AddKonfigurasi(c *gin.Context) {
	var req services.AddKonfigurasiReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := dsc.Svc.AddKonfigurasi(c.Param("bank_id"), req)
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Konfigurasi berhasil dibuat", "data": result})
}

// PATCH /distribusi-sisa/konfigurasi/:bank_id
func (dsc *DistribusiSisaController) UpdateKonfigurasi(c *gin.Context) {
	var req services.UpdateKonfigurasiReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := dsc.Svc.UpdateKonfigurasi(c.Param("bank_id"), req)
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Konfigurasi berhasil diperbarui", "data": result})
}

// GET /distribusi-sisa/preview/:bagi_hasil_id
func (dsc *DistribusiSisaController) PreviewDistribusiSisa(c *gin.Context) {
	result, err := dsc.Svc.PreviewDistribusiSisa(c.Param("bagi_hasil_id"))
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /distribusi-sisa/submit/:bagi_hasil_id
func (dsc *DistribusiSisaController) SubmitDistribusiSisa(c *gin.Context) {
	var req services.SubmitDistribusiReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := dsc.Svc.Submit(c.Param("bagi_hasil_id"), req)
	if err != nil {
		svcErr(c, err)
		return
	}

	// Kirim notifikasi ke admin tiap BSU penerima
	ctx := c.Request.Context()
	for _, item := range result.PenerimaBSU {
		pesanNilai := utils.FormatBagiHasilNilai(item.Nominal, result.Satuan)
		admins, err := dsc.Svc.GetAdminsByBankID(item.BankID)
		if err != nil {
			log.Printf("[DistribusiSisa] gagal ambil admin BSU %s: %v", item.BankID, err)
			continue
		}
		for _, adm := range admins {
			if err := dsc.NotifSvc.NotifBagiHasilDiterima(ctx,
				adm.UserID, adm.User.FCMToken,
				pesanNilai, item.NamaBank, result.NamaBSI,
				item.PenerimaSisaID,
			); err != nil {
				log.Printf("[DistribusiSisa] notif gagal untuk admin %s: %v", adm.AdminID, err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Distribusi sisa bagi hasil berhasil",
		"distribusi_id": result.DistribusiID,
		"bagi_hasil_id": result.BagiHasilID,
		"total_sisa":    result.TotalSisa,
		"satuan":        result.Satuan,
		"nominal_bsi":   result.NominalBSI,
		"penerima_bsu":  result.PenerimaBSU,
	})
}

// GET /distribusi-sisa/detail/:distribusi_id
func (dsc *DistribusiSisaController) GetDetailDistribusiSisa(c *gin.Context) {
	result, err := dsc.Svc.GetDetailDistribusiSisa(c.Param("distribusi_id"))
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /distribusi-sisa/list-bh-bank/:bank_id
func (dsc *DistribusiSisaController) ListBagiHasilBank(c *gin.Context) {
	var filter struct {
		StartDate string `form:"start_date"`
		EndDate   string `form:"end_date"`
	}
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format query param tidak valid"})
		return
	}

	const dateLayout = "2006-01-02"
	var startDate, endDate time.Time

	if filter.StartDate != "" {
		t, err := time.ParseInLocation(dateLayout, filter.StartDate, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format start_date tidak valid, gunakan YYYY-MM-DD"})
			return
		}
		startDate = t
	}
	if filter.EndDate != "" {
		t, err := time.ParseInLocation(dateLayout, filter.EndDate, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format end_date tidak valid, gunakan YYYY-MM-DD"})
			return
		}
		endDate = t.Add(24*time.Hour - time.Second)
	}
	if !startDate.IsZero() && !endDate.IsZero() && startDate.After(endDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date tidak boleh lebih besar dari end_date"})
		return
	}

	result, err := dsc.Svc.ListBagiHasilBank(c.Param("bank_id"), startDate, endDate)
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /distribusi-sisa/detail-bh-bank/:penerima_sisa_id
func (dsc *DistribusiSisaController) DetailBagiHasilBank(c *gin.Context) {
	result, err := dsc.Svc.DetailBagiHasilBank(c.Param("penerima_sisa_id"))
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
