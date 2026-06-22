package repositories

import (
	"enviroo-be/internal/models"
	"time"

	"gorm.io/gorm"
)

type BagiHasilRepo struct {
	DB *gorm.DB
}

func NewBagiHasilRepo(db *gorm.DB) *BagiHasilRepo {
	return &BagiHasilRepo{DB: db}
}

// ── Basic reads ─────────────────────────────────────────────────────────────

func (r *BagiHasilRepo) GetBankByID(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.DB.Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *BagiHasilRepo) GetPenjualanWithReward(penjualanID, bankID string) (*models.Penjualan, error) {
	var p models.Penjualan
	err := r.DB.Preload("Reward").
		Where("penjualan_id = ? AND bank_id = ?", penjualanID, bankID).
		First(&p).Error
	return &p, err
}

func (r *BagiHasilRepo) GetDetailPenjualan(db *gorm.DB, penjualanID string) ([]models.DetailPenjualan, error) {
	var details []models.DetailPenjualan
	err := db.Where("penjualan_id = ?", penjualanID).Find(&details).Error
	return details, err
}

func (r *BagiHasilRepo) GetBSUList(db *gorm.DB, parentBankID string) ([]models.BankSampah, error) {
	var list []models.BankSampah
	err := db.Where("parent_bank_id = ? AND jenis_bank = ?", parentBankID, models.BSU).Find(&list).Error
	return list, err
}

func (r *BagiHasilRepo) GetTabunganNasabah(db *gorm.DB, sampahID string, bankIDs []string) ([]models.TabunganSampah, error) {
	var tabungan []models.TabunganSampah
	err := db.Where("entitas = ? AND sampah_id = ? AND sisa_qty > 0 AND bank_id IN ?",
		models.EntitasNasabah, sampahID, bankIDs).
		Order("created_at ASC").Find(&tabungan).Error
	return tabungan, err
}

func (r *BagiHasilRepo) GetTabunganBSU(db *gorm.DB, sampahID string, bsuIDs []string) ([]models.TabunganSampah, error) {
	var tabungan []models.TabunganSampah
	err := db.Where("entitas = ? AND sampah_id = ? AND sisa_qty > 0 AND bank_id IN ?",
		models.EntitasBankSampah, sampahID, bsuIDs).
		Order("created_at ASC").Find(&tabungan).Error
	return tabungan, err
}

func (r *BagiHasilRepo) CreatePerhitunganSisa(db *gorm.DB, ps *models.PerhitunganSisa) error {
	return db.Create(ps).Error
}

// NasabahRow is a projection from the batch nasabah info query.
type NasabahRow struct {
	NasabahID   string
	NamaNasabah string
	BankID      string
	NamaBank    string
}

func (r *BagiHasilRepo) GetNasabahInfoBatch(nasabahIDs []string) ([]NasabahRow, error) {
	var rows []NasabahRow
	err := r.DB.Table("nasabah").
		Select("nasabah.nasabah_id, users.nama AS nama_nasabah, nasabah.bank_id, bank_sampah.nama_bank").
		Joins("JOIN users ON users.user_id = nasabah.user_id").
		Joins("JOIN bank_sampah ON bank_sampah.bank_id = nasabah.bank_id").
		Where("nasabah.nasabah_id IN ?", nasabahIDs).
		Scan(&rows).Error
	return rows, err
}

func (r *BagiHasilRepo) GetNasabahWithUser(nasabahID string) (*models.Nasabah, error) {
	var nasabah models.Nasabah
	err := r.DB.Preload("User").Where("nasabah_id = ?", nasabahID).First(&nasabah).Error
	return &nasabah, err
}

// ── Writes (accept tx or main DB) ──────────────────────────────────────────

func (r *BagiHasilRepo) SaveTabungan(db *gorm.DB, t *models.TabunganSampah) error {
	return db.Save(t).Error
}

// UpsertSchemaHargaNasabah creates or updates the nasabah price schema,
// recording a history entry when the price changes.
func (r *BagiHasilRepo) UpsertSchemaHargaNasabah(db *gorm.DB, sampahID string, harga float64, satuan models.SatuanRewardEnum, adminID string) error {
	var schema models.SchemaHargaSampah
	if err := db.Where("sampah_id = ? AND level_user = ?", sampahID, models.LevelNasabah).First(&schema).Error; err != nil {
		return db.Create(&models.SchemaHargaSampah{
			SampahID: sampahID, LevelUser: models.LevelNasabah,
			Harga: harga, SatuanReward: satuan,
		}).Error
	}
	if schema.Harga == harga {
		return nil
	}
	if err := db.Create(&models.KatalogSampahHistory{
		SchemaID:  int(schema.SchemaID),
		HargaLama: schema.Harga,
		HargaBaru: harga,
		ChangedBy: adminID,
	}).Error; err != nil {
		return err
	}
	schema.Harga = harga
	return db.Save(&schema).Error
}

func (r *BagiHasilRepo) CreateBagiHasil(db *gorm.DB, bh *models.BagiHasil) error {
	return db.Create(bh).Error
}

func (r *BagiHasilRepo) CreatePenerimaBagiHasil(db *gorm.DB, p *models.PenerimaBagiHasil) error {
	return db.Create(p).Error
}

func (r *BagiHasilRepo) CreateDetailBagiHasil(db *gorm.DB, d *models.DetailBagiHasil) error {
	return db.Create(d).Error
}

func (r *BagiHasilRepo) CreatePencairanTabungan(db *gorm.DB, p *models.PencairanTabungan) error {
	return db.Create(p).Error
}

func (r *BagiHasilRepo) UpdatePenjualanStatus(db *gorm.DB, penjualan *models.Penjualan, status models.StatusBagiHasilEnum) error {
	return db.Model(penjualan).Update("status_bagi_hasil", status).Error
}

func (r *BagiHasilRepo) UpdateSaldoRekening(db *gorm.DB, bankID *string, nasabahID *string, rewardID int, reward models.Reward, nilai float64, entitas models.EntitasEnum, adminID string, now time.Time) error {
	return UpdateSaldoRekening(db, bankID, nasabahID, rewardID, reward, nilai, entitas, adminID, now)
}

// ── Read endpoints ──────────────────────────────────────────────────────────

func (r *BagiHasilRepo) GetBagiHasilByPenjualan(penjualanID string) (*models.BagiHasil, error) {
	var bh models.BagiHasil
	err := r.DB.Preload("Penjualan.Reward").Preload("Bank").
		Where("penjualan_id = ?", penjualanID).First(&bh).Error
	return &bh, err
}

func (r *BagiHasilRepo) GetBagiHasilWithAll(bagiHasilID string) (*models.BagiHasil, error) {
	var bh models.BagiHasil
	err := r.DB.Preload("Penjualan.Reward").Preload("Bank").
		Where("bagi_hasil_id = ?", bagiHasilID).First(&bh).Error
	return &bh, err
}

func (r *BagiHasilRepo) GetPenerimaBagiHasil(bagiHasilID string) ([]models.PenerimaBagiHasil, error) {
	var list []models.PenerimaBagiHasil
	err := r.DB.Preload("Nasabah.User").Preload("Nasabah.Bank").
		Where("bagi_hasil_id = ?", bagiHasilID).Find(&list).Error
	return list, err
}

func (r *BagiHasilRepo) GetPenerimaNasabahFull(bagiHasilID string) ([]models.PenerimaBagiHasil, error) {
	var list []models.PenerimaBagiHasil
	err := r.DB.Preload("Nasabah.User").Preload("Nasabah.Bank").
		Where("bagi_hasil_id = ? AND nasabah_id IS NOT NULL", bagiHasilID).Find(&list).Error
	return list, err
}

func (r *BagiHasilRepo) GetNamaPetugas(adminID string) string {
	var nama string
	r.DB.Table("admin").Select("users.nama").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("admin.admin_id = ?", adminID).Scan(&nama)
	return nama
}

// FindDistribusiSisaID returns the distribusi_id if already distributed, nil otherwise.
func (r *BagiHasilRepo) FindDistribusiSisaID(bagiHasilID string) *string {
	var d models.DistribusiSisa
	if err := r.DB.Where("bagi_hasil_id = ?", bagiHasilID).First(&d).Error; err != nil {
		return nil
	}
	return &d.DistribusiID
}

func (r *BagiHasilRepo) GetPenerimaByNasabah(nasabahID, startDate, endDate string) ([]models.PenerimaBagiHasil, error) {
	var list []models.PenerimaBagiHasil
	q := r.DB.Preload("BagiHasil.Penjualan.Reward").
		Where("nasabah_id = ?", nasabahID)

	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			q = q.Where("created_at <= ?", t.Add(24*time.Hour-time.Second))
		}
	}

	err := q.Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *BagiHasilRepo) GetPenerimaBagiHasilNasabah(penerimaID string) (*models.PenerimaBagiHasil, error) {
	var p models.PenerimaBagiHasil
	err := r.DB.Preload("BagiHasil.Penjualan.Reward").Preload("Nasabah.User").
		Where("penerima_id = ? AND nasabah_id IS NOT NULL", penerimaID).First(&p).Error
	return &p, err
}

func (r *BagiHasilRepo) GetDetailBagiHasilItems(penerimaID string) ([]models.DetailBagiHasil, error) {
	var details []models.DetailBagiHasil
	err := r.DB.Preload("KatalogSampah.Sarok").
		Where("penerima_id = ?", penerimaID).Find(&details).Error
	return details, err
}

func (r *BagiHasilRepo) GetBagiHasilByBankWithFilter(bankID string, startTime, endTime time.Time) ([]models.BagiHasil, error) {
	query := r.DB.Preload("Penjualan.Reward").Where("bank_id = ?", bankID)
	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}
	var list []models.BagiHasil
	err := query.Order("created_at DESC").Find(&list).Error
	return list, err
}
