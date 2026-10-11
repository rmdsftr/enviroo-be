package controllers

import (
	"net/http"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/adapters/storage"
	"enviroo-be/internal/utils"

	"github.com/gin-gonic/gin"
)

type BSMController struct {
	Svc       services.BankService
	CFStorage *storage.CloudflareStorage
}

func NewBSMController(svc services.BankService, cfStorage *storage.CloudflareStorage) *BSMController {
	return &BSMController{Svc: svc, CFStorage: cfStorage}
}

type AddBSMRequest struct {
	NamaBSM       string   `form:"nama_bsm" binding:"required"`
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

func (bc *BSMController) AddNewBSM(c *gin.Context) {
	var req AddBSMRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fotoURL, ok := uploadFotoBank(c, bc.CFStorage, "bsm_photos")
	if !ok {
		return
	}

	newBSM, err := bc.Svc.CreateBank(services.CreateBankInput{
		JenisBank:     models.BSM,
		Nama:          req.NamaBSM,
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
		"message": "BSM and admins created successfully",
		"data":    newBSM,
	})
}

func (bc *BSMController) GetBSM(c *gin.Context) {
	rows, meta, err := bc.Svc.ListBSM(utils.ParsePage(c.Query("page")))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	resp := gin.H{
		"message": "BSM fetched successfully",
		"data":    rows,
	}
	if meta != nil {
		resp["pagination"] = meta
	}
	c.JSON(http.StatusOK, resp)
}
