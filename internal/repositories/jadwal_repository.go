package repositories

import (
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
)

type JadwalLengkapRow struct {
	JadwalID          string           `gorm:"column:jadwal_id"`
	BankID            string           `gorm:"column:bank_id"`
	NamaBank          string           `gorm:"column:nama_bank"`
	JenisBank         models.JenisBank `gorm:"column:jenis_bank"`
	Hari              *models.HariEnum `gorm:"column:hari"`
	MingguKe          *int             `gorm:"column:minggu_ke"`
	JamMulai          string           `gorm:"column:jam_mulai"`
	JamSelesai        string           `gorm:"column:jam_selesai"`
	IsActive          *bool            `gorm:"column:is_active"`
	IsRutin           *bool            `gorm:"column:is_rutin"`
	Tanggal           *time.Time       `gorm:"column:tanggal"`
	NamaJadwalSpesial string           `gorm:"column:nama_jadwal_spesial"`
	TargetBankID      string           `gorm:"column:target_bank_id"`
	TargetBankName    string           `gorm:"column:target_bank_name"`
	CreatedAt         time.Time        `gorm:"column:created_at"`
}

type JadwalBankRow struct {
	models.Jadwal
	BankName       string `gorm:"column:bank_name"       json:"bank_name"`
	TargetBankName string `gorm:"column:target_bank_name" json:"target_bank_name"`
}

type PemakaianJadwal struct {
	Label  string
	Jumlah int64
}

var tabelPemakaiJadwal = []struct {
	Tabel string
	Label string
}{
	{"penimbangan", "sesi penimbangan"},
	{"pengangkutan_sampah", "pengangkutan sampah"},
}

type JadwalRepo struct {
	db *gorm.DB
}

func NewJadwalRepo(db *gorm.DB) *JadwalRepo {
	return &JadwalRepo{db: db}
}

func (r *JadwalRepo) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *JadwalRepo) FindBank(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.db.Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *JadwalRepo) FindJadwal(tx *gorm.DB, jadwalID string) (*models.Jadwal, error) {
	var jadwal models.Jadwal
	err := r.conn(tx).Where("jadwal_id = ?", jadwalID).First(&jadwal).Error
	return &jadwal, err
}

func (r *JadwalRepo) ListJadwalBank(bankID string, jenis models.JadwalEnum) ([]models.Jadwal, error) {
	var jadwals []models.Jadwal
	err := r.db.Where("bank_id = ? AND jenis_jadwal = ?", bankID, jenis).Find(&jadwals).Error
	return jadwals, err
}

func (r *JadwalRepo) ListJadwalBankWithNama(bankID string, jenis models.JadwalEnum, sebagaiTarget bool, month, year int) ([]JadwalBankRow, error) {
	q := r.db.Table("jadwal").
		Select("jadwal.*, b1.nama_bank AS bank_name, b2.nama_bank AS target_bank_name").
		Joins("LEFT JOIN bank_sampah b1 ON b1.bank_id = jadwal.bank_id").
		Joins("LEFT JOIN bank_sampah b2 ON b2.bank_id = jadwal.target_bank_id").
		Where("jadwal.jenis_jadwal = ?", jenis)

	if sebagaiTarget {
		q = q.Where("jadwal.target_bank_id = ?", bankID)
	} else {
		q = q.Where("jadwal.bank_id = ?", bankID)
	}

	if month >= 1 && month <= 12 && year > 0 {
		q = q.Where(
			"jadwal.is_rutin = true OR (EXTRACT(MONTH FROM jadwal.tanggal) = ? AND EXTRACT(YEAR FROM jadwal.tanggal) = ?)",
			month, year,
		)
	}

	var rows []JadwalBankRow
	err := q.Find(&rows).Error
	return rows, err
}

func (r *JadwalRepo) ListSemuaJadwal(jenis models.JadwalEnum) ([]JadwalLengkapRow, error) {
	rows := []JadwalLengkapRow{}
	err := r.db.Table("jadwal").
		Select(`jadwal.jadwal_id, jadwal.bank_id, b1.nama_bank, b1.jenis_bank,
			jadwal.hari, jadwal.minggu_ke, jadwal.jam_mulai, jadwal.jam_selesai,
			jadwal.is_active, jadwal.is_rutin, jadwal.tanggal, jadwal.nama_jadwal_spesial,
			jadwal.target_bank_id, b2.nama_bank as target_bank_name, jadwal.created_at`).
		Joins("JOIN bank_sampah b1 ON b1.bank_id = jadwal.bank_id").
		Joins("LEFT JOIN bank_sampah b2 ON b2.bank_id = jadwal.target_bank_id").
		Where("jadwal.jenis_jadwal = ?", jenis).
		Order("b1.nama_bank, b2.nama_bank, jadwal.created_at").
		Scan(&rows).Error
	return rows, err
}

func (r *JadwalRepo) ListStatusSesi(jadwalIDs []string, dari, sampai time.Time) (map[string]string, error) {
	hasil := make(map[string]string, len(jadwalIDs))
	if len(jadwalIDs) == 0 {
		return hasil, nil
	}

	type sesiRow struct {
		JadwalID string `gorm:"column:jadwal_id"`
		Tanggal  string `gorm:"column:tanggal_sesi"`
		Status   string `gorm:"column:status_penimbangan"`
	}

	var rows []sesiRow
	if err := r.db.Table("penimbangan").
		Select("jadwal_id, TO_CHAR(tanggal_sesi, 'YYYY-MM-DD') AS tanggal_sesi, status_penimbangan").
		Where("jadwal_id IN ? AND tanggal_sesi BETWEEN ? AND ?", jadwalIDs, dari, sampai).
		Order("penimbangan_id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		key := row.JadwalID + "|" + row.Tanggal
		if _, ada := hasil[key]; !ada {
			hasil[key] = row.Status
		}
	}
	return hasil, nil
}

func (r *JadwalRepo) CreateJadwal(tx *gorm.DB, jadwal *models.Jadwal) error {
	return r.conn(tx).Create(jadwal).Error
}

func (r *JadwalRepo) SaveJadwal(tx *gorm.DB, jadwal *models.Jadwal) error {
	return r.conn(tx).Save(jadwal).Error
}

func (r *JadwalRepo) DeleteJadwal(tx *gorm.DB, jadwal *models.Jadwal) error {
	return r.conn(tx).Delete(jadwal).Error
}

func (r *JadwalRepo) CountPemakaianJadwal(jadwalID string) ([]PemakaianJadwal, error) {
	var hasil []PemakaianJadwal
	for _, t := range tabelPemakaiJadwal {
		var count int64
		if err := r.db.Table(t.Tabel).Where("jadwal_id = ?", jadwalID).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			hasil = append(hasil, PemakaianJadwal{Label: t.Label, Jumlah: count})
		}
	}
	return hasil, nil
}

func (r *JadwalRepo) FindAdminIDByUserBank(userID, bankID string) (string, error) {
	var adminID string
	err := r.db.Model(&models.Admin{}).
		Where("user_id = ? AND bank_id = ? AND status_admin = ?", userID, bankID, models.Aktif).
		Limit(1).
		Pluck("admin_id", &adminID).Error
	return adminID, err
}

func (r *JadwalRepo) CountSesiPenimbangan(jadwalID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Penimbangan{}).Where("jadwal_id = ?", jadwalID).Count(&count).Error
	return count, err
}
