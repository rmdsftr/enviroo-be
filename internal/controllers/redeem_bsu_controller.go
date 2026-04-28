package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RedeemBSUController struct {
	DB *gorm.DB
}

func NewRedeemBSUController(db *gorm.DB) *RedeemBSUController {
	return &RedeemBSUController{DB: db}
}

// RequestRedeemBSU: BSU mengajukan permintaan redeem poin ke BSI.
// BSU bisa memilih reward berupa sembako (detail item) atau reward lain (uang/emas).
// Transaksi akan berstatus "waiting" hingga diproses oleh admin BSI.
func (rpc *RedeemBSUController) RequestRedeemBSU(c *gin.Context) {
	bsuID := c.Param("bsu_id")
	adminBSUID := c.Param("admin_bsu_id")

	// 1. Validasi BSU
	var bank models.BankSampah
	if err := rpc.DB.Where("bank_id = ? AND jenis_bank = ? AND is_active = ?", bsuID, models.BSU, true).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Bank BSU tidak ditemukan atau tidak aktif"})
		return
	}

	// BSU harus memiliki parent BSI
	if bank.ParentBankID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "BSU tidak memiliki BSI induk"})
		return
	}
	bsiID := *bank.ParentBankID

	// 2. Bind input
	type redeemSembakoItem struct {
		SembakoID string  `json:"sembako_id" binding:"required"`
		Qty       float64 `json:"qty" binding:"required,gt=0"`
	}

	type inputRequest struct {
		RewardID          int                 `json:"reward_id" binding:"required"`
		PoinRedeem        float64             `json:"poin_redeem" binding:"required,gt=0"`
		RedeemSembakoItem []redeemSembakoItem `json:"redeem_sembako_item"`
	}

	var input inputRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Input tidak valid: " + err.Error()})
		return
	}

	// 3. Validasi reward: apakah BSI menyediakan reward ini?
	var reward models.NilaiRewardBank
	if err := rpc.DB.Preload("Reward").
		Where("reward_id = ? AND bank_id = ? AND is_active = ?", input.RewardID, bsiID, true).
		First(&reward).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "BSI tidak menyediakan reward tersebut atau reward tidak aktif"})
		return
	}

	// 4. Validasi saldo BSU
	var saldoBSU models.SaldoBank
	if err := rpc.DB.Where("bank_id = ?", bsuID).First(&saldoBSU).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Saldo BSU tidak ditemukan"})
		return
	}

	// 5. Cek apakah reward ini adalah sembako
	isSembako := strings.EqualFold(reward.Reward.NamaReward, "sembako")

	transaksiID := utils.GenerateID("RDM")

	if isSembako {
		// ── ALUR SEMBAKO ────────────────────────────────────────────────
		if len(input.RedeemSembakoItem) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Item sembako tidak boleh kosong untuk reward sembako"})
			return
		}

		totalPoinSembako := 0.0
		type sembakoKalkulasi struct {
			SembakoID    string
			Qty          float64
			NilaiPoin    float64
			SubtotalPoin float64
		}
		var kalkulasiList []sembakoKalkulasi

		for _, item := range input.RedeemSembakoItem {
			var schema models.SchemaHargaSembako
			if err := rpc.DB.
				Where("sembako_id = ? AND level_user = ?", item.SembakoID, models.LevelBSU).
				First(&schema).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"message": "Harga sembako untuk level BSU tidak ditemukan: " + item.SembakoID})
				return
			}

			var stok models.StokSembakoBank
			if err := rpc.DB.
				Where("bank_id = ? AND sembako_id = ?", bsiID, item.SembakoID).
				First(&stok).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"message": "Sembako tidak ditemukan di stok BSI: " + item.SembakoID})
				return
			}
			if item.Qty > stok.Stok {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Stok sembako di BSI tidak mencukupi untuk item: " + item.SembakoID})
				return
			}

			subtotal := item.Qty * schema.PoinHarga
			totalPoinSembako += subtotal
			kalkulasiList = append(kalkulasiList, sembakoKalkulasi{
				SembakoID:    item.SembakoID,
				Qty:          item.Qty,
				NilaiPoin:    schema.PoinHarga,
				SubtotalPoin: subtotal,
			})
		}

		if totalPoinSembako > saldoBSU.TotalPoin {
			c.JSON(http.StatusBadRequest, gin.H{
				"message":               "Saldo BSU tidak mencukupi",
				"saldo_bsu":             saldoBSU.TotalPoin,
				"total_poin_dibutuhkan": totalPoinSembako,
			})
			return
		}

		err := rpc.DB.Transaction(func(tx *gorm.DB) error {
			newRequest := models.TransaksiReward{
				TransaksiID:     transaksiID,
				JenisTransaksi:  models.TransferBSIToBSU,
				BankAsalID:      &bsuID,
				BankTujuanID:    &bsiID,
				RewardID:        &input.RewardID,
				Poin:            totalPoinSembako,
				Nominal:         0,
				StatusTransaksi: models.StatusTransaksiWaiting,
				UpdatedBy:       &adminBSUID,
			}
			if err := tx.Create(&newRequest).Error; err != nil {
				return err
			}

			for _, k := range kalkulasiList {
				detail := models.TransaksiRewardDetail{
					TransaksiID:  transaksiID,
					SembakoID:    k.SembakoID,
					Qty:          k.Qty,
					NilaiPoin:    k.NilaiPoin,
					SubtotalPoin: k.SubtotalPoin,
				}
				if err := tx.Create(&detail).Error; err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat permintaan redeem sembako"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":             "Permintaan redeem sembako berhasil dibuat, menunggu persetujuan BSI",
			"transaksi_id":        transaksiID,
			"total_poin_sembako":  totalPoinSembako,
		})

	} else {
		// ── ALUR NON-SEMBAKO (Uang / Emas) ─────────────────────────────
		if input.PoinRedeem > saldoBSU.TotalPoin {
			c.JSON(http.StatusBadRequest, gin.H{
				"message":     "Saldo BSU tidak mencukupi",
				"saldo_bsu":   saldoBSU.TotalPoin,
				"poin_redeem": input.PoinRedeem,
			})
			return
		}

		var saldoBSI models.SaldoBank
		if err := rpc.DB.Where("bank_id = ?", bsiID).First(&saldoBSI).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "Saldo BSI tidak ditemukan"})
			return
		}
		if input.PoinRedeem > saldoBSI.TotalPoin {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Saldo BSI tidak mencukupi untuk melayani redeem ini"})
			return
		}

		var nominal float64
		if reward.NilaiPoin > 0 {
			nominal = (input.PoinRedeem / reward.NilaiPoin) * reward.NilaiKonversi
		}

		err := rpc.DB.Transaction(func(tx *gorm.DB) error {
			newRequest := models.TransaksiReward{
				TransaksiID:     transaksiID,
				JenisTransaksi:  models.TransferBSIToBSU,
				BankAsalID:      &bsuID,
				BankTujuanID:    &bsiID,
				RewardID:        &input.RewardID,
				Poin:            input.PoinRedeem,
				Nominal:         nominal,
				StatusTransaksi: models.StatusTransaksiWaiting,
				UpdatedBy:       &adminBSUID,
			}
			return tx.Create(&newRequest).Error
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat permintaan redeem"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":      "Permintaan redeem berhasil dibuat, menunggu persetujuan BSI",
			"transaksi_id": transaksiID,
			"poin":         input.PoinRedeem,
			"nominal":      nominal,
		})
	}
}

// ConfirmRedeemBSU: Admin BSI memproses pengajuan redeem dari BSU.
// Status yang bisa dipilih: approved, rejected, canceled, success.
// Operasi finansial hanya terjadi saat status = "success".
func (rbc *RedeemBSUController) ConfirmRedeemBSU(c *gin.Context) {
	transaksiID := c.Param("transaksi_id")
	adminBSIID := c.Param("admin_bsi_id")

	// 1. Ambil data transaksi
	var transaksiReward models.TransaksiReward
	if err := rbc.DB.Where("transaksi_id = ?", transaksiID).First(&transaksiReward).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Pengajuan redeem tidak ditemukan"})
		return
	}

	// Hanya transaksi berstatus "waiting" atau "approved" yang bisa diproses
	if transaksiReward.StatusTransaksi != models.StatusTransaksiWaiting &&
		transaksiReward.StatusTransaksi != models.StatusTransaksiApproved {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Transaksi tidak dapat diproses, status saat ini: " + string(transaksiReward.StatusTransaksi),
		})
		return
	}

	// 2. Bind input
	type inputRequest struct {
		StatusTransaksi string `json:"status_transaksi" binding:"required"`
		Catatan         string `json:"catatan"`
	}

	var input inputRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Input tidak valid: " + err.Error()})
		return
	}

	targetStatus := models.StatusTransaksiReward(input.StatusTransaksi)

	// 3. Tangani berdasarkan status yang diminta
	switch targetStatus {

	case models.StatusTransaksiApproved, models.StatusTransaksiRejected:
		// Hanya update status dan catatan — tidak ada operasi finansial
		transaksiReward.StatusTransaksi = targetStatus
		transaksiReward.Catatan = &input.Catatan
		transaksiReward.UpdatedBy = &adminBSIID
		if err := rbc.DB.Save(&transaksiReward).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengupdate status transaksi"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":      "Status transaksi berhasil diupdate",
			"transaksi_id": transaksiID,
			"status":       string(targetStatus),
		})

	case models.StatusTransaksiSuccess:
		// ── OPERASI FINANSIAL ──────────────────────────────────────────────
		bsuID := transaksiReward.BankAsalID   // *string
		bsiID := transaksiReward.BankTujuanID // *string

		if bsuID == nil || bsiID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Data bank asal atau tujuan tidak valid"})
			return
		}

		poinRedeem := transaksiReward.Poin
		nominalRedeem := transaksiReward.Nominal
		rewardID := transaksiReward.RewardID

		// Cek apakah reward ini sembako
		var reward models.Reward
		if err := rbc.DB.Where("reward_id = ?", rewardID).First(&reward).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "Reward tidak ditemukan"})
			return
		}
		isSembako := strings.EqualFold(reward.NamaReward, "sembako")

		// Ambil detail sembako jika diperlukan (sebelum transaksi DB)
		var detailSembako []models.TransaksiRewardDetail
		if isSembako {
			if err := rbc.DB.Where("transaksi_id = ?", transaksiID).Find(&detailSembako).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengambil detail sembako"})
				return
			}
			if len(detailSembako) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Detail sembako tidak ditemukan untuk transaksi ini"})
				return
			}
		}

		// Bungkus semua operasi finansial dalam satu DB transaction
		err := rbc.DB.Transaction(func(tx *gorm.DB) error {

			// ── BSU: Saldo BERKURANG (BSU menghabiskan poinnya) ─────────
			var saldoBSU models.SaldoBank
			if err := tx.Where("bank_id = ?", *bsuID).First(&saldoBSU).Error; err != nil {
				return err
			}
			saldoBSUSebelum := saldoBSU.TotalPoin
			saldoBSUSesudah := saldoBSUSebelum - poinRedeem
			saldoBSU.TotalPoin = saldoBSUSesudah
			if err := tx.Save(&saldoBSU).Error; err != nil {
				return err
			}
			logSaldoBSU := models.TransaksiSaldoBank{
				TransaksiBankID: utils.GenerateID("TSB"),
				SaldoBankID:     saldoBSU.SaldoBankID,
				JenisTransaksi:  models.TransaksiKonversi,
				Jumlah:          poinRedeem,
				SaldoSebelum:    saldoBSUSebelum,
				SaldoSesudah:    saldoBSUSesudah,
			}
			if err := tx.Create(&logSaldoBSU).Error; err != nil {
				return err
			}

			// ── BSI: Saldo BERKURANG (BSI membayar hutang ke BSU) ───────
			var saldoBSI models.SaldoBank
			if err := tx.Where("bank_id = ?", *bsiID).First(&saldoBSI).Error; err != nil {
				return err
			}
			saldoBSISebelum := saldoBSI.TotalPoin
			saldoBSISesudah := saldoBSISebelum - poinRedeem
			saldoBSI.TotalPoin = saldoBSISesudah
			if err := tx.Save(&saldoBSI).Error; err != nil {
				return err
			}
			logSaldoBSI := models.TransaksiSaldoBank{
				TransaksiBankID: utils.GenerateID("TSB"),
				SaldoBankID:     saldoBSI.SaldoBankID,
				JenisTransaksi:  models.TransaksiKonversi,
				Jumlah:          poinRedeem,
				SaldoSebelum:    saldoBSISebelum,
				SaldoSesudah:    saldoBSISesudah,
			}
			if err := tx.Create(&logSaldoBSI).Error; err != nil {
				return err
			}

			// ── BSI: Kas BERKURANG (hanya untuk reward non-sembako) ─────
			if !isSembako {
				var kasBSI models.KasBank
				if err := tx.Where("bank_id = ? AND reward_id = ?", *bsiID, rewardID).First(&kasBSI).Error; err != nil {
					return err
				}
				kasSebelum := kasBSI.Nominal
				kasSesudah := kasSebelum - nominalRedeem
				kasBSI.Nominal = kasSesudah
				if err := tx.Save(&kasBSI).Error; err != nil {
					return err
				}
				logKas := models.RiwayatArusKas{
					ArusKasID:      utils.GenerateID("ARK"),
					KasID:          kasBSI.KasID.String(),
					Jumlah:         nominalRedeem,
					NominalSebelum: kasSebelum,
					NominalSesudah: kasSesudah,
				}
				if err := tx.Create(&logKas).Error; err != nil {
					return err
				}
			}

			// ── Stok Sembako BSI BERKURANG per item ─────────────────────
			if isSembako {
				for _, item := range detailSembako {
					var stok models.StokSembakoBank
					if err := tx.Where("bank_id = ? AND sembako_id = ?", *bsiID, item.SembakoID).First(&stok).Error; err != nil {
						return err
					}
					stok.Stok -= item.Qty
					if err := tx.Save(&stok).Error; err != nil {
						return err
					}
				}
			}

			// ── Update status transaksi jadi success ─────────────────────
			transaksiReward.StatusTransaksi = models.StatusTransaksiSuccess
			transaksiReward.Catatan = &input.Catatan
			transaksiReward.UpdatedBy = &adminBSIID
			if err := tx.Save(&transaksiReward).Error; err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memproses redeem: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "Redeem BSU berhasil diproses",
			"transaksi_id": transaksiID,
			"poin":         poinRedeem,
			"nominal":      nominalRedeem,
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Status tidak valid untuk konfirmasi BSI. Pilihan: approved, rejected, success",
		})
	}
}

// CancelRequestRedeemBSU: BSU membatalkan pengajuan redeem yang sudah dibuat (hanya jika masih waiting).
func (rbc *RedeemBSUController) CancelRequestRedeemBSU(c *gin.Context) {
	transaksiID := c.Param("transaksi_id")
	adminBSUID := c.Param("admin_bsu_id")

	var transaksiReward models.TransaksiReward
	if err := rbc.DB.Where("transaksi_id = ?", transaksiID).First(&transaksiReward).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Pengajuan redeem tidak ditemukan"})
		return
	}

	// Hanya bisa membatalkan jika status masih waiting
	if transaksiReward.StatusTransaksi != models.StatusTransaksiWaiting {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Pengajuan tidak dapat dibatalkan karena status sudah: " + string(transaksiReward.StatusTransaksi),
		})
		return
	}

	transaksiReward.StatusTransaksi = models.StatusTransaksiCanceled
	transaksiReward.UpdatedBy = &adminBSUID

	if err := rbc.DB.Save(&transaksiReward).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membatalkan pengajuan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Pengajuan redeem berhasil dibatalkan",
		"transaksi_id": transaksiID,
	})
}

// GetListRedeemBSU: Mengambil daftar transaksi redeem berdasarkan bank_id.
// - BSU: melihat pengajuan yang mereka buat (bank_asal_id)
// - BSI/BSM: melihat semua permintaan masuk (bank_tujuan_id)
// Query param opsional: ?status=waiting|approved|rejected|canceled|success|failed
func (rbc *RedeemBSUController) GetListRedeemBSU(c *gin.Context) {
	bankID := c.Param("bank_id")
	statusFilter := c.Query("status") // opsional

	// Validasi bank
	var bank models.BankSampah
	if err := rbc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Bank tidak ditemukan"})
		return
	}

	// Tentukan kolom filter berdasarkan jenis bank
	var filterCol string
	if bank.JenisBank == models.BSU {
		filterCol = "bank_asal_id" // BSU melihat pengajuan milik mereka
	} else {
		filterCol = "bank_tujuan_id" // BSI/BSM melihat permintaan masuk
	}

	query := rbc.DB.
		Preload("Reward").
		Preload("BankAsal").
		Preload("BankTujuan").
		Preload("Nasabah").
		Where(filterCol+" = ?", bankID)

	// Terapkan filter status jika ada
	if statusFilter != "" {
		query = query.Where("status_transaksi = ?", statusFilter)
	}

	var transaksiList []models.TransaksiReward
	if err := query.Order("created_at DESC").Find(&transaksiList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengambil data transaksi reward"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data transaksi reward berhasil diambil",
		"total":   len(transaksiList),
		"data":    transaksiList,
	})
}

// GetDetailRedeemBSU: Mengambil detail satu transaksi redeem beserta
// relasi lengkap dan detail sembako (jika ada).
func (rbc *RedeemBSUController) GetDetailRedeemBSU(c *gin.Context) {
	redeemID := c.Param("redeem_id")

	var transaksiReward models.TransaksiReward
	err := rbc.DB.
		Preload("Reward").
		Preload("BankAsal").
		Preload("BankTujuan").
		Preload("Nasabah").
		Preload("Details").
		Preload("Details.Sembako").
		Where("transaksi_id = ?", redeemID).
		First(&transaksiReward).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Transaksi tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail transaksi reward berhasil diambil",
		"data":    transaksiReward,
	})
}

// VerifikasiManualTransaksiReward: Memverifikasi keberadaan dan keabsahan sebuah transaksi reward.
// Digunakan untuk keperluan verifikasi manual (misal: scan QR atau input kode transaksi).
// Mengembalikan "verified" beserta ringkasan transaksi jika valid.
func (rbc *RedeemBSUController) VerifikasiManualTransaksiReward(c *gin.Context) {
	transaksiID := c.Param("transaksi_id")
	adminBSIID := c.Param("admin_bsi_id")

	// Verifikasi identitas: hanya admin_bsi atau petugas_bsi yang boleh mengkonfirmasi
	var staffBSI models.Admin
	if err := rbc.DB.
		Where("admin_id = ? AND role IN ?", adminBSIID, []models.RoleAdmin{models.AdminBSI, models.PetugasBSI}).
		First(&staffBSI).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "not_verified",
			"message": "Tidak memiliki hak akses. Hanya admin_bsi atau petugas_bsi yang diizinkan",
		})
		return
	}

	var transaksiReward models.TransaksiReward
	err := rbc.DB.
		Preload("Reward").
		Preload("BankAsal").
		Preload("BankTujuan").
		Where("transaksi_id = ?", transaksiID).
		First(&transaksiReward).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "not_verified",
			"message": "Transaksi tidak ditemukan",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "verified",
		"message": "Transaksi ditemukan dan valid",
		"data": gin.H{
			"transaksi_id":    transaksiReward.TransaksiID,
			"jenis_transaksi": string(transaksiReward.JenisTransaksi),
			"status_transaksi": string(transaksiReward.StatusTransaksi),
			"poin":            transaksiReward.Poin,
			"nominal":         transaksiReward.Nominal,
			"reward":          transaksiReward.Reward.NamaReward,
			"bank_asal":       transaksiReward.BankAsal,
			"bank_tujuan":     transaksiReward.BankTujuan,
			"created_at":      transaksiReward.CreatedAt,
		},
	})
}