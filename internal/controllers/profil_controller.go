package controllers

import (
	"net/http"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/services"

	"github.com/gin-gonic/gin"
)

type ProfilController struct {
	BankSvc    services.BankService
	NasabahSvc services.NasabahService
	AkunSvc    services.AkunService
	UserSvc    services.UserService
}

func NewProfilController(
	bankSvc services.BankService,
	nasabahSvc services.NasabahService,
	akunSvc services.AkunService,
	userSvc services.UserService,
) *ProfilController {
	return &ProfilController{
		BankSvc:    bankSvc,
		NasabahSvc: nasabahSvc,
		AkunSvc:    akunSvc,
		UserSvc:    userSvc,
	}
}

func (pc *ProfilController) GetProfilBankSampah(c *gin.Context) {
	profil, err := pc.BankSvc.GetProfilBank(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank sampah profile fetched successfully",
		"data":    profil,
	})
}

func (pc *ProfilController) GetHistoryAkunBank(c *gin.Context) {
	rows, err := pc.BankSvc.GetHistoryAkunBank(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "History bank fetched successfully",
		"data":    rows,
	})
}

func (pc *ProfilController) DeleteBankSampah(c *gin.Context) {
	if err := handleServiceError(c, pc.BankSvc.DeleteBank(c.Param("bank_id"))); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank sampah deleted successfully",
	})
}

func (pc *ProfilController) DetailBankSampah(c *gin.Context) {
	detail, err := pc.BankSvc.GetDetailBank(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil detail bank sampah",
		"data":    detail,
	})
}

func (pc *ProfilController) GetProfilNasabah(c *gin.Context) {
	profil, err := pc.NasabahSvc.GetProfilNasabah(c.Param("nasabah_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah profile fetched successfully",
		"data":    profil,
	})
}

func (pc *ProfilController) GetDetailNasabah(c *gin.Context) {
	detail, err := pc.NasabahSvc.GetDetailNasabah(c.Param("nasabah_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail nasabah berhasil diambil",
		"data":    detail,
	})
}

func (pc *ProfilController) AktivasiNasabah(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	nasabah, err := pc.AkunSvc.ToggleStatusNasabah(services.ToggleStatusNasabahInput{
		NasabahID:   c.Param("nasabah_id"),
		ActorBankID: claims.BankID,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah status updated successfully",
		"data":    nasabah,
	})
}

func (pc *ProfilController) GetProfilUser(c *gin.Context) {
	profil, err := pc.UserSvc.GetProfilUser(c.Param("user_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User profile fetched successfully",
		"data":    profil,
	})
}

func (pc *ProfilController) GetDetailPetugas(c *gin.Context) {
	detail, err := pc.UserSvc.GetDetailPetugas(c.Param("petugas_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail petugas berhasil diambil",
		"data":    detail,
	})
}
