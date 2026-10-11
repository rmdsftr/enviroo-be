package controllers

import (
	"net/http"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/adapters/storage"
	"enviroo-be/internal/utils"

	"github.com/gin-gonic/gin"
)

type BSIController struct {
	Svc       services.BankService
	CFStorage *storage.CloudflareStorage
}

func NewBSIController(svc services.BankService, cfStorage *storage.CloudflareStorage) *BSIController {
	return &BSIController{Svc: svc, CFStorage: cfStorage}
}

type AddBSIRequest struct {
	NamaBSI       string   `form:"nama_bsi" binding:"required"`
	Deskripsi     string   `form:"deskripsi" binding:"required"`
	Provinsi      string   `form:"provinsi" binding:"required"`
	KabupatenKota string   `form:"kabupaten_kota" binding:"required"`
	IDKecamatan   int      `form:"id_kecamatan" binding:"required"`
	IDKelurahan   int      `form:"id_kelurahan" binding:"required"`
	AlamatLengkap string   `form:"alamat_lengkap" binding:"required"`
	Latitude      float64  `form:"latitude"`
	Longitude     float64  `form:"longitude"`
	UserIDs       []string `form:"user_id[]"`
	AdminID       string   `form:"admin_id" binding:"required"`
}

func (bc *BSIController) AddNewBSI(c *gin.Context) {
	var req AddBSIRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fotoURL, ok := uploadFotoBank(c, bc.CFStorage, "bsi_photos")
	if !ok {
		return
	}

	newBSI, err := bc.Svc.CreateBank(services.CreateBankInput{
		JenisBank:     models.BSI,
		Nama:          req.NamaBSI,
		Deskripsi:     req.Deskripsi,
		Provinsi:      req.Provinsi,
		KabupatenKota: req.KabupatenKota,
		IDKecamatan:   req.IDKecamatan,
		IDKelurahan:   req.IDKelurahan,
		Alamat:        req.AlamatLengkap,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		FotoURL:       fotoURL,
		UserIDs:       req.UserIDs,
		ActorAdminID:  req.AdminID,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "BSI and admins created successfully",
		"data":    newBSI,
	})
}

func (bc *BSIController) GetBSI(c *gin.Context) {
	rows, meta, err := bc.Svc.ListBSI(utils.ParsePage(c.Query("page")))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	resp := gin.H{
		"message": "BSI fetched successfully",
		"data":    rows,
	}
	if meta != nil {
		resp["pagination"] = meta
	}
	c.JSON(http.StatusOK, resp)
}

func (bc *BSIController) GetUnitBSI(c *gin.Context) {
	rows, meta, err := bc.Svc.ListUnitBSI(c.Param("bank_id"), utils.ParsePage(c.Query("page")))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	resp := gin.H{
		"message": "BSU fetched successfully",
		"data":    rows,
	}
	if meta != nil {
		resp["pagination"] = meta
	}
	c.JSON(http.StatusOK, resp)
}

type AddUnitRequest struct {
	NamaUnit      string   `form:"nama_unit" binding:"required"`
	Deskripsi     string   `form:"deskripsi" binding:"required"`
	Provinsi      string   `form:"provinsi" binding:"required"`
	KabupatenKota string   `form:"kabupaten_kota" binding:"required"`
	IDKecamatan   int      `form:"id_kecamatan" binding:"required"`
	IDKelurahan   int      `form:"id_kelurahan" binding:"required"`
	AlamatLengkap string   `form:"alamat_lengkap" binding:"required"`
	Latitude      float64  `form:"latitude"`
	Longitude     float64  `form:"longitude"`
	UserIDs       []string `form:"user_id[]"`
	AdminID       string   `form:"admin_id" binding:"required"`
}

func (bc *BSIController) AddNewUnit(c *gin.Context) {
	bankID := c.Param("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank ID is required"})
		return
	}

	var req AddUnitRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fotoURL, ok := uploadFotoBank(c, bc.CFStorage, "bsu_photos")
	if !ok {
		return
	}

	newBSU, err := bc.Svc.CreateBank(services.CreateBankInput{
		JenisBank:     models.BSU,
		ParentBankID:  &bankID,
		Nama:          req.NamaUnit,
		Deskripsi:     req.Deskripsi,
		Provinsi:      req.Provinsi,
		KabupatenKota: req.KabupatenKota,
		IDKecamatan:   req.IDKecamatan,
		IDKelurahan:   req.IDKelurahan,
		Alamat:        req.AlamatLengkap,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		FotoURL:       fotoURL,
		UserIDs:       req.UserIDs,
		ActorAdminID:  req.AdminID,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "BSU and admins created successfully",
		"data":    newBSU,
	})
}
