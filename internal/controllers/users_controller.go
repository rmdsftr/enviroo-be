package controllers

import (
	"net/http"
	"strconv"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/services"
	"enviroo-be/internal/adapters/storage"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	Svc       services.UserService
	AkunSvc   services.AkunService
	CFStorage *storage.CloudflareStorage
}

func NewUserController(svc services.UserService, akunSvc services.AkunService, cfStorage *storage.CloudflareStorage) *UserController {
	return &UserController{Svc: svc, AkunSvc: akunSvc, CFStorage: cfStorage}
}

type AddUserRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	Nama       string `json:"nama" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	NoWhatsapp string `json:"no_whatsapp" binding:"required"`
}

func (uc *UserController) AddUser(c *gin.Context) {
	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := uc.Svc.TambahUser(services.TambahUserInput{
		UserID:     req.UserID,
		Nama:       req.Nama,
		Email:      req.Email,
		NoWhatsapp: req.NoWhatsapp,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"data":    user,
	})
}

func (uc *UserController) UpdateProfilUser(c *gin.Context) {
	userID := c.Param("user_id")

	in := services.UpdateProfilUserInput{
		Nama:       c.PostForm("nama"),
		NoWhatsapp: c.PostForm("no_whatsapp"),
		HapusFoto:  c.PostForm("delete_photo") == "true",
	}

	if fileHeader, err := c.FormFile("photo_profile"); err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file foto"})
			return
		}
		defer file.Close()

		photoURL, err := uc.CFStorage.UploadFile(file, fileHeader, "profile")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload foto profil: " + err.Error()})
			return
		}
		in.FotoURL = photoURL
		in.AdaFileFoto = true
	}

	user, err := uc.Svc.UpdateProfil(userID, in)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profil berhasil diperbarui",
		"data":    user,
	})
}

func (uc *UserController) GetNonAdminUser(c *gin.Context) {
	users, err := uc.Svc.ListNonAdmin()
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Users fetched successfully",
		"data":    users,
	})
}

func (uc *UserController) GetNonNasabahUser(c *gin.Context) {
	users, err := uc.Svc.ListNonNasabah()
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Users fetched successfully",
		"data":    users,
	})
}

func (uc *UserController) ActiveAdmin(c *gin.Context) {
	result, err := uc.Svc.GetActiveAdmin(c.Param("admin_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Current active Admin fetch successfully",
		"data":    result,
	})
}

func (uc *UserController) ActivePetugas(c *gin.Context) {
	result, err := uc.Svc.GetActivePetugas(c.Param("admin_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data petugas aktif berhasil diambil",
		"data":    result,
	})
}

func (uc *UserController) ActiveUser(c *gin.Context) {
	result, err := uc.Svc.GetUserRingkas(c.Param("user_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data user berhasil diambil",
		"data":    result,
	})
}

type AddSuperadminRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	Nama       string `json:"nama" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	NoWhatsapp string `json:"no_whatsapp" binding:"required"`
}

func (uc *UserController) AddSuperadmin(c *gin.Context) {
	var req AddSuperadminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := uc.Svc.TambahSuperadmin(services.TambahSuperadminInput{
		UserID:     req.UserID,
		Nama:       req.Nama,
		Email:      req.Email,
		NoWhatsapp: req.NoWhatsapp,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Superadmin berhasil ditambahkan",
		"data":    result,
	})
}

func (uc *UserController) GetListSuperadmin(c *gin.Context) {
	list, err := uc.Svc.ListSuperadmin()
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Daftar superadmin berhasil diambil",
		"data":    list,
	})
}

func (uc *UserController) NonaktifkanSuperadmin(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	targetAdminID := c.Param("admin_id")
	err := uc.AkunSvc.NonaktifkanSuperadmin(services.NonaktifkanSuperadminInput{
		TargetAdminID: targetAdminID,
		ActorUserID:   claims.UserID,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Superadmin berhasil dinonaktifkan",
		"admin_id": targetAdminID,
	})
}

func (uc *UserController) UpdateFCMToken(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	var req struct {
		FCMToken string `json:"fcm_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := handleServiceError(c, uc.Svc.UpdateFCMToken(claims.UserID, req.FCMToken)); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "FCM Token berhasil diperbarui",
	})
}

func (uc *UserController) LogAkun(c *gin.Context) {
	result, err := uc.Svc.GetLogAkun(c.Param("user_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Log akun berhasil diambil",
		"data":    result,
	})
}

func (uc *UserController) GetDetailUser(c *gin.Context) {
	result, err := uc.Svc.GetDetailUser(c.Param("user_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail user berhasil diambil",
		"data":    result,
	})
}

func (uc *UserController) DeleteUser(c *gin.Context) {
	if err := handleServiceError(c, uc.Svc.HapusUser(c.Param("user_id"))); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User berhasil dihapus"})
}

func (uc *UserController) GetAllUsers(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		page = 1
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if err != nil {
		limit = 25
	}

	items, meta, err := uc.Svc.ListSemuaUser(page, limit)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       items,
		"pagination": meta,
	})
}
