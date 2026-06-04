package controllers

import (
	"enviroo-be/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type InfoMobileController struct {
	db *gorm.DB
}

func NewInfoMobileController(db *gorm.DB) *InfoMobileController {
	return &InfoMobileController{db: db}
}

func (imc *InfoMobileController) JadwalPenimbanganForNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	var nasabah models.Nasabah
	// Preload Bank untuk mendapatkan nama bank
	if err := imc.db.Preload("Bank").Where("nasabah_id = ? AND status_nasabah=?", nasabahID, models.Aktif).First(&nasabah).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		return
	}

	bankID := nasabah.BankID

	var bank models.BankSampah
	if err := imc.db.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil bank sampah"})
		return 
	}

	// Gunakan Find untuk mengambil semua jadwal rutin penimbangan (bisa jadi lebih dari satu hari)
	var jadwalList []models.Jadwal
	if err := imc.db.Where("bank_id = ? AND jenis_jadwal = ? AND is_active = ? AND is_rutin=?", bankID, models.JadwalPenimbangan, true, true).Find(&jadwalList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil jadwal penimbangan"})
		return
	}

	var responseJadwal []gin.H
	for _, j := range jadwalList {
		responseJadwal = append(responseJadwal, gin.H{
			"hari":        j.Hari,
			"minggu_ke":   j.MingguKe,
			"jam_mulai":   j.JamMulai,
			"jam_selesai": j.JamSelesai,
		})
	}

	if responseJadwal == nil {
		responseJadwal = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil jadwal penimbangan",
		"data": gin.H{
			"nama_bank": bank.NamaBank,
			"jadwal":    responseJadwal,
		},
	})
}


func (imc *InfoMobileController) RewardOverviewNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")
	
	var nasabah models.Nasabah
	if err := imc.db.Where("nasabah_id = ? AND status_nasabah=?", nasabahID, models.Aktif).First(&nasabah).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		return
	}

	bankID := nasabah.BankID
	var bank models.BankSampah
	if err := imc.db.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil bank sampah"})
		return 
	}

	if bank.JenisBank == models.BSU {
		if bank.ParentBankID != nil {
			bankID = *bank.ParentBankID
		}		
	} 

	var rewardsBank []models.NilaiRewardBank
	if err := imc.db.Preload("Reward").Where("bank_id = ? AND is_active = ?", bankID, true).Find(&rewardsBank).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil nilai reward bank"})
		return 
	}

	type UangCard struct {
		RewardID        int     `json:"reward_id"`
		NamaReward      string  `json:"nama_reward"`
		DeskripsiReward string  `json:"deskripsi_reward"`
		PersentaseBagiHasil float64 `json:"persentase_bagi_hasil"`
	}

	type SembakoCard struct {
		RewardID        int     `json:"reward_id"`
		NamaReward      string  `json:"nama_reward"`
		DeskripsiReward string  `json:"deskripsi_reward"`
		PersentaseBagiHasil float64 `json:"persentase_bagi_hasil"`
	}

	var RewardCards struct {
		Uang    []UangCard    `json:"uang"`
		Sembako []SembakoCard `json:"sembako"`
	}

	// Inisialisasi agar tidak null di response
	RewardCards.Uang = []UangCard{}
	RewardCards.Sembako = []SembakoCard{}

	// Hanya ambil data untuk level_user = nasabah
	for _, rb := range rewardsBank {
		if rb.LevelUser != models.LevelNasabah {
			continue
		}
		if rb.RewardID == 0 {
			continue
		}

		deskripsi := ""
		if rb.Reward.Deskripsi != nil {
			deskripsi = *rb.Reward.Deskripsi
		}

		switch rb.Reward.NamaReward {
		case models.RewardEnumUang:
			RewardCards.Uang = append(RewardCards.Uang, UangCard{
				RewardID:            rb.RewardID,
				NamaReward:          string(rb.Reward.NamaReward),
				DeskripsiReward:     deskripsi,
				PersentaseBagiHasil: rb.PersenBagiHasil,
			})

		case models.RewardEnumSembako:
			RewardCards.Sembako = append(RewardCards.Sembako, SembakoCard{
				RewardID:            rb.RewardID,
				NamaReward:          string(rb.Reward.NamaReward),
				DeskripsiReward:     deskripsi,
				PersentaseBagiHasil: rb.PersenBagiHasil,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil informasi reward nasabah",
		"data":    RewardCards,
	})
}
