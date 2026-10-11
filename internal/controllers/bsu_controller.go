package controllers

import (
	"net/http"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/adapters/storage"
	"enviroo-be/internal/utils"

	"github.com/gin-gonic/gin"
)

type BSUController struct {
	Svc       services.BankService
	CFStorage *storage.CloudflareStorage
}

func NewBSUController(svc services.BankService, cfStorage *storage.CloudflareStorage) *BSUController {
	return &BSUController{Svc: svc, CFStorage: cfStorage}
}

type AddBSURequest struct {
	NamaBSU       string   `form:"nama_bsu" binding:"required"`
	ParentBankID  string   `form:"parent_bank_id" binding:"required"`
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

func (bc *BSUController) AddNewBSU(c *gin.Context) {
	var req AddBSURequest
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
		ParentBankID:  &req.ParentBankID,
		Nama:          req.NamaBSU,
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

func (bc *BSUController) GetBSU(c *gin.Context) {
	rows, meta, err := bc.Svc.ListBSU(utils.ParsePage(c.Query("page")))
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

func (bc *BSUController) GetBSUbyBankID(c *gin.Context) {
	rows, err := bc.Svc.ListBSUByParent(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "BSU fetched successfully",
		"data":    rows,
	})
}
