package controllers

import (
	"net/http"

	"enviroo-be/internal/services"

	"github.com/gin-gonic/gin"
)

type TabunganSampahController struct {
	Svc services.TabunganService
}

func NewTabunganSampahController(svc services.TabunganService) *TabunganSampahController {
	return &TabunganSampahController{Svc: svc}
}

func (tc *TabunganSampahController) GetBukuTabunganSampahNasabah(c *gin.Context) {
	buku, err := tc.Svc.BukuTabunganNasabah(
		c.Param("nasabah_id"),
		c.Query("start_date"),
		c.Query("end_date"),
	)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, buku)
}

func (tc *TabunganSampahController) GetBukuTabunganSampahBSU(c *gin.Context) {
	buku, err := tc.Svc.BukuTabunganBSU(
		c.Param("bsu_id"),
		c.Query("start_date"),
		c.Query("end_date"),
	)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, buku)
}
