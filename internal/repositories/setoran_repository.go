package repositories

import (
	"enviroo-be/internal/models"
	"time"

	"gorm.io/gorm"
)

// ── Row types untuk raw query ──────────────────────────────────────────────

type SetoranDetailHeader struct {
	SetoranID          string               `json:"setoran_id"          gorm:"column:setoran_id"`
	NamaPetugas        string               `json:"nama_petugas"        gorm:"column:nama_petugas"`
	NamaNasabah        string               `json:"nama_nasabah"        gorm:"column:nama_nasabah"`
	TransaksiTimestamp time.Time            `json:"transaksi_timestamp" gorm:"column:transaksi_timestamp"`
	TotalItem          int                  `json:"total_item"          gorm:"column:total_item"`
	StatusSetoran      models.StatusSetoran `json:"status_setoran"      gorm:"column:status_setoran"`
	BuktiViaManual     string               `json:"bukti_via_manual"    gorm:"column:bukti_via_manual"`
}

type SetoranDetailItem struct {
	NamaSampah string  `json:"nama_sampah" gorm:"column:nama_sampah"`
	Satuan     string  `json:"satuan"      gorm:"column:satuan"`
	Qty        float64 `json:"qty"         gorm:"column:qty"`
}

type RiwayatSetoranRow struct {
	SetoranID          string               `json:"setoran_id"          gorm:"column:setoran_id"`
	NamaPetugas        string               `json:"nama_petugas"        gorm:"column:nama_petugas"`
	TransaksiTimestamp time.Time            `json:"transaksi_timestamp" gorm:"column:transaksi_timestamp"`
	TotalItem          int                  `json:"total_item"          gorm:"column:total_item"`
	StatusSetoran      models.StatusSetoran `json:"status_setoran"      gorm:"column:status_setoran"`
}

// ── Repo ───────────────────────────────────────────────────────────────────

type SetoranRepo struct {
	db *gorm.DB
}

func NewSetoranRepo(db *gorm.DB) *SetoranRepo {
	return &SetoranRepo{db: db}
}

// ── Non-transactional reads ────────────────────────────────────────────────

func (r *SetoranRepo) FindNasabahAktifWithUser(nasabahID string) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.db.Preload("User").
		Where("nasabah_id = ? AND status_nasabah = ?", nasabahID, models.Aktif).
		First(&n).Error
	return &n, err
}

func (r *SetoranRepo) FindNasabahWithUser(nasabahID string) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.db.Preload("User").Where("nasabah_id = ?", nasabahID).First(&n).Error
	return &n, err
}

func (r *SetoranRepo) FindAdminAktif(adminID string) (*models.Admin, error) {
	var a models.Admin
	err := r.db.Where("admin_id = ? AND status_admin = ?", adminID, models.Aktif).First(&a).Error
	return &a, err
}

func (r *SetoranRepo) FindPenimbanganAktif(penimbanganID string) (*models.Penimbangan, error) {
	var p models.Penimbangan
	err := r.db.Where("penimbangan_id = ? AND status_penimbangan = ?", penimbanganID, models.StatusAktif).
		First(&p).Error
	return &p, err
}

func (r *SetoranRepo) FindKatalogSampah(sampahID string) (*models.KatalogSampah, error) {
	var s models.KatalogSampah
	err := r.db.Preload("Reward").Preload("Sarok").Where("sampah_id = ?", sampahID).First(&s).Error
	return &s, err
}

func (r *SetoranRepo) FindBankSampah(bankID string) (*models.BankSampah, error) {
	var b models.BankSampah
	err := r.db.Where("bank_id = ?", bankID).First(&b).Error
	return &b, err
}

func (r *SetoranRepo) GetDetailHeader(setoranID string) (*SetoranDetailHeader, error) {
	var h SetoranDetailHeader
	err := r.db.Table("setoran_nasabah").
		Select(`setoran_nasabah.setoran_id,
			u_petugas.nama  AS nama_petugas,
			u_nasabah.nama  AS nama_nasabah,
			setoran_nasabah.created_at  AS transaksi_timestamp,
			setoran_nasabah.total_item,
			setoran_nasabah.status_setoran,
			setoran_nasabah.bukti_via_manual`).
		Joins("LEFT JOIN admin ON admin.admin_id = setoran_nasabah.admin_id").
		Joins("LEFT JOIN users u_petugas ON u_petugas.user_id = admin.user_id").
		Joins("LEFT JOIN nasabah ON nasabah.nasabah_id = setoran_nasabah.nasabah_id").
		Joins("LEFT JOIN users u_nasabah ON u_nasabah.user_id = nasabah.user_id").
		Where("setoran_nasabah.setoran_id = ?", setoranID).
		First(&h).Error
	return &h, err
}

func (r *SetoranRepo) GetDetailItems(setoranID string) ([]SetoranDetailItem, error) {
	var items []SetoranDetailItem
	err := r.db.Table("detail_setoran_nasabah").
		Select("sampah.nama_sampah, sampah.satuan, detail_setoran_nasabah.qty").
		Joins("LEFT JOIN katalog_sampah ON katalog_sampah.sampah_id = detail_setoran_nasabah.sampah_id").
		Joins("LEFT JOIN sampah ON sampah.sarok_id = katalog_sampah.sarok_id").
		Where("detail_setoran_nasabah.setoran_id = ?", setoranID).
		Find(&items).Error
	return items, err
}

func (r *SetoranRepo) GetRiwayat(nasabahID, startDate, endDate string) ([]RiwayatSetoranRow, error) {
	var rows []RiwayatSetoranRow
	q := r.db.Table("setoran_nasabah").
		Select(`setoran_nasabah.setoran_id,
			u_petugas.nama AS nama_petugas,
			setoran_nasabah.created_at AS transaksi_timestamp,
			setoran_nasabah.total_item,
			setoran_nasabah.status_setoran`).
		Joins("LEFT JOIN admin ON admin.admin_id = setoran_nasabah.admin_id").
		Joins("LEFT JOIN users u_petugas ON u_petugas.user_id = admin.user_id").
		Where("setoran_nasabah.nasabah_id = ?", nasabahID)

	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			q = q.Where("setoran_nasabah.created_at >= ?", t)
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			q = q.Where("setoran_nasabah.created_at <= ?", t.Add(24*time.Hour-time.Second))
		}
	}

	err := q.Order("setoran_nasabah.created_at DESC").Find(&rows).Error
	return rows, err
}

// ── Transactional (caller menyediakan tx dari db.Transaction) ─────────────

func (r *SetoranRepo) FindPenimbanganTx(tx *gorm.DB, penimbanganID string) (*models.Penimbangan, error) {
	var p models.Penimbangan
	err := tx.Where("penimbangan_id = ? AND status_penimbangan = ?", penimbanganID, models.StatusAktif).
		First(&p).Error
	return &p, err
}

func (r *SetoranRepo) CreateSetoran(tx *gorm.DB, s *models.SetoranNasabah) error {
	return tx.Create(s).Error
}

func (r *SetoranRepo) CreateDetailSetoran(tx *gorm.DB, d *models.DetailSetoranNasabah) error {
	return tx.Create(d).Error
}

// UpsertStokSampah menambah stok jika sudah ada, atau membuat baris baru jika belum.
func (r *SetoranRepo) UpsertStokSampah(tx *gorm.DB, bankID, sampahID string, qty float64) error {
	var stok models.StokSampah
	res := tx.Where("bank_id = ? AND sampah_id = ?", bankID, sampahID).First(&stok)
	switch res.Error {
	case gorm.ErrRecordNotFound:
		return tx.Create(&models.StokSampah{BankID: bankID, SampahID: sampahID, Stok: qty}).Error
	case nil:
		return tx.Model(&stok).Update("stok", gorm.Expr("stok + ?", qty)).Error
	default:
		return res.Error
	}
}

func (r *SetoranRepo) CreateTabunganSampah(tx *gorm.DB, t *models.TabunganSampah) error {
	return tx.Create(t).Error
}
