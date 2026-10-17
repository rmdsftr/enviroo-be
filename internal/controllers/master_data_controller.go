package controllers

import (
	"enviroo-be/internal/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MasterDataController struct {
	DB *gorm.DB
}

func NewMasterDataController(db *gorm.DB) *MasterDataController {
	return &MasterDataController{DB: db}
}

func (mc *MasterDataController) GetAllMasterSampah(c *gin.Context) {
	q := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	db := mc.DB.Model(&models.Sampah{})
	if q != "" {
		db = db.Where("nama_sampah ILIKE ?", "%"+q+"%")
	}

	var total int64
	db.Count(&total)

	var results []models.Sampah
	if err := db.Preload("Satuan").Offset(offset).Limit(limit).Order("nama_sampah ASC").Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data sampah"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (mc *MasterDataController) CreateMasterSampah(c *gin.Context) {
	var req struct {
		NamaSampah string `json:"nama_sampah" binding:"required"`
		SatuanID   int    `json:"satuan_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var satuan models.Satuan
	if err := mc.DB.First(&satuan, req.SatuanID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "satuan_id tidak valid"})
		return
	}

	sampah := models.Sampah{
		NamaSampah: req.NamaSampah,
		SatuanID:   req.SatuanID,
	}
	if err := mc.DB.Create(&sampah).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat data sampah"})
		return
	}
	sampah.Satuan = satuan

	c.JSON(http.StatusCreated, gin.H{
		"message": "Data sampah berhasil dibuat",
		"data":    sampah,
	})
}

func (mc *MasterDataController) UpdateMasterSampah(c *gin.Context) {
	sarokID, err := strconv.Atoi(c.Param("sarok_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sarok_id tidak valid"})
		return
	}

	var sampah models.Sampah
	if err := mc.DB.First(&sampah, sarokID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data sampah tidak ditemukan"})
		return
	}

	var req struct {
		NamaSampah string `json:"nama_sampah"`
		SatuanID   int    `json:"satuan_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.NamaSampah != "" {
		sampah.NamaSampah = req.NamaSampah
	}
	if req.SatuanID != 0 {
		if err := mc.DB.First(&models.Satuan{}, req.SatuanID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "satuan_id tidak valid"})
			return
		}
		sampah.SatuanID = req.SatuanID
	}

	if err := mc.DB.Save(&sampah).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate data sampah"})
		return
	}
	mc.DB.First(&sampah.Satuan, sampah.SatuanID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Data sampah berhasil diupdate",
		"data":    sampah,
	})
}

func (mc *MasterDataController) DeleteMasterSampah(c *gin.Context) {
	sarokID, err := strconv.Atoi(c.Param("sarok_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sarok_id tidak valid"})
		return
	}

	var sampah models.Sampah
	if err := mc.DB.First(&sampah, sarokID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data sampah tidak ditemukan"})
		return
	}

	var usedCount int64
	mc.DB.Model(&models.KatalogSampah{}).Where("sarok_id = ?", sarokID).Count(&usedCount)
	if usedCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Data sampah tidak dapat dihapus karena sudah digunakan dalam katalog"})
		return
	}

	if err := mc.DB.Delete(&sampah).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus data sampah"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data sampah berhasil dihapus"})
}

func (mc *MasterDataController) GetStatistikSampah(c *gin.Context) {
	type StatistikSampahItem struct {
		SarokID       int     `gorm:"column:sarok_id"        json:"sarok_id"`
		NamaSampah    string  `gorm:"column:nama_sampah"     json:"nama_sampah"`
		Satuan        string  `gorm:"column:satuan"          json:"satuan"`
		JenisReward   string  `gorm:"column:jenis_reward"    json:"jenis_reward"`
		SatuanReward  string  `gorm:"column:satuan_reward"   json:"satuan_reward"`
		RataRataHarga float64 `gorm:"column:rata_rata_harga" json:"rata_rata_harga"`
		JumlahKatalog int64   `gorm:"column:jumlah_katalog"  json:"jumlah_katalog"`
	}

	var startDate, endDate time.Time
	var filterTanggal bool

	bmStr := c.Query("bulan_mulai")
	tmStr := c.Query("tahun_mulai")
	bsStr := c.Query("bulan_selesai")
	tsStr := c.Query("tahun_selesai")

	if bmStr != "" || tmStr != "" || bsStr != "" || tsStr != "" {
		bm, e1 := strconv.Atoi(bmStr)
		tm, e2 := strconv.Atoi(tmStr)
		bs, e3 := strconv.Atoi(bsStr)
		ts, e4 := strconv.Atoi(tsStr)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil ||
			bm < 1 || bm > 12 || bs < 1 || bs > 12 || tm < 2000 || ts < 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter bulan_mulai, tahun_mulai, bulan_selesai, tahun_selesai tidak valid"})
			return
		}
		startDate = time.Date(tm, time.Month(bm), 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(ts, time.Month(bs+1), 1, 0, 0, 0, 0, time.UTC)
		if endDate.Before(startDate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Rentang tanggal tidak valid: bulan_selesai lebih awal dari bulan_mulai"})
			return
		}
		filterTanggal = true
	}

	dateFilter := ""
	args := []interface{}{}
	if filterTanggal {
		dateFilter = "AND pj.created_at >= ? AND pj.created_at < ?"
		args = append(args, startDate, endDate)
	}

	query := `
		SELECT
			s.sarok_id,
			s.nama_sampah,
			sat.satuan,
			r.nama_reward                               AS jenis_reward,
			r.satuan                                    AS satuan_reward,
			COALESCE(AVG(dp.harga_nasabah_snapshot), 0) AS rata_rata_harga,
			COUNT(DISTINCT ks.sampah_id)                AS jumlah_katalog
		FROM sampah s
		JOIN satuan sat               ON sat.satuan_id  = s.satuan_id
		LEFT JOIN katalog_sampah ks   ON ks.sarok_id    = s.sarok_id
		LEFT JOIN reward r            ON r.reward_id    = ks.reward_id
		LEFT JOIN detail_penjualan dp ON dp.sampah_id   = ks.sampah_id
		LEFT JOIN penjualan pj        ON pj.penjualan_id = dp.penjualan_id ` + dateFilter + `
		GROUP BY s.sarok_id, s.nama_sampah, sat.satuan, r.nama_reward, r.satuan
		ORDER BY s.nama_sampah ASC, r.nama_reward ASC
	`

	var results []StatistikSampahItem
	if err := mc.DB.Raw(query, args...).Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil statistik sampah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Statistik sampah berhasil diambil",
		"data":    results,
	})
}

func (mc *MasterDataController) GetFavoritSampah(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	var startDate, endDate time.Time
	var filterTanggal bool

	bmStr := c.Query("bulan_mulai")
	tmStr := c.Query("tahun_mulai")
	bsStr := c.Query("bulan_selesai")
	tsStr := c.Query("tahun_selesai")

	if bmStr != "" || tmStr != "" || bsStr != "" || tsStr != "" {
		bm, e1 := strconv.Atoi(bmStr)
		tm, e2 := strconv.Atoi(tmStr)
		bs, e3 := strconv.Atoi(bsStr)
		ts, e4 := strconv.Atoi(tsStr)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil ||
			bm < 1 || bm > 12 || bs < 1 || bs > 12 || tm < 2000 || ts < 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter bulan_mulai, tahun_mulai, bulan_selesai, tahun_selesai tidak valid"})
			return
		}
		startDate = time.Date(tm, time.Month(bm), 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(ts, time.Month(bs+1), 1, 0, 0, 0, 0, time.UTC)
		if endDate.Before(startDate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Rentang tanggal tidak valid: bulan_selesai lebih awal dari bulan_mulai"})
			return
		}
		filterTanggal = true
	}

	type FavoritItem struct {
		SarokID       int     `gorm:"column:sarok_id"       json:"sarok_id"`
		NamaSampah    string  `gorm:"column:nama_sampah"    json:"nama_sampah"`
		Satuan        string  `gorm:"column:satuan"         json:"satuan"`
		TotalQty      float64 `gorm:"column:total_qty"      json:"total_qty"`
		JumlahNasabah int64   `gorm:"column:jumlah_nasabah" json:"jumlah_nasabah"`
		JumlahSetoran int64   `gorm:"column:jumlah_setoran" json:"jumlah_setoran"`
	}

	dateFilter := ""
	args := []interface{}{models.EntitasNasabah}
	if filterTanggal {
		dateFilter = "AND ts.created_at >= ? AND ts.created_at < ?"
		args = append(args, startDate, endDate)
	}
	args = append(args, limit)

	query := `
		SELECT
			s.sarok_id,
			s.nama_sampah,
			sat.satuan,
			SUM(ts.qty)                   AS total_qty,
			COUNT(DISTINCT ts.nasabah_id) AS jumlah_nasabah,
			COUNT(ts.tabungan_id)         AS jumlah_setoran
		FROM tabungan_sampah ts
		JOIN katalog_sampah ks ON ks.sampah_id = ts.sampah_id
		JOIN sampah s          ON s.sarok_id   = ks.sarok_id
		JOIN satuan sat        ON sat.satuan_id = s.satuan_id
		WHERE ts.entitas = ? ` + dateFilter + `
		GROUP BY s.sarok_id, s.nama_sampah, sat.satuan
		ORDER BY total_qty DESC
		LIMIT ?
	`

	var results []FavoritItem
	if err := mc.DB.Raw(query, args...).Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data favorit sampah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sampah favorit nasabah berhasil diambil",
		"data":    results,
	})
}

func (mc *MasterDataController) GetSampahPerKategori(c *gin.Context) {
	type SampahRow struct {
		KategoriID int    `gorm:"column:kategori_id"`
		Kategori   string `gorm:"column:kategori"`
		SarokID    int    `gorm:"column:sarok_id"`
		NamaSampah string `gorm:"column:nama_sampah"`
		Satuan     string `gorm:"column:satuan"`
	}

	var rows []SampahRow
	if err := mc.DB.Raw(`
		SELECT
			k.kategori_id,
			k.kategori,
			s.sarok_id,
			s.nama_sampah,
			sat.satuan
		FROM kategori_sampah k
		LEFT JOIN katalog_sampah ks ON ks.kategori_id = k.kategori_id
		LEFT JOIN sampah s          ON s.sarok_id     = ks.sarok_id
		LEFT JOIN satuan sat        ON sat.satuan_id  = s.satuan_id
		GROUP BY k.kategori_id, k.kategori, s.sarok_id, s.nama_sampah, sat.satuan
		ORDER BY k.kategori ASC, s.nama_sampah ASC
	`).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data: " + err.Error()})
		return
	}

	type SampahItem struct {
		SarokID    int    `json:"sarok_id"`
		NamaSampah string `json:"nama_sampah"`
		Satuan     string `json:"satuan"`
	}
	type KategoriItem struct {
		KategoriID int          `json:"kategori_id"`
		Kategori   string       `json:"kategori"`
		Sampah     []SampahItem `json:"sampah"`
	}

	ordered := []int{}
	grouped := map[int]*KategoriItem{}
	for _, r := range rows {
		if _, exists := grouped[r.KategoriID]; !exists {
			grouped[r.KategoriID] = &KategoriItem{
				KategoriID: r.KategoriID,
				Kategori:   r.Kategori,
				Sampah:     []SampahItem{},
			}
			ordered = append(ordered, r.KategoriID)
		}
		if r.SarokID != 0 {
			grouped[r.KategoriID].Sampah = append(grouped[r.KategoriID].Sampah, SampahItem{
				SarokID:    r.SarokID,
				NamaSampah: r.NamaSampah,
				Satuan:     r.Satuan,
			})
		}
	}

	result := make([]*KategoriItem, 0, len(ordered))
	for _, id := range ordered {
		result = append(result, grouped[id])
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data sampah per kategori berhasil diambil",
		"data":    result,
	})
}

func (mc *MasterDataController) GetAllMasterBarang(c *gin.Context) {
	q := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	db := mc.DB.Model(&models.Barang{})
	if q != "" {
		db = db.Where("nama_barang ILIKE ?", "%"+q+"%")
	}

	var total int64
	db.Count(&total)

	var results []models.Barang
	if err := db.Offset(offset).Limit(limit).Order("nama_barang ASC").Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data barang"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (mc *MasterDataController) CreateMasterBarang(c *gin.Context) {
	var req struct {
		NamaBarang string `json:"nama_barang" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	barang := models.Barang{
		NamaBarang: req.NamaBarang,
	}
	if err := mc.DB.Create(&barang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat data barang"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Data barang berhasil dibuat",
		"data":    barang,
	})
}

func (mc *MasterDataController) UpdateMasterBarang(c *gin.Context) {
	barangID, err := strconv.Atoi(c.Param("barang_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "barang_id tidak valid"})
		return
	}

	var barang models.Barang
	if err := mc.DB.First(&barang, barangID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data barang tidak ditemukan"})
		return
	}

	var req struct {
		NamaBarang string `json:"nama_barang"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.NamaBarang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama_barang tidak boleh kosong"})
		return
	}

	barang.NamaBarang = req.NamaBarang
	if err := mc.DB.Save(&barang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate data barang"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data barang berhasil diupdate",
		"data":    barang,
	})
}

func (mc *MasterDataController) DeleteMasterBarang(c *gin.Context) {
	barangID, err := strconv.Atoi(c.Param("barang_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "barang_id tidak valid"})
		return
	}

	var barang models.Barang
	if err := mc.DB.First(&barang, barangID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data barang tidak ditemukan"})
		return
	}

	var usedCount int64
	mc.DB.Model(&models.KatalogBarang{}).Where("barang_id = ?", barangID).Count(&usedCount)
	if usedCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Data barang tidak dapat dihapus karena sudah digunakan dalam katalog"})
		return
	}

	if err := mc.DB.Delete(&barang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus data barang"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data barang berhasil dihapus"})
}

func (mc *MasterDataController) GetFavoritBarang(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	var startDate, endDate time.Time
	var filterTanggal bool

	bmStr := c.Query("bulan_mulai")
	tmStr := c.Query("tahun_mulai")
	bsStr := c.Query("bulan_selesai")
	tsStr := c.Query("tahun_selesai")

	if bmStr != "" || tmStr != "" || bsStr != "" || tsStr != "" {
		bm, e1 := strconv.Atoi(bmStr)
		tm, e2 := strconv.Atoi(tmStr)
		bs, e3 := strconv.Atoi(bsStr)
		ts, e4 := strconv.Atoi(tsStr)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil ||
			bm < 1 || bm > 12 || bs < 1 || bs > 12 || tm < 2000 || ts < 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter bulan_mulai, tahun_mulai, bulan_selesai, tahun_selesai tidak valid"})
			return
		}
		startDate = time.Date(tm, time.Month(bm), 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(ts, time.Month(bs+1), 1, 0, 0, 0, 0, time.UTC)
		if endDate.Before(startDate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Rentang tanggal tidak valid: bulan_selesai lebih awal dari bulan_mulai"})
			return
		}
		filterTanggal = true
	}

	type FavoritBarangItem struct {
		BarangID      int     `gorm:"column:barang_id"      json:"barang_id"`
		NamaBarang    string  `gorm:"column:nama_barang"    json:"nama_barang"`
		TotalQty      float64 `gorm:"column:total_qty"      json:"total_qty"`
		TotalPoin     float64 `gorm:"column:total_poin"     json:"total_poin"`
		JumlahNasabah int64   `gorm:"column:jumlah_nasabah" json:"jumlah_nasabah"`
		JumlahTukar   int64   `gorm:"column:jumlah_tukar"   json:"jumlah_tukar"`
	}

	dateFilter := ""
	args := []interface{}{models.StatusPenarikanCompleted}
	if filterTanggal {
		dateFilter = "AND rpc.created_at >= ? AND rpc.created_at < ?"
		args = append(args, startDate, endDate)
	}
	args = append(args, limit)

	query := `
		SELECT
			s.barang_id,
			s.nama_barang,
			SUM(dps.qty)                 AS total_qty,
			SUM(dps.subtotal_poin)       AS total_poin,
			COUNT(DISTINCT p.nasabah_id) AS jumlah_nasabah,
			COUNT(dps.penarikan_id)      AS jumlah_tukar
		FROM detail_penarikan_barang dps
		JOIN penarikan p        ON p.penarikan_id  = dps.penarikan_id
		JOIN katalog_barang ks ON ks.produk_id   = dps.produk_id
		JOIN barang s          ON s.barang_id     = ks.barang_id
		JOIN riwayat_penarikan rpc ON rpc.penarikan_id = p.penarikan_id AND rpc.status_penarikan = 'pending'
		WHERE p.status_penarikan = ? ` + dateFilter + `
		GROUP BY s.barang_id, s.nama_barang
		ORDER BY total_qty DESC
		LIMIT ?
	`

	var results []FavoritBarangItem
	if err := mc.DB.Raw(query, args...).Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data favorit barang: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Barang favorit nasabah berhasil diambil",
		"data":    results,
	})
}

func (mc *MasterDataController) GetStatistikBarang(c *gin.Context) {
	type StatistikBarangItem struct {
		BarangID      int     `gorm:"column:barang_id"      json:"barang_id"`
		NamaBarang    string  `gorm:"column:nama_barang"    json:"nama_barang"`
		RataRataPoin  float64 `gorm:"column:rata_rata_poin" json:"rata_rata_poin"`
		JumlahKatalog int64   `gorm:"column:jumlah_katalog" json:"jumlah_katalog"`
	}

	var results []StatistikBarangItem
	if err := mc.DB.Raw(`
		SELECT
			s.barang_id,
			s.nama_barang,
			COALESCE(AVG(ks.nilai_poin), 0) AS rata_rata_poin,
			COUNT(ks.produk_id)            AS jumlah_katalog
		FROM barang s
		LEFT JOIN katalog_barang ks ON ks.barang_id = s.barang_id
		GROUP BY s.barang_id, s.nama_barang
		ORDER BY s.nama_barang ASC
	`).Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil statistik barang: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Statistik barang berhasil diambil",
		"data":    results,
	})
}
