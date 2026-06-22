package controllers

import (
	"encoding/json"
	"net/http"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/storage"

	"github.com/gin-gonic/gin"
)

type PengangkutanController struct {
	Svc      services.PengangkutanService
	CFStorage *storage.CloudflareStorage
}

func NewPengangkutanController(svc services.PengangkutanService, cfStorage *storage.CloudflareStorage) *PengangkutanController {
	return &PengangkutanController{Svc: svc, CFStorage: cfStorage}
}

// GET /pengangkutan/check/:bsi_id
func (p *PengangkutanController) CheckJadwalPengangkutan(c *gin.Context) {
	rows, err := p.Svc.GetJadwalHariIni(c.Param("bsi_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total_jadwal_hari_ini": len(rows),
		"jadwal_hari_ini":       rows,
	})
}

// POST /pengangkutan/start
func (p *PengangkutanController) StartSesiPengangkutan(c *gin.Context) {
	var req struct {
		BSIID      string `json:"bsi_id"       binding:"required"`
		BSUID      string `json:"bsu_id"       binding:"required"`
		AdminBSIID string `json:"admin_bsi_id" binding:"required"`
		IsMandiri  bool   `json:"is_mandiri"`
		JadwalID   string `json:"jadwal_id"    binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pgk, err := p.Svc.StartSesi(services.StartSesiReq{
		BSIID:      req.BSIID,
		BSUID:      req.BSUID,
		AdminBSIID: req.AdminBSIID,
		IsMandiri:  req.IsMandiri,
		JadwalID:   req.JadwalID,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Sesi pengangkutan berhasil dimulai", "data": pgk})
}

// GET /pengangkutan/get-all/:bank_id
func (p *PengangkutanController) GetAllPengangkutan(c *gin.Context) {
	rows, err := p.Svc.GetList(c.Param("bank_id"), c.Query("start_date"), c.Query("end_date"))
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Data pengangkutan berhasil diambil", "data": rows})
}

// PATCH /pengangkutan/update/:pengangkutan_id/:admin_bsi_id
func (p *PengangkutanController) UpdatePengangkutanByBSI(c *gin.Context) {
	var req struct {
		CurrentStatus string `json:"current_status" binding:"required"`
		NewStatus     string `json:"new_status"     binding:"required"`
		Notes         string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	riwayat, err := p.Svc.UpdateStatus(
		c.Param("pengangkutan_id"),
		c.Param("admin_bsi_id"),
		services.UpdateStatusReq{
			CurrentStatus: models.StatusPengangkutan(req.CurrentStatus),
			NewStatus:     models.StatusPengangkutan(req.NewStatus),
			Notes:         req.Notes,
		},
	)
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Status pengangkutan berhasil diperbarui", "data": riwayat})
}

// POST /pengangkutan/request/:bsu_id/:admin_bsu_id
func (p *PengangkutanController) RequestPengangkutanByBSU(c *gin.Context) {
	var req struct {
		Tanggal  string `json:"tanggal"   binding:"required"`
		JamMulai string `json:"jam_mulai" binding:"required"`
		Notes    string `json:"notes"     binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pgk, err := p.Svc.RequestByBSU(
		c.Param("bsu_id"),
		c.Param("admin_bsu_id"),
		services.RequestBSUReq{Tanggal: req.Tanggal, JamMulai: req.JamMulai, Notes: req.Notes},
	)
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Permintaan pengangkutan berhasil diajukan", "data": pgk})
}

// POST /pengangkutan/preview/:pengangkutan_id
func (p *PengangkutanController) PreviewPengangkutanSampah(c *gin.Context) {
	itemsStr := c.PostForm("items")
	if itemsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data items tidak boleh kosong"})
		return
	}
	var items []services.ItemPengangkutanReq
	if err := json.Unmarshal([]byte(itemsStr), &items); err != nil || len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data items tidak valid atau kosong"})
		return
	}

	result, err := p.Svc.Preview(c.Param("pengangkutan_id"), items)
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Preview pengangkutan berhasil dihitung", "data": result})
}

// POST /pengangkutan/input
func (p *PengangkutanController) InputSampahPengangkutan(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal memproses form data"})
		return
	}

	qrData := c.PostForm("qr_data")
	if qrData == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "qr_data wajib diisi"})
		return
	}
	var qrPayload struct {
		Type           string `json:"type"`
		AdminBSUID     string `json:"admin_bsu_id"`
		PengangkutanID string `json:"pengangkutan_id"`
	}
	if err := json.Unmarshal([]byte(qrData), &qrPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format QR code tidak valid"})
		return
	}
	if qrPayload.Type != "ENVIROO-ANGKUTBSU" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "QR code bukan milik aplikasi ini"})
		return
	}

	adminBSIID := c.PostForm("admin_bsi_id")
	if adminBSIID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin_bsi_id wajib diisi"})
		return
	}

	itemsStr := c.PostForm("items")
	if itemsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data items tidak boleh kosong"})
		return
	}
	var items []services.ItemPengangkutanReq
	if err := json.Unmarshal([]byte(itemsStr), &items); err != nil || len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data items tidak valid atau kosong"})
		return
	}

	var buktiURL *string
	if file, header, err := c.Request.FormFile("bukti_foto"); err == nil {
		defer file.Close()
		url, errUpload := p.CFStorage.UploadFile(file, header, "pengangkutan_bukti")
		if errUpload != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload bukti foto: " + errUpload.Error()})
			return
		}
		buktiURL = &url
	}

	err := p.Svc.Input(qrPayload.PengangkutanID, adminBSIID, qrPayload.AdminBSUID, buktiURL, items)
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message":    "Sampah berhasil diinput ke sesi pengangkutan",
		"total_item": len(items),
	})
}

// GET /pengangkutan/detail-sampah/:pengangkutan_id
func (p *PengangkutanController) DetailSampahPengangkutan(c *gin.Context) {
	detail, err := p.Svc.GetDetailSampah(c.Param("pengangkutan_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Detail sampah pengangkutan berhasil diambil",
		"data":    gin.H{"header": detail.Header, "items": detail.Items},
	})
}

// GET /pengangkutan/list-sampah/:bsi_id/:bsu_id
func (p *PengangkutanController) ListSampahPengangkutan(c *gin.Context) {
	rows, err := p.Svc.GetSampahList(c.Param("bsi_id"), c.Param("bsu_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "List sampah berhasil diambil", "data": rows})
}

// GET /pengangkutan/check-sesi-active/:bsu_id
func (p *PengangkutanController) CheckSesiActivePengangkutan(c *gin.Context) {
	result, err := p.Svc.CheckSesiAktif(c.Param("bsu_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /pengangkutan/detail-sesi-active/:pengangkutan_id
func (p *PengangkutanController) DetailSesiActivePengangkutan(c *gin.Context) {
	result, err := p.Svc.GetDetailSesiAktif(c.Param("pengangkutan_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /pengangkutan/get-all-active/:bsi_id/:admin_id
func (p *PengangkutanController) GetAllActivePengangkutan(c *gin.Context) {
	result, err := p.Svc.GetAllAktif(c.Param("bsi_id"), c.Param("admin_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, result)
}

