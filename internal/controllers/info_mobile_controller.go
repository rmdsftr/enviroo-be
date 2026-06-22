package controllers

import (
	"enviroo-be/internal/models"
	"net/http"
	"sort"
	"time"

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
	if err := imc.db.Preload("Bank").Where("nasabah_id = ? AND status_nasabah = ?", nasabahID, models.Aktif).First(&nasabah).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		return
	}

	bankID := nasabah.BankID
	namaBank := nasabah.Bank.NamaBank

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	until := today.AddDate(0, 1, 0)

	var semuaJadwal []models.Jadwal
	if err := imc.db.Where(
		"bank_id = ? AND jenis_jadwal = ? AND is_active = ?",
		bankID, models.JadwalPenimbangan, true,
	).Find(&semuaJadwal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil jadwal penimbangan"})
		return
	}

	type JadwalItem struct {
		Tanggal    string `json:"tanggal"`
		JamMulai   string `json:"jam_mulai"`
		JamSelesai string `json:"jam_selesai"`
		NamaJadwal string `json:"nama_jadwal"`
	}

	hariOf := func(t time.Time) models.HariEnum {
		return []models.HariEnum{
			models.Minggu, models.Senin, models.Selasa, models.Rabu,
			models.Kamis, models.Jumat, models.Sabtu,
		}[t.Weekday()]
	}

	weekOfMonth := func(d time.Time) int {
		first := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
		return (d.Day()+int(first.Weekday())-1)/7 + 1
	}

	var result []JadwalItem

	for _, j := range semuaJadwal {
		isRutin := j.IsRutin != nil && *j.IsRutin

		if isRutin {
			for d := today; !d.After(until); d = d.AddDate(0, 0, 1) {
				if hariOf(d) == j.Hari && weekOfMonth(d) == j.MingguKe {
					result = append(result, JadwalItem{
						Tanggal:    d.Format("2006-01-02"),
						JamMulai:   j.JamMulai,
						JamSelesai: j.JamSelesai,
						NamaJadwal: "Penimbangan Rutin",
					})
				}
			}
		} else {
			tgl := time.Date(j.Tanggal.Year(), j.Tanggal.Month(), j.Tanggal.Day(), 0, 0, 0, 0, now.Location())
			if !tgl.Before(today) && !tgl.After(until) {
				nama := j.NamaJadwalSpesial
				if nama == "" {
					nama = "Penimbangan Rutin"
				}
				result = append(result, JadwalItem{
					Tanggal:    tgl.Format("2006-01-02"),
					JamMulai:   j.JamMulai,
					JamSelesai: j.JamSelesai,
					NamaJadwal: nama,
				})
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Tanggal < result[j].Tanggal
	})

	if result == nil {
		result = []JadwalItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil jadwal penimbangan",
		"data": gin.H{
			"nama_bank": namaBank,
			"jadwal":    result,
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
