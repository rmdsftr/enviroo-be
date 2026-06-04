package controllers

import (
	"enviroo-be/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type JadwalController struct {
	db *gorm.DB
}

func NewJadwalController(db *gorm.DB) *JadwalController {
	return &JadwalController{db: db}
}

func getWeekOfMonth(date time.Time) int {
	firstOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	firstWeekday := int(firstOfMonth.Weekday())
	adjustedDay := date.Day() + firstWeekday - 1
	return (adjustedDay / 7) + 1
}

func (jc *JadwalController) AddNewJadwal(c *gin.Context) {
	bankID := c.Param("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank ID is required"})
		return
	}

	var bank models.BankSampah
	if err := jc.db.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank not found"})
		return
	}

	type AddNewJadwalRequest struct {
		Hari              models.HariEnum   `json:"hari"`
		MingguKe          int               `json:"minggu_ke"`
		JamMulai          string            `json:"jam_mulai"`
		JamSelesai        string            `json:"jam_selesai"`
		JenisJadwal       models.JadwalEnum `json:"jenis_jadwal"`
		TargetBankID      string            `json:"target_bank_id"`
		IsActive          bool              `json:"is_active"`
		IsRutin           bool              `json:"is_rutin"`
		Tanggal           string            `json:"tanggal"`
		NamaJadwalSpesial string            `json:"nama_jadwal_spesial"`
		AdminID      string            `json:"admin_id"`
	}

	var req AddNewJadwalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tanggal time.Time
	if !req.IsRutin && req.Tanggal != "" {
		t, err := time.Parse("2006-01-02", req.Tanggal)
		if err != nil {
			t, err = time.Parse(time.RFC3339, req.Tanggal)
		}
		if err == nil {
			tanggal = t
			if req.Hari == "" {
				weekdays := []models.HariEnum{models.Minggu, models.Senin, models.Selasa, models.Rabu, models.Kamis, models.Jumat, models.Sabtu}
				req.Hari = weekdays[t.Weekday()]
			}
			if req.MingguKe == 0 {
				req.MingguKe = getWeekOfMonth(t)
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Tanggal format. Use YYYY-MM-DD"})
			return
		}
	}

	if req.IsRutin {
		tanggal = time.Time{}
		req.NamaJadwalSpesial = ""
	}

	if req.Hari == "" {
		req.Hari = models.Senin // Fallback to avoid Postgres enum error empty string
	}

	if bank.JenisBank == models.BSU {
		req.TargetBankID = ""
	}

	newJadwal := models.Jadwal{
		BankID:            bankID,
		Hari:              req.Hari,
		MingguKe:          req.MingguKe,
		JamMulai:          req.JamMulai,
		JamSelesai:        req.JamSelesai,
		JenisJadwal:       req.JenisJadwal,
		TargetBankID:      req.TargetBankID,
		IsActive:          &req.IsActive,
		IsRutin:           &req.IsRutin,
		Tanggal:           tanggal,
		NamaJadwalSpesial: req.NamaJadwalSpesial,
		CreatedBy:         req.AdminID,
	}

	if err := jc.db.Create(&newJadwal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil ditambahkan", "data": newJadwal})
}

func (jc *JadwalController) GetJadwalBank(c *gin.Context) {
	bankID := c.Param("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank ID is required"})
		return
	}

	var bank models.BankSampah
	if err := jc.db.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank not found"})
		return
	}

	type PenimbanganResponse struct {
		models.Jadwal
		BankName string `json:"bank_name"`
	}

	type PengangkutanResponse struct {
		models.Jadwal
		BankName       string `json:"bank_name"`
		TargetBankName string `json:"target_bank_name"`
	}

	var listPenimbangan []PenimbanganResponse
	var listPengangkutan []PengangkutanResponse

	// Get Penimbangan where bank_id = bankID and jenis_jadwal = "penimbangan"
	if err := jc.db.Table("jadwal").
		Select("jadwal.*, bank_sampah.nama_bank as bank_name").
		Joins("left join bank_sampah on bank_sampah.bank_id = jadwal.bank_id").
		Where("jadwal.bank_id = ? AND jadwal.jenis_jadwal = ?", bankID, models.JadwalPenimbangan).
		Find(&listPenimbangan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get Pengangkutan based on JenisBank
	if bank.JenisBank == models.BSU {
		// BSU fetches pengangkutan where they are the target
		if err := jc.db.Table("jadwal").
			Select("jadwal.*, b1.nama_bank as bank_name, b2.nama_bank as target_bank_name").
			Joins("left join bank_sampah b1 on b1.bank_id = jadwal.bank_id").
			Joins("left join bank_sampah b2 on b2.bank_id = jadwal.target_bank_id").
			Where("jadwal.target_bank_id = ? AND jadwal.jenis_jadwal = ?", bankID, models.JadwalPengangkutan).
			Find(&listPengangkutan).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// Other banks fetch pengangkutan where they are the initiator (bank_id)
		if err := jc.db.Table("jadwal").
			Select("jadwal.*, b1.nama_bank as bank_name, b2.nama_bank as target_bank_name").
			Joins("left join bank_sampah b1 on b1.bank_id = jadwal.bank_id").
			Joins("left join bank_sampah b2 on b2.bank_id = jadwal.target_bank_id").
			Where("jadwal.bank_id = ? AND jadwal.jenis_jadwal = ?", bankID, models.JadwalPengangkutan).
			Find(&listPengangkutan).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"penimbangan": listPenimbangan,
			"pengangkutan": listPengangkutan,
		},
	})
}

func (jc *JadwalController) GetAllJadwal(c *gin.Context) {
	jenisParam := c.Query("type")
	if jenisParam != string(models.JadwalPenimbangan) && jenisParam != string(models.JadwalPengangkutan) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query param 'type' harus 'penimbangan' atau 'pengangkutan'"})
		return
	}
	jenisJadwal := models.JadwalEnum(jenisParam)

	type JadwalItem struct {
		JadwalID          string          `json:"jadwal_id"`
		Hari              models.HariEnum `json:"hari"`
		MingguKe          int             `json:"minggu_ke"`
		JamMulai          string          `json:"jam_mulai"`
		JamSelesai        string          `json:"jam_selesai"`
		IsActive          *bool           `json:"is_active"`
		Tanggal           time.Time       `json:"tanggal,omitempty"`
		NamaJadwalSpesial string          `json:"nama_jadwal_spesial,omitempty"`
		CreatedAt         time.Time       `json:"created_at"`
	}

	type rawRow struct {
		JadwalID          string           `gorm:"column:jadwal_id"`
		BankID            string           `gorm:"column:bank_id"`
		NamaBank          string           `gorm:"column:nama_bank"`
		JenisBank         models.JenisBank `gorm:"column:jenis_bank"`
		Hari              models.HariEnum  `gorm:"column:hari"`
		MingguKe          int              `gorm:"column:minggu_ke"`
		JamMulai          string           `gorm:"column:jam_mulai"`
		JamSelesai        string           `gorm:"column:jam_selesai"`
		IsActive          *bool            `gorm:"column:is_active"`
		IsRutin           *bool            `gorm:"column:is_rutin"`
		Tanggal           time.Time        `gorm:"column:tanggal"`
		NamaJadwalSpesial string           `gorm:"column:nama_jadwal_spesial"`
		TargetBankID      string           `gorm:"column:target_bank_id"`
		TargetBankName    string           `gorm:"column:target_bank_name"`
		CreatedAt         time.Time        `gorm:"column:created_at"`
	}

	var rows []rawRow
	if err := jc.db.Table("jadwal").
		Select(`jadwal.jadwal_id, jadwal.bank_id, b1.nama_bank, b1.jenis_bank,
			jadwal.hari, jadwal.minggu_ke, jadwal.jam_mulai, jadwal.jam_selesai,
			jadwal.is_active, jadwal.is_rutin, jadwal.tanggal, jadwal.nama_jadwal_spesial,
			jadwal.target_bank_id, b2.nama_bank as target_bank_name, jadwal.created_at`).
		Joins("JOIN bank_sampah b1 ON b1.bank_id = jadwal.bank_id").
		Joins("LEFT JOIN bank_sampah b2 ON b2.bank_id = jadwal.target_bank_id").
		Where("jadwal.jenis_jadwal = ?", jenisJadwal).
		Order("b1.nama_bank, b2.nama_bank, jadwal.created_at").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data jadwal: " + err.Error()})
		return
	}

	toItem := func(row rawRow) JadwalItem {
		return JadwalItem{
			JadwalID:          row.JadwalID,
			Hari:              row.Hari,
			MingguKe:          row.MingguKe,
			JamMulai:          row.JamMulai,
			JamSelesai:        row.JamSelesai,
			IsActive:          row.IsActive,
			Tanggal:           row.Tanggal,
			NamaJadwalSpesial: row.NamaJadwalSpesial,
			CreatedAt:         row.CreatedAt,
		}
	}

	if jenisJadwal == models.JadwalPenimbangan {
		// Penimbangan: dikelompokkan per jenis bank (BSI/BSM/BSU) → per bank
		type BankGroup struct {
			BankID     string       `json:"bank_id"`
			NamaBank   string       `json:"nama_bank"`
			Rutin      []JadwalItem `json:"rutin"`
			TidakRutin []JadwalItem `json:"tidak_rutin"`
		}

		grouped := map[models.JenisBank]map[string]*BankGroup{
			models.BSI: {},
			models.BSM: {},
			models.BSU: {},
		}

		for _, row := range rows {
			bankMap, ok := grouped[row.JenisBank]
			if !ok {
				continue
			}
			if _, exists := bankMap[row.BankID]; !exists {
				bankMap[row.BankID] = &BankGroup{
					BankID:     row.BankID,
					NamaBank:   row.NamaBank,
					Rutin:      make([]JadwalItem, 0),
					TidakRutin: make([]JadwalItem, 0),
				}
			}
			g := bankMap[row.BankID]
			if row.IsRutin != nil && *row.IsRutin {
				g.Rutin = append(g.Rutin, toItem(row))
			} else {
				g.TidakRutin = append(g.TidakRutin, toItem(row))
			}
		}

		toSlice := func(m map[string]*BankGroup) []BankGroup {
			result := make([]BankGroup, 0, len(m))
			for _, v := range m {
				result = append(result, *v)
			}
			return result
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"bsi": toSlice(grouped[models.BSI]),
				"bsm": toSlice(grouped[models.BSM]),
				"bsu": toSlice(grouped[models.BSU]),
			},
		})
		return
	}

	// Pengangkutan: dikelompokkan per BSI → per BSU tujuan
	// bank_id = BSI (yang mengangkut), target_bank_id = BSU (tujuan)
	type RuteBSU struct {
		BSUID      string       `json:"bsu_id"`
		NamaBSU    string       `json:"nama_bsu"`
		Rutin      []JadwalItem `json:"rutin"`
		TidakRutin []JadwalItem `json:"tidak_rutin"`
	}

	type RuteBSI struct {
		BSIID      string    `json:"bsi_id"`
		NamaBSI    string    `json:"nama_bsi"`
		RuteBSU    []RuteBSU `json:"rute_bsu"`
	}

	// map[bsi_id]map[bsu_id]
	bsiMap := map[string]map[string]*RuteBSU{}
	bsiOrder := []string{}

	for _, row := range rows {
		if _, exists := bsiMap[row.BankID]; !exists {
			bsiMap[row.BankID] = map[string]*RuteBSU{}
			bsiOrder = append(bsiOrder, row.BankID)
		}
		bsuMap := bsiMap[row.BankID]
		if _, exists := bsuMap[row.TargetBankID]; !exists {
			bsuMap[row.TargetBankID] = &RuteBSU{
				BSUID:      row.TargetBankID,
				NamaBSU:    row.TargetBankName,
				Rutin:      make([]JadwalItem, 0),
				TidakRutin: make([]JadwalItem, 0),
			}
		}
		rute := bsuMap[row.TargetBankID]
		if row.IsRutin != nil && *row.IsRutin {
			rute.Rutin = append(rute.Rutin, toItem(row))
		} else {
			rute.TidakRutin = append(rute.TidakRutin, toItem(row))
		}
	}

	// Simpan nama BSI untuk setiap bsi_id
	bsiNama := map[string]string{}
	for _, row := range rows {
		if _, exists := bsiNama[row.BankID]; !exists {
			bsiNama[row.BankID] = row.NamaBank
		}
	}

	result := make([]RuteBSI, 0, len(bsiOrder))
	for _, bsiID := range bsiOrder {
		bsuMap := bsiMap[bsiID]
		rutes := make([]RuteBSU, 0, len(bsuMap))
		for _, v := range bsuMap {
			rutes = append(rutes, *v)
		}
		result = append(result, RuteBSI{
			BSIID:   bsiID,
			NamaBSI: bsiNama[bsiID],
			RuteBSU: rutes,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (jc *JadwalController) DeleteJadwal(c *gin.Context) {
	jadwalID := c.Param("jadwal_id")
	if jadwalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jadwal ID is required"})
		return
	}

	var jadwal models.Jadwal
	if err := jc.db.Where("jadwal_id = ?", jadwalID).First(&jadwal).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jadwal not found"})
		return
	}

	if err := jc.db.Delete(&jadwal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil dihapus"})
}

func (jc *JadwalController) UpdateJadwal(c *gin.Context) {
	jadwalID := c.Param("jadwal_id")
	if jadwalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jadwal ID is required"})
		return
	}

	var jadwal models.Jadwal
	if err := jc.db.Where("jadwal_id = ?", jadwalID).First(&jadwal).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jadwal not found"})
		return
	}

	type UpdateJadwalRequest struct {
		Hari         models.HariEnum   `json:"hari"`
		MingguKe     int               `json:"minggu_ke"`
		JamMulai     string            `json:"jam_mulai"`
		JamSelesai   string            `json:"jam_selesai"`
		TargetBankID string            `json:"target_bank_id"`
		IsActive     *bool             `json:"is_active"`
		Tanggal      string            `json:"tanggal"`
		NamaJadwalSpesial string         `json:"nama_jadwal_spesial"`
		AdminID      string            `json:"admin_id"`
	}

	var req UpdateJadwalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tanggal time.Time
	var err error
	
	// Handle Tanggal parsing only if it is provided and the schedule is not rutin
	// If it's rutin, we don't allow modifying Tanggal
	if req.Tanggal != "" && jadwal.IsRutin != nil && !*jadwal.IsRutin {
		tanggal, err = time.Parse("2006-01-02", req.Tanggal)
		if err != nil {
			tanggal, err = time.Parse(time.RFC3339, req.Tanggal)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Tanggal format. Use YYYY-MM-DD"})
				return
			}
		}
	}

	if req.TargetBankID != "" {
		jadwal.TargetBankID = req.TargetBankID
	}

	if req.Hari != "" {
		jadwal.Hari = req.Hari
	}

	if req.MingguKe != 0 {
		jadwal.MingguKe = req.MingguKe
	}

	if req.JamMulai != "" {
		jadwal.JamMulai = req.JamMulai
	}

	if req.JamSelesai != "" {
		jadwal.JamSelesai = req.JamSelesai
	}

	if req.IsActive != nil {
		jadwal.IsActive = req.IsActive
	}

	// Update tanggal only if it's explicitly provided and the schedule is not rutin
	if req.Tanggal != "" && jadwal.IsRutin != nil && !*jadwal.IsRutin {
		jadwal.Tanggal = tanggal
	}
	
	if req.NamaJadwalSpesial != "" && jadwal.IsRutin != nil && !*jadwal.IsRutin {
		jadwal.NamaJadwalSpesial = req.NamaJadwalSpesial
	}

	if req.AdminID != "" {
		jadwal.CreatedBy = req.AdminID
	}

	if err := jc.db.Save(&jadwal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil diupdate", "data": jadwal})
}
