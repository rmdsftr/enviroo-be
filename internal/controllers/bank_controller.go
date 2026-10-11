package controllers

import (
	"fmt"
	"net/http"

	"enviroo-be/internal/services"
	"enviroo-be/internal/adapters/storage"
	"enviroo-be/internal/utils"

	"github.com/gin-gonic/gin"
)

type BankController struct {
	Svc       services.BankService
	CFStorage *storage.CloudflareStorage
}

func NewBankController(svc services.BankService, cfStorage *storage.CloudflareStorage) *BankController {
	return &BankController{Svc: svc, CFStorage: cfStorage}
}

func (bc *BankController) AktivasiBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	var req struct {
		AdminID    string `json:"admin_id"`
		Informasi  string `json:"informasi"`
		Keterangan string `json:"keterangan"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format request tidak valid"})
		return
	}

	bank, err := bc.Svc.ToggleAktivasiBank(bankID, services.AktivasiBankInput{
		AdminID:    req.AdminID,
		Informasi:  req.Informasi,
		Keterangan: req.Keterangan,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	statusMsg := "diaktifkan"
	if !bank.IsActive {
		statusMsg = "dinonaktifkan"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank berhasil " + statusMsg,
		"data":    bank,
	})
}

func (bc *BankController) GetNasabahByBankID(c *gin.Context) {
	rows, meta, err := bc.Svc.ListNasabahBank(c.Param("bank_id"), utils.ParsePage(c.Query("page")))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	resp := gin.H{
		"message": "Nasabah fetched successfully",
		"data":    rows,
	}
	if meta != nil {
		resp["pagination"] = meta
	}
	c.JSON(http.StatusOK, resp)
}

func (bc *BankController) GetAllBankSampah(c *gin.Context) {
	banks, err := bc.Svc.ListBankAktif()
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All active banks fetched successfully",
		"data":    banks,
	})
}

func (bc *BankController) EditProfilBankSampah(c *gin.Context) {
	bankID := c.Param("bank_id")

	adminID := c.PostForm("admin_id")
	if adminID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Admin ID wajib diisi"})
		return
	}

	in := services.UpdateProfilBankInput{
		AdminID:   adminID,
		NamaBank:  c.PostForm("nama_bank"),
		Alamat:    c.PostForm("alamat"),
		Deskripsi: c.PostForm("deskripsi"),
		HapusFoto: c.PostForm("hapus_foto") == "true",
	}

	if latStr := c.PostForm("latitude"); latStr != "" {
		var latF float64
		if _, err := fmt.Sscanf(latStr, "%f", &latF); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format latitude tidak valid"})
			return
		}
		in.Latitude = &latF
	}
	if lngStr := c.PostForm("longitude"); lngStr != "" {
		var lngF float64
		if _, err := fmt.Sscanf(lngStr, "%f", &lngF); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format longitude tidak valid"})
			return
		}
		in.Longitude = &lngF
	}

	if fileHeader, err := c.FormFile("foto_profil"); err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file foto"})
			return
		}
		defer file.Close()

		photoURL, err := bc.CFStorage.UploadFile(file, fileHeader, "bank")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload foto profil: " + err.Error()})
			return
		}
		in.FotoURL = photoURL
	}

	bank, err := bc.Svc.UpdateProfilBank(bankID, in)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profil bank berhasil diperbarui",
		"data":    bank,
	})
}

func uploadFotoBank(c *gin.Context, cfStorage *storage.CloudflareStorage, folder string) (string, bool) {
	fileHeader, err := c.FormFile("foto")
	if err != nil {
		return "", true
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca foto"})
		return "", false
	}
	defer file.Close()

	fotoURL, err := cfStorage.UploadFile(file, fileHeader, folder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload foto ke cloud: " + err.Error()})
		return "", false
	}

	return fotoURL, true
}
