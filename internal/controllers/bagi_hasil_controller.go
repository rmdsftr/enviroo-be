package controllers

import (
	"context"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BagiHasilController struct {
	Svc      *services.BagiHasilService
	NotifSvc services.NotifikasiService
}

func NewBagiHasilController(db *gorm.DB, notifSvc services.NotifikasiService) *BagiHasilController {
	repo := repositories.NewBagiHasilRepo(db)
	svc := services.NewBagiHasilService(db, repo)
	return &BagiHasilController{Svc: svc, NotifSvc: notifSvc}
}

// POST /bagi-hasil/preview/:penjualan_id/:bank_id
func (bhc *BagiHasilController) PreviewHitungBagiHasil(c *gin.Context) {
	result, err := bhc.Svc.Preview(c.Param("penjualan_id"), c.Param("bank_id"))
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /bagi-hasil/submit/:penjualan_id/:bank_id
func (bhc *BagiHasilController) SubmitBagiHasil(c *gin.Context) {
	var req struct {
		AdminID string `json:"admin_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := bhc.Svc.Submit(c.Param("penjualan_id"), c.Param("bank_id"), req.AdminID)
	if err != nil {
		svcErr(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                  "Bagi hasil berhasil dilakukan",
		"bagi_hasil_id":            result.BagiHasilID,
		"gross_bank":               result.GrossBank,
		"total_distribusi_nasabah": result.TotalDistribusiNasabah,
		"sisa_bagi_hasil":          result.SisaBagiHasil,
	})

	notifSvc := bhc.NotifSvc
	notifs := result.NasabahNotifs
	go func() {
		for _, nd := range notifs {
			pesanNilai := utils.FormatBagiHasilNilai(nd.Total, nd.Satuan)
			if err := notifSvc.NotifBagiHasilNasabahDiterima(
				context.Background(),
				nd.UserID,
				nd.FCMToken,
				pesanNilai,
				nd.PenerimaID,
			); err != nil {
				fmt.Printf("[Notif] Gagal kirim bagi hasil nasabah ke user %s: %v\n", nd.UserID, err)
			}
		}
	}()
}

// GET /bagi-hasil/detail/:penjualan_id
func (bhc *BagiHasilController) GetDetailBagiHasil(c *gin.Context) {
	result, err := bhc.Svc.GetDetail(c.Param("penjualan_id"))
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /bagi-hasil/list-bh-nasabah/:nasabah_id
func (bhc *BagiHasilController) GetListBagiHasilPerNasabah(c *gin.Context) {
	result, err := bhc.Svc.GetListNasabah(c.Param("nasabah_id"), c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /bagi-hasil/detail-bh-nasabah/:penerima_id
func (bhc *BagiHasilController) GetDetailBagiHasilNasabah(c *gin.Context) {
	result, err := bhc.Svc.GetDetailNasabah(c.Param("penerima_id"))
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /bagi-hasil/list-bh-bsu/:bsu_id — deprecated, redirected to distribusi-sisa
func (bhc *BagiHasilController) GetListBagiHasilPerBsu(c *gin.Context) {
	c.JSON(http.StatusGone, gin.H{
		"error": "Endpoint ini sudah tidak digunakan. Gunakan GET /distribusi-sisa untuk melihat bagi hasil BSU.",
	})
}

// GET /bagi-hasil/detail-bh-bsu/:penerima_id — deprecated, redirected to distribusi-sisa
func (bhc *BagiHasilController) GetDetailBagiHasilBSU(c *gin.Context) {
	c.JSON(http.StatusGone, gin.H{
		"error": "Endpoint ini sudah tidak digunakan. Gunakan GET /distribusi-sisa/detail/:distribusi_id untuk melihat detail bagi hasil BSU.",
	})
}

// GET /bagi-hasil/list-bh-bank/:bank_id
func (bhc *BagiHasilController) GetListBagiHasilBankPusat(c *gin.Context) {
	var filter struct {
		StartDate string `form:"start_date"`
		EndDate   string `form:"end_date"`
	}
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format query param tidak valid"})
		return
	}

	const dateLayout = "2006-01-02"
	var startTime, endTime time.Time

	if filter.StartDate != "" {
		t, err := time.ParseInLocation(dateLayout, filter.StartDate, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format start_date tidak valid, gunakan YYYY-MM-DD"})
			return
		}
		startTime = t
	}
	if filter.EndDate != "" {
		t, err := time.ParseInLocation(dateLayout, filter.EndDate, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format end_date tidak valid, gunakan YYYY-MM-DD"})
			return
		}
		endTime = t.Add(24*time.Hour - time.Second)
	}
	if !startTime.IsZero() && !endTime.IsZero() && startTime.After(endTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date tidak boleh lebih besar dari end_date"})
		return
	}

	result, err := bhc.Svc.GetListBankPusat(c.Param("bank_id"), startTime, endTime)
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /bagi-hasil/detail-bh-bank/:bagi_hasil_id
func (bhc *BagiHasilController) GetDetailBagiHasilBankPusat(c *gin.Context) {
	result, err := bhc.Svc.GetDetailBankPusat(c.Param("bagi_hasil_id"))
	if err != nil {
		svcErr(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
