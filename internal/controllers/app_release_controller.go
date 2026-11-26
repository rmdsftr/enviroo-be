package controllers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/services"

	"github.com/gin-gonic/gin"
)

const maxBodyUploadAPK = services.MaxUkuranAPK + 10*1024*1024

type AppReleaseController struct {
	Svc            services.AppReleaseService
	UploadDeadline time.Duration
}

func NewAppReleaseController(svc services.AppReleaseService, uploadDeadline time.Duration) *AppReleaseController {
	return &AppReleaseController{Svc: svc, UploadDeadline: uploadDeadline}
}

func (ac *AppReleaseController) CekVersi(c *gin.Context) {
	row, err := ac.Svc.CekVersi(c.Query("current"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Versi aplikasi berhasil diambil",
		"data":    row,
	})
}

func (ac *AppReleaseController) Latest(c *gin.Context) {
	url, err := ac.Svc.URLTerbaru()
	if err := handleServiceError(c, err); err != nil {
		return
	}
	c.Redirect(http.StatusFound, url)
}

func (ac *AppReleaseController) ListRilis(c *gin.Context) {
	rows, err := ac.Svc.ListRilis()
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Riwayat rilis berhasil diambil",
		"data":    rows,
	})
}

func (ac *AppReleaseController) TambahRilis(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	if ac.UploadDeadline > 0 {
		rc := http.NewResponseController(c.Writer)
		batasUpload := time.Now().Add(ac.UploadDeadline)
		if err := rc.SetReadDeadline(batasUpload); err != nil {
			log.Printf("[AppRelease] gagal melonggarkan read deadline upload: %v", err)
		}
		if err := rc.SetWriteDeadline(batasUpload); err != nil {
			log.Printf("[AppRelease] gagal melonggarkan write deadline upload: %v", err)
		}
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyUploadAPK)

	fileHeader, err := c.FormFile("apk")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Ukuran upload terlalu besar"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "File APK wajib diupload (field: apk)"})
		return
	}

	versionCode, err := strconv.Atoi(strings.TrimSpace(c.PostForm("version_code")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version_code harus berupa angka"})
		return
	}

	wajib := false
	if raw := strings.TrimSpace(c.PostForm("wajib")); raw != "" {
		wajib, err = strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "wajib harus true atau false"})
			return
		}
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file APK"})
		return
	}
	defer file.Close()

	row, err := ac.Svc.TambahRilis(services.TambahRilisInput{
		File:        file,
		FileName:    fileHeader.Filename,
		FileSize:    fileHeader.Size,
		VersionCode: versionCode,
		VersionName: c.PostForm("version_name"),
		Changelog:   c.PostForm("changelog"),
		Wajib:       wajib,
		CreatedBy:   claims.UserID,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Rilis aplikasi berhasil ditambahkan",
		"data":    row,
	})
}

func (ac *AppReleaseController) UbahRilis(c *gin.Context) {
	var req struct {
		Changelog *string `json:"changelog"`
		Wajib     *bool   `json:"wajib"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	row, err := ac.Svc.UbahRilis(c.Param("release_id"), services.UbahRilisInput{
		Changelog: req.Changelog,
		Wajib:     req.Wajib,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Rilis aplikasi berhasil diubah",
		"data":    row,
	})
}

func (ac *AppReleaseController) HapusRilis(c *gin.Context) {
	if err := handleServiceError(c, ac.Svc.HapusRilis(c.Param("release_id"))); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rilis aplikasi berhasil dihapus"})
}
