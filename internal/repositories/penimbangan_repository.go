package repositories

import (
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SesiHariIniRow struct {
	PenimbanganID     string                   `gorm:"column:penimbangan_id"`
	TanggalSesi       time.Time                `gorm:"column:tanggal_sesi"`
	JamMulai          string                   `gorm:"column:jam_mulai"`
	JamSelesai        string                   `gorm:"column:jam_selesai"`
	NamaJadwalSpesial string                   `gorm:"column:nama_jadwal_spesial"`
	IsRutin           *bool                    `gorm:"column:is_rutin"`
	StatusPenimbangan models.StatusPenimbangan `gorm:"column:status_penimbangan"`
}

type RiwayatPenimbanganRow struct {
	PenimbanganID     string                   `gorm:"column:penimbangan_id"`
	TanggalSesi       time.Time                `gorm:"column:tanggal_sesi"`
	StatusPenimbangan models.StatusPenimbangan `gorm:"column:status_penimbangan"`
	JumlahSetoran     int64                    `gorm:"column:jumlah_setoran"`
}

type SetoranSesiRow struct {
	SetoranID          string    `json:"setoran_id" gorm:"column:setoran_id"`
	NamaPetugas        string    `json:"nama_petugas" gorm:"column:nama_petugas"`
	NasabahID          string    `json:"nasabah_id" gorm:"column:nasabah_id"`
	NamaNasabah        string    `json:"nama_nasabah" gorm:"column:nama_nasabah"`
	TransaksiTimestamp time.Time `json:"transaksi_timestamp" gorm:"column:transaksi_timestamp"`
	TotalItem          int       `json:"total_item" gorm:"column:total_item"`
}

type HeaderSesiRow struct {
	PenimbanganID     string           `json:"penimbangan_id" gorm:"column:penimbangan_id"`
	BankID            string           `json:"-" gorm:"column:bank_id"`
	NamaBank          string           `json:"nama_bank" gorm:"column:nama_bank"`
	TanggalSesi       time.Time        `json:"tanggal_sesi" gorm:"column:tanggal_sesi"`
	StartedAt         *time.Time       `json:"started_at" gorm:"column:started_at"`
	EndedAt           *time.Time       `json:"ended_at" gorm:"column:ended_at"`
	StartedBy         string           `json:"started_by" gorm:"column:started_by"`
	EndedBy           string           `json:"ended_by" gorm:"column:ended_by"`
	StatusPenimbangan string           `json:"status_penimbangan" gorm:"column:status_penimbangan"`
	AlasanPembatalan  *string          `json:"alasan_pembatalan" gorm:"column:alasan_pembatalan"`
	JamMulai          *string          `json:"jam_mulai" gorm:"column:jam_mulai"`
	JamSelesai        *string          `json:"jam_selesai" gorm:"column:jam_selesai"`
	ListSetoran       []SetoranSesiRow `json:"list_setoran" gorm:"-"`
}

type NotifTarget struct {
	UserID   string `gorm:"column:user_id"`
	FCMToken string `gorm:"column:fcm_token"`
}

type PenimbanganRepo struct {
	db *gorm.DB
}

func NewPenimbanganRepo(db *gorm.DB) *PenimbanganRepo {
	return &PenimbanganRepo{db: db}
}

func (r *PenimbanganRepo) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *PenimbanganRepo) FindPenimbangan(penimbanganID string) (*models.Penimbangan, error) {
	var p models.Penimbangan
	err := r.db.Where("penimbangan_id = ?", penimbanganID).First(&p).Error
	return &p, err
}

func (r *PenimbanganRepo) FindPenimbanganLock(tx *gorm.DB, penimbanganID string, status models.StatusPenimbangan) (*models.Penimbangan, error) {
	var p models.Penimbangan
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("penimbangan_id = ? AND status_penimbangan = ?", penimbanganID, status).
		First(&p).Error
	return &p, err
}

func (r *PenimbanganRepo) SavePenimbangan(tx *gorm.DB, p *models.Penimbangan) error {
	return r.conn(tx).Save(p).Error
}

func (r *PenimbanganRepo) CreatePenimbangan(tx *gorm.DB, p *models.Penimbangan) error {
	return r.conn(tx).Create(p).Error
}

func (r *PenimbanganRepo) CountSesiAktif(tx *gorm.DB, bankID string, tanggal string) (int64, error) {
	var count int64
	err := r.conn(tx).Model(&models.Penimbangan{}).
		Where("bank_id = ? AND tanggal_sesi = ? AND status_penimbangan = ?", bankID, tanggal, models.StatusAktif).
		Count(&count).Error
	return count, err
}

func (r *PenimbanganRepo) CountSesiByJadwalTanggal(jadwalID string, tanggal time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&models.Penimbangan{}).
		Where("jadwal_id = ? AND tanggal_sesi = ?", jadwalID, tanggal).
		Count(&count).Error
	return count, err
}

func (r *PenimbanganRepo) ListSesiTanggal(bankID, tanggal string, statuses []models.StatusPenimbangan) ([]SesiHariIniRow, error) {
	q := r.db.Table("penimbangan").
		Select("penimbangan.penimbangan_id, penimbangan.tanggal_sesi, penimbangan.status_penimbangan, jadwal.jam_mulai, jadwal.jam_selesai, jadwal.nama_jadwal_spesial, jadwal.is_rutin").
		Joins("JOIN jadwal ON jadwal.jadwal_id = penimbangan.jadwal_id").
		Where("penimbangan.bank_id = ? AND penimbangan.tanggal_sesi = ?", bankID, tanggal)

	if len(statuses) > 0 {
		q = q.Where("penimbangan.status_penimbangan IN ?", statuses)
	} else {
		q = q.Order("jadwal.jam_mulai ASC")
	}

	var rows []SesiHariIniRow
	err := q.Find(&rows).Error
	return rows, err
}

func (r *PenimbanganRepo) ListRiwayat(bankID string, start, end *time.Time) ([]RiwayatPenimbanganRow, error) {
	q := r.db.Table("penimbangan").
		Select(`
			penimbangan.penimbangan_id,
			penimbangan.tanggal_sesi,
			penimbangan.status_penimbangan,
			COUNT(setoran_nasabah.setoran_id) AS jumlah_setoran
		`).
		Joins("LEFT JOIN setoran_nasabah ON setoran_nasabah.penimbangan_id = penimbangan.penimbangan_id").
		Where("penimbangan.bank_id = ? AND penimbangan.status_penimbangan IN ?",
			bankID, []models.StatusPenimbangan{models.StatusSelesai, models.StatusDibatalkan}).
		Group("penimbangan.penimbangan_id, penimbangan.tanggal_sesi, penimbangan.status_penimbangan")

	if start != nil {
		q = q.Where("penimbangan.tanggal_sesi >= ?", *start)
	}
	if end != nil {
		q = q.Where("penimbangan.tanggal_sesi <= ?", *end)
	}

	var rows []RiwayatPenimbanganRow
	err := q.Order("penimbangan.tanggal_sesi desc").Find(&rows).Error
	return rows, err
}

func (r *PenimbanganRepo) FindHeaderSesi(penimbanganID string) (*HeaderSesiRow, error) {
	var header HeaderSesiRow
	err := r.db.Table("penimbangan").
		Select(`
			penimbangan.penimbangan_id,
			penimbangan.bank_id,
			bank_sampah.nama_bank,
			penimbangan.tanggal_sesi,
			penimbangan.started_at,
			penimbangan.ended_at,
			penimbangan.status_penimbangan,
			penimbangan.alasan_pembatalan,
			jadwal.jam_mulai,
			jadwal.jam_selesai,
			COALESCE(u_started.nama, '') as started_by,
			CASE WHEN penimbangan.ended_by = 'sistem' THEN 'Sistem' ELSE COALESCE(u_ended.nama, '') END as ended_by
		`).
		Joins("LEFT JOIN bank_sampah ON bank_sampah.bank_id = penimbangan.bank_id").
		Joins("LEFT JOIN jadwal ON jadwal.jadwal_id = penimbangan.jadwal_id").
		Joins("LEFT JOIN users u_started ON u_started.user_id = penimbangan.started_by").
		Joins("LEFT JOIN users u_ended ON u_ended.user_id = penimbangan.ended_by").
		Where("penimbangan.penimbangan_id = ?", penimbanganID).
		First(&header).Error
	return &header, err
}

func (r *PenimbanganRepo) ListSetoranSesi(penimbanganID string) ([]SetoranSesiRow, error) {
	var list []SetoranSesiRow
	err := r.db.Table("setoran_nasabah").
		Select("setoran_nasabah.setoran_id, u_petugas.nama as nama_petugas, nasabah.nasabah_id, u_nasabah.nama as nama_nasabah, setoran_nasabah.created_at as transaksi_timestamp, setoran_nasabah.total_item").
		Joins("LEFT JOIN admin ON admin.admin_id = setoran_nasabah.admin_id").
		Joins("LEFT JOIN users u_petugas ON u_petugas.user_id = admin.user_id").
		Joins("LEFT JOIN nasabah ON nasabah.nasabah_id = setoran_nasabah.nasabah_id").
		Joins("LEFT JOIN users u_nasabah ON u_nasabah.user_id = nasabah.user_id").
		Where("setoran_nasabah.penimbangan_id = ?", penimbanganID).
		Order("setoran_nasabah.created_at desc").
		Find(&list).Error
	return list, err
}

func (r *PenimbanganRepo) TutupSesiTerlewat() (int64, error) {
	result := r.db.Exec(`
		UPDATE penimbangan p
		SET status_penimbangan = (CASE WHEN p.status_penimbangan = 'aktif' THEN 'selesai' ELSE 'dibatalkan' END)::status_penimbangan_enum,
		    ended_by            = 'sistem',
		    ended_at            = NOW(),
		    alasan_pembatalan   = CASE WHEN p.status_penimbangan = 'pending'
		                               THEN 'Sesi tidak dilaksanakan hingga jadwal berakhir'
		                               ELSE p.alasan_pembatalan END
		FROM jadwal j
		WHERE j.jadwal_id              = p.jadwal_id
		  AND p.status_penimbangan     IN ('aktif', 'pending')
		  AND p.tanggal_sesi          <= CURRENT_DATE
		  AND (p.tanggal_sesi + j.jam_selesai + INTERVAL '2 hours') < NOW()
	`)
	return result.RowsAffected, result.Error
}

func (r *PenimbanganRepo) FindBank(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.db.Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *PenimbanganRepo) FindJadwal(jadwalID string) (*models.Jadwal, error) {
	var jadwal models.Jadwal
	err := r.db.Where("jadwal_id = ?", jadwalID).First(&jadwal).Error
	return &jadwal, err
}

func (r *PenimbanganRepo) FindJadwalPenimbangan(jadwalID string) (*models.Jadwal, error) {
	var jadwal models.Jadwal
	err := r.db.Where("jadwal_id = ? AND jenis_jadwal = ?", jadwalID, models.JadwalPenimbangan).
		First(&jadwal).Error
	return &jadwal, err
}

func (r *PenimbanganRepo) ListJadwalJatuhTempo(hari models.HariEnum, mingguKe int, tanggal string) ([]models.Jadwal, error) {
	var jadwals []models.Jadwal
	err := r.db.
		Where(
			"jenis_jadwal = ? AND is_active = true AND ((is_rutin = true AND hari = ? AND minggu_ke = ?) OR (is_rutin = false AND DATE(tanggal) = ?))",
			models.JadwalPenimbangan, hari, mingguKe, tanggal,
		).
		Find(&jadwals).Error
	return jadwals, err
}

func (r *PenimbanganRepo) FindNasabahTargets(bankID string) ([]NotifTarget, error) {
	var targets []NotifTarget
	err := r.db.Table("nasabah").
		Select("users.user_id, users.fcm_token").
		Joins("JOIN users ON users.user_id = nasabah.user_id").
		Where("nasabah.bank_id = ? AND nasabah.status_nasabah = ?", bankID, models.Aktif).
		Scan(&targets).Error
	return targets, err
}

func (r *PenimbanganRepo) FindAdminTargets(bankID string) ([]NotifTarget, error) {
	var targets []NotifTarget
	err := r.db.Table("admin").
		Select("users.user_id, users.fcm_token").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("admin.bank_id = ? AND admin.status_admin = ?", bankID, models.Aktif).
		Scan(&targets).Error
	return targets, err
}
