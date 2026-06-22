package repositories

import (
	"enviroo-be/internal/models"
	"time"

	"gorm.io/gorm"
)

type DistribusiSisaRepo struct {
	DB *gorm.DB
}

func NewDistribusiSisaRepo(db *gorm.DB) *DistribusiSisaRepo {
	return &DistribusiSisaRepo{DB: db}
}

// ── Bank & Setting ──────────────────────────────────────────────────────────

func (r *DistribusiSisaRepo) GetBankByID(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.DB.Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *DistribusiSisaRepo) GetBankByIDAndJenis(bankID string, jenis []models.JenisBank) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.DB.Where("bank_id = ? AND jenis_bank IN ?", bankID, jenis).First(&bank).Error
	return &bank, err
}

func (r *DistribusiSisaRepo) GetSetting(bankID string) (*models.SettingSisaBagiHasil, error) {
	var s models.SettingSisaBagiHasil
	err := r.DB.Where("bank_id = ?", bankID).First(&s).Error
	return &s, err
}

func (r *DistribusiSisaRepo) CreateSetting(s *models.SettingSisaBagiHasil) error {
	return r.DB.Create(s).Error
}

func (r *DistribusiSisaRepo) SaveSetting(s *models.SettingSisaBagiHasil) error {
	return r.DB.Save(s).Error
}

// ── Preview & Submit ────────────────────────────────────────────────────────

func (r *DistribusiSisaRepo) GetBagiHasilWithBank(bagiHasilID string) (*models.BagiHasil, error) {
	var bh models.BagiHasil
	err := r.DB.Preload("Bank").Where("bagi_hasil_id = ?", bagiHasilID).First(&bh).Error
	return &bh, err
}

// FindDistribusiByBagiHasil returns (record, found). found=true means a conflict exists.
func (r *DistribusiSisaRepo) FindDistribusiByBagiHasil(bagiHasilID string) (*models.DistribusiSisa, bool) {
	var d models.DistribusiSisa
	if err := r.DB.Where("bagi_hasil_id = ?", bagiHasilID).First(&d).Error; err != nil {
		return nil, false
	}
	return &d, true
}

func (r *DistribusiSisaRepo) GetPenerimaNasabah(db *gorm.DB, bagiHasilID string) ([]models.PenerimaBagiHasil, error) {
	var penerima []models.PenerimaBagiHasil
	err := db.Preload("Nasabah").
		Where("bagi_hasil_id = ? AND nasabah_id IS NOT NULL", bagiHasilID).
		Find(&penerima).Error
	return penerima, err
}

func (r *DistribusiSisaRepo) GetBSUList(db *gorm.DB, bsiID string) ([]models.BankSampah, error) {
	var list []models.BankSampah
	err := db.Where("parent_bank_id = ? AND jenis_bank = ?", bsiID, models.BSU).Find(&list).Error
	return list, err
}

func (r *DistribusiSisaRepo) GetRewardByNama(nama models.RewardEnum) (*models.Reward, error) {
	var reward models.Reward
	err := r.DB.Where("nama_reward = ?", nama).First(&reward).Error
	return &reward, err
}

func (r *DistribusiSisaRepo) GetDetailPenjualan(db *gorm.DB, penjualanID string) ([]models.DetailPenjualan, error) {
	var details []models.DetailPenjualan
	err := db.Where("penjualan_id = ?", penjualanID).Find(&details).Error
	return details, err
}

type KontribusiBSURow struct {
	BankID       string
	NilaiDipakai float64
}

type TransportSplitRow struct {
	BankID        string
	QtyMandiri    float64
	QtyNonMandiri float64
}

func (r *DistribusiSisaRepo) GetKontribusiPerBSU(db *gorm.DB, bagiHasilID string) ([]KontribusiBSURow, error) {
	var rows []KontribusiBSURow
	err := db.Table("perhitungan_sisa ps").
		Select("ts.bank_id, SUM(ps.nilai_dipakai) AS nilai_dipakai").
		Joins("JOIN tabungan_sampah ts ON ts.tabungan_id = ps.tabungan_id").
		Where("ps.bagi_hasil_id = ? AND ps.penerima_sisa_id IS NULL", bagiHasilID).
		Group("ts.bank_id").
		Scan(&rows).Error
	return rows, err
}

// GetTransportSplitPerBSU returns, per BSU, how many kg was delivered mandiri (is_mandiri=true)
// vs non-mandiri for the given bagi_hasil, by joining perhitungan_sisa → tabungan_sampah → pengangkutan_sampah.
func (r *DistribusiSisaRepo) GetTransportSplitPerBSU(db *gorm.DB, bagiHasilID string) ([]TransportSplitRow, error) {
	var rows []TransportSplitRow
	err := db.Table("perhitungan_sisa ps").
		Select(`ts.bank_id,
			SUM(CASE WHEN pg.is_mandiri = true THEN ps.qty_dipakai ELSE 0 END) AS qty_mandiri,
			SUM(CASE WHEN pg.is_mandiri = false THEN ps.qty_dipakai ELSE 0 END) AS qty_non_mandiri`).
		Joins("JOIN tabungan_sampah ts ON ts.tabungan_id = ps.tabungan_id").
		Joins("JOIN pengangkutan_sampah pg ON pg.pengangkutan_id = ts.source_id").
		Where("ps.bagi_hasil_id = ? AND ps.penerima_sisa_id IS NULL", bagiHasilID).
		Group("ts.bank_id").
		Scan(&rows).Error
	return rows, err
}

func (r *DistribusiSisaRepo) SetPenerimaSisaID(db *gorm.DB, bagiHasilID, bsuID, penerimaSisaID string) error {
	return db.Exec(`
		UPDATE perhitungan_sisa ps
		SET penerima_sisa_id = ?
		FROM tabungan_sampah ts
		WHERE ts.tabungan_id = ps.tabungan_id
		  AND ps.bagi_hasil_id = ?
		  AND ts.bank_id = ?`,
		penerimaSisaID, bagiHasilID, bsuID).Error
}

func (r *DistribusiSisaRepo) CreateDistribusiSisa(db *gorm.DB, d *models.DistribusiSisa) error {
	return db.Create(d).Error
}

func (r *DistribusiSisaRepo) CreatePenerimaSisa(db *gorm.DB, p *models.PenerimaDistribusiSisa) error {
	return db.Create(p).Error
}

func (r *DistribusiSisaRepo) UpdateSaldoRekening(db *gorm.DB, bankID *string, nasabahID *string, rewardID int, reward models.Reward, nilai float64, entitas models.EntitasEnum, adminID string, now time.Time) error {
	return UpdateSaldoRekening(db, bankID, nasabahID, rewardID, reward, nilai, entitas, adminID, now)
}

func (r *DistribusiSisaRepo) GetAdminsByBankID(bankID string) ([]models.Admin, error) {
	var admins []models.Admin
	err := r.DB.Preload("User").Where("bank_id = ?", bankID).Find(&admins).Error
	return admins, err
}

// ── Detail & List ───────────────────────────────────────────────────────────

func (r *DistribusiSisaRepo) GetDistribusiSisa(distribusiID string) (*models.DistribusiSisa, error) {
	var d models.DistribusiSisa
	err := r.DB.Preload("BagiHasil.Bank").Where("distribusi_id = ?", distribusiID).First(&d).Error
	return &d, err
}

func (r *DistribusiSisaRepo) GetAdminNama(adminID string) string {
	var admin models.Admin
	if err := r.DB.Preload("User").Where("admin_id = ?", adminID).First(&admin).Error; err != nil {
		return ""
	}
	return admin.User.Nama
}

func (r *DistribusiSisaRepo) GetPenerimaBSI(distribusiID string) (*models.PenerimaDistribusiSisa, error) {
	var p models.PenerimaDistribusiSisa
	err := r.DB.Preload("BankSampah").
		Where("distribusi_id = ? AND jenis_penerimaan = ?", distribusiID, models.JenisTerimaBagiHasilBSI).
		First(&p).Error
	return &p, err
}

func (r *DistribusiSisaRepo) GetListPenerimaBSU(distribusiID string) ([]models.PenerimaDistribusiSisa, error) {
	var list []models.PenerimaDistribusiSisa
	err := r.DB.Preload("BankSampah").
		Where("distribusi_id = ? AND jenis_penerimaan = ?", distribusiID, models.JenisTerimaBagiHasilBSU).
		Find(&list).Error
	return list, err
}

func (r *DistribusiSisaRepo) GetPenerimaByBank(bankID string, jenis models.JenisTerimaEnum, startDate, endDate time.Time) ([]models.PenerimaDistribusiSisa, error) {
	var penerima []models.PenerimaDistribusiSisa
	q := r.DB.Preload("DistribusiSisa").
		Joins("JOIN distribusi_sisa ON distribusi_sisa.distribusi_id = penerima_distribusi_sisa.distribusi_id").
		Where("penerima_distribusi_sisa.bank_id = ? AND penerima_distribusi_sisa.jenis_penerimaan = ?", bankID, jenis)
	if !startDate.IsZero() {
		q = q.Where("distribusi_sisa.created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		q = q.Where("distribusi_sisa.created_at <= ?", endDate)
	}
	err := q.Find(&penerima).Error
	return penerima, err
}

func (r *DistribusiSisaRepo) GetPenerimaSisaWithDetails(penerimaSisaID string) (*models.PenerimaDistribusiSisa, error) {
	var p models.PenerimaDistribusiSisa
	err := r.DB.Preload("BankSampah").Preload("DistribusiSisa.BagiHasil").
		Where("penerima_sisa_id = ?", penerimaSisaID).First(&p).Error
	return &p, err
}

func (r *DistribusiSisaRepo) GetPerhitunganSisa(penerimaSisaID string) ([]models.PerhitunganSisa, error) {
	var ph []models.PerhitunganSisa
	err := r.DB.Preload("TabunganSampah.KatalogSampah.Sarok").
		Where("penerima_sisa_id = ?", penerimaSisaID).Find(&ph).Error
	return ph, err
}
