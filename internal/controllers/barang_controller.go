package controllers

import (
	"net/http"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/services"
	"enviroo-be/internal/adapters/storage"
	"enviroo-be/internal/utils"

	"github.com/gin-gonic/gin"
)

type BarangController struct {
	Svc       services.BarangService
	CFStorage *storage.CloudflareStorage
}

func NewBarangController(svc services.BarangService, cfStorage *storage.CloudflareStorage) *BarangController {
	return &BarangController{Svc: svc, CFStorage: cfStorage}
}

func aksesBarang(c *gin.Context) (services.AksesBarangInput, bool) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return services.AksesBarangInput{}, false
	}
	return services.AksesBarangInput{ActorBankID: claims.BankID, ActorRole: claims.Role}, true
}

func (sc *BarangController) uploadFotoBarang(c *gin.Context) (url *string, ok bool) {
	fileHeader, err := c.FormFile("foto")
	if err != nil {
		return nil, true
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca foto"})
		return nil, false
	}
	defer file.Close()

	fotoURL, err := sc.CFStorage.UploadFile(file, fileHeader, "katalog_barang")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload foto: " + err.Error()})
		return nil, false
	}
	return &fotoURL, true
}

type itemDistribusiReq struct {
	ProdukID  string  `json:"produk_id" binding:"required"`
	StokKirim float64 `json:"stok" binding:"required,gt=0"`
}

func toItemsDistribusi(items []itemDistribusiReq) []services.ItemDistribusiInput {
	hasil := make([]services.ItemDistribusiInput, 0, len(items))
	for _, item := range items {
		hasil = append(hasil, services.ItemDistribusiInput{
			ProdukID:  item.ProdukID,
			StokKirim: item.StokKirim,
		})
	}
	return hasil
}

func (sc *BarangController) GetMasterBarang(c *gin.Context) {
	results, err := sc.Svc.ListMasterBarang(c.Query("q"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}

func (sc *BarangController) AddNewBarang(c *gin.Context) {
	var req struct {
		NamaBarang string  `form:"nama_barang" binding:"required"`
		BarangID   int     `form:"barang_id"`
		NilaiPoin  float64 `form:"nilai_poin" binding:"required"`
		StokAwal   float64 `form:"stok_awal"`
		CreatedBy  string  `form:"created_by"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fotoURL, ok := sc.uploadFotoBarang(c)
	if !ok {
		return
	}

	result, err := sc.Svc.TambahBarang(services.TambahBarangInput{
		BankID:     c.Param("bank_id"),
		NamaBarang: req.NamaBarang,
		BarangID:   req.BarangID,
		NilaiPoin:  req.NilaiPoin,
		StokAwal:   req.StokAwal,
		CreatedBy:  req.CreatedBy,
		FotoURL:    fotoURL,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Barang berhasil ditambahkan",
		"data":    result,
	})
}

func (sc *BarangController) GetBarangBank(c *gin.Context) {
	rows, meta, err := sc.Svc.ListBarangBank(c.Param("bank_id"), utils.ParsePage(c.Query("page")))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	resp := gin.H{
		"message": "Katalog barang berhasil diambil",
		"data":    rows,
	}
	if meta != nil {
		resp["pagination"] = meta
	}
	c.JSON(http.StatusOK, resp)
}

func (sc *BarangController) GetDetailBarang(c *gin.Context) {
	akses, ok := aksesBarang(c)
	if !ok {
		return
	}

	riwayat, err := sc.Svc.RiwayatDistribusiBarang(c.Param("produk_id"), c.Query("bank_id"), akses)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail barang berhasil diambil",
		"data": gin.H{
			"riwayat_distribusi": riwayat,
		},
	})
}

func (sc *BarangController) EditBarang(c *gin.Context) {
	akses, ok := aksesBarang(c)
	if !ok {
		return
	}

	var req struct {
		NilaiPoin  float64 `form:"nilai_poin"`
		Stok       float64 `form:"stok"`
		TambahStok float64 `form:"tambah_stok"`
		UpdatedBy  string  `form:"updated_by"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fotoURL, ok := sc.uploadFotoBarang(c)
	if !ok {
		return
	}

	barang, err := sc.Svc.EditBarang(c.Param("produk_id"), services.EditBarangInput{
		NilaiPoin:    req.NilaiPoin,
		SetNilaiPoin: c.PostForm("nilai_poin") != "",
		Stok:         req.Stok,
		SetStok:      c.PostForm("stok") != "",
		TambahStok:   req.TambahStok,
		UpdatedBy:    req.UpdatedBy,
		FotoURL:      fotoURL,
		Akses:        akses,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Barang berhasil diupdate",
		"data":    barang,
	})
}

func (sc *BarangController) DeleteBarang(c *gin.Context) {
	akses, ok := aksesBarang(c)
	if !ok {
		return
	}

	err := sc.Svc.HapusBarang(services.HapusBarangInput{
		ProdukID: c.Param("produk_id"),
		Akses:    akses,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil dihapus"})
}

func (sc *BarangController) QRDistribusiBarang(c *gin.Context) {
	akses, ok := aksesBarang(c)
	if !ok {
		return
	}

	var req struct {
		BSIID      string              `json:"bsi_id" binding:"required"`
		BSUID      string              `json:"bsu_id" binding:"required"`
		AdminBSIID string              `json:"admin_bsi_id" binding:"required"`
		Items      []itemDistribusiReq `json:"items" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	disbaID, err := sc.Svc.BuatSesiDistribusi(services.BuatSesiDistribusiInput{
		BSIID:      req.BSIID,
		BSUID:      req.BSUID,
		AdminBSIID: req.AdminBSIID,
		Items:      toItemsDistribusi(req.Items),
		Akses:      akses,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{"disba_id": disbaID})
}

func (sc *BarangController) PreviewDistribusiBarangBSU(c *gin.Context) {
	akses, ok := aksesBarang(c)
	if !ok {
		return
	}

	var req struct {
		Items []itemDistribusiReq `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	previews, validationErrors, err := sc.Svc.PreviewDistribusi(services.PreviewDistribusiInput{
		BSIID: c.Param("bsi_id"),
		BSUID: c.Param("bsu_id"),
		Items: toItemsDistribusi(req.Items),
		Akses: akses,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Preview gagal: ada item yang tidak valid",
			"errors":  validationErrors,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Preview distribusi berhasil",
		"data":    previews,
	})
}

func (sc *BarangController) AddNewDistribusiBarangBSU(c *gin.Context) {
	akses, ok := aksesBarang(c)
	if !ok {
		return
	}

	var req struct {
		DisbaID    string `json:"disba_id" binding:"required"`
		BSUID      string `json:"bsu_id" binding:"required"`
		AdminBSUID string `json:"admin_bsu_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := sc.Svc.SelesaikanDistribusi(services.SelesaikanDistribusiInput{
		DisbaID:    req.DisbaID,
		BSUID:      req.BSUID,
		AdminBSUID: req.AdminBSUID,
		Akses:      akses,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Distribusi barang ke BSU berhasil dilakukan"})
}

func (sc *BarangController) ListDistribusiBarang(c *gin.Context) {
	rows, err := sc.Svc.ListDistribusi(c.Param("bank_id"), c.Query("start_date"), c.Query("end_date"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data distribusi barang berhasil diambil",
		"data":    rows,
	})
}

func (sc *BarangController) GetDetailDistribusiBarang(c *gin.Context) {
	akses, ok := aksesBarang(c)
	if !ok {
		return
	}

	detail, err := sc.Svc.DetailDistribusi(c.Param("disba_id"), akses)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail distribusi barang berhasil diambil",
		"data":    detail,
	})
}
