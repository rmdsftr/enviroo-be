package repositories

import (
	"fmt"
	"time"

	"enviroo-be/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Row types ──────────────────────────────────────────────────────────────

type ListJadwalRow struct {
	BsuID             string `json:"bsu_id"              gorm:"column:bsu_id"`
	NamaBsu           string `json:"nama_bsu"            gorm:"column:nama_bsu"`
	JadwalID          string `json:"jadwal_id"           gorm:"column:jadwal_id"`
	NamaJadwalSpesial string `json:"nama_jadwal_spesial" gorm:"column:nama_jadwal_spesial"`
	IsCompleted       bool   `json:"is_completed"        gorm:"column:is_completed"`
}

type PengangkutanListRow struct {
	PengangkutanID     string    `json:"pengangkutan_id"     gorm:"column:pengangkutan_id"`
	BSIID              string    `json:"bsi_id"              gorm:"column:bsi_id"`
	BSUID              string    `json:"bsu_id"              gorm:"column:bsu_id"`
	AdminBSIID         *string   `json:"admin_bsi_id"        gorm:"column:admin_bsi_id"`
	AdminBSUID         *string   `json:"admin_bsu_id"        gorm:"column:admin_bsu_id"`
	JadwalID           string    `json:"jadwal_id"           gorm:"column:jadwal_id"`
	NamaBSI            string    `json:"nama_bsi"            gorm:"column:nama_bsi"`
	NamaBSU            string    `json:"nama_bsu"            gorm:"column:nama_bsu"`
	NamaAdminBSI       *string   `json:"nama_admin_bsi"      gorm:"column:nama_admin_bsi"`
	NamaAdminBSU       *string   `json:"nama_admin_bsu"      gorm:"column:nama_admin_bsu"`
	StatusPengangkutan string    `json:"status_pengangkutan" gorm:"column:status_pengangkutan"`
	ChangedAt          time.Time `json:"changed_at"          gorm:"column:changed_at"`
}

type PengangkutanDetailHeader struct {
	PengangkutanID string  `json:"pengangkutan_id" gorm:"column:pengangkutan_id"`
	NamaBSI        string  `json:"nama_bsi"        gorm:"column:nama_bsi"`
	NamaBSU        string  `json:"nama_bsu"        gorm:"column:nama_bsu"`
	NamaAdminBSI   *string `json:"nama_admin_bsi"  gorm:"column:nama_admin_bsi"`
	NamaAdminBSU   *string `json:"nama_admin_bsu"  gorm:"column:nama_admin_bsu"`
	TotalItem      float64 `json:"total_item"      gorm:"column:total_item"`
	IsMandiri      bool    `json:"is_mandiri"      gorm:"column:is_mandiri"`
	StatusTerkini  string  `json:"status_terkini"  gorm:"column:status_terkini"`
	BuktiFoto      *string `json:"bukti_foto"      gorm:"column:bukti_foto"`
}

type PengangkutanDetailItem struct {
	SampahID   string  `json:"sampah_id"   gorm:"column:sampah_id"`
	NamaSampah string  `json:"nama_sampah" gorm:"column:nama_sampah"`
	Satuan     string  `json:"satuan"      gorm:"column:satuan"`
	Qty        float64 `json:"qty"         gorm:"column:qty"`
}

type SampahPengangkutanRow struct {
	SampahID   string  `json:"sampah_id"   gorm:"column:sampah_id"`
	NamaSampah string  `json:"nama_sampah" gorm:"column:nama_sampah"`
	FotoSampah string  `json:"foto_sampah" gorm:"column:foto_sampah"`
	Satuan     string  `json:"satuan"      gorm:"column:satuan"`
	NamaReward string  `json:"nama_reward" gorm:"column:nama_reward"`
	Stok       float64 `json:"stok"        gorm:"column:stok"`
}

type RiwayatDetailRow struct {
	Status      models.StatusPengangkutan `json:"status"     gorm:"column:status_pengangkutan"`
	ChangedAt   time.Time                 `json:"changed_at" gorm:"column:changed_at"`
	ChangedBy   string                    `json:"changed_by" gorm:"column:changed_by"`
	Notes       string                    `json:"notes"      gorm:"column:notes"`
	NamaPetugas string                    `json:"-"          gorm:"column:nama"`
}

type PengangkutanAdminUser struct {
	UserID   string `gorm:"column:user_id"`
	FCMToken string `gorm:"column:fcm_token"`
}

// ── Repo ───────────────────────────────────────────────────────────────────

type PengangkutanRepo struct {
	db *gorm.DB
}

func NewPengangkutanRepo(db *gorm.DB) *PengangkutanRepo {
	return &PengangkutanRepo{db: db}
}

// ── Non-tx reads ───────────────────────────────────────────────────────────

func (r *PengangkutanRepo) GetJadwalHariIni(bsiID string, todayHari models.HariEnum, mingguKe int, todayDate string) ([]ListJadwalRow, error) {
	var rows []ListJadwalRow
	err := r.db.Table("jadwal j").
		Select(`j.jadwal_id, j.target_bank_id AS bsu_id, j.nama_jadwal_spesial,
			bs.nama_bank AS nama_bsu,
			CASE WHEN rp.status_pengangkutan = 'completed' THEN true ELSE false END AS is_completed`).
		Joins("LEFT JOIN bank_sampah bs ON bs.bank_id = j.target_bank_id").
		Joins("LEFT JOIN pengangkutan_sampah ps ON ps.jadwal_id = j.jadwal_id").
		Joins(`LEFT JOIN riwayat_pengangkutan rp ON rp.riwayat_pengangkutan_id = (
			SELECT MAX(riwayat_pengangkutan_id)
			FROM riwayat_pengangkutan
			WHERE pengangkutan_id = ps.pengangkutan_id
		)`).
		Where("j.bank_id = ? AND j.jenis_jadwal = ? AND j.is_active = ? AND ("+
			"(j.is_rutin = true AND j.hari = ? AND (j.minggu_ke = ? OR j.minggu_ke = 0)) OR "+
			"(j.is_rutin = false AND j.tanggal = ?))",
			bsiID, models.JadwalPengangkutan, true, todayHari, mingguKe, todayDate).
		Find(&rows).Error
	return rows, err
}

func (r *PengangkutanRepo) FindBank(bankID string) (*models.BankSampah, error) {
	var b models.BankSampah
	err := r.db.Where("bank_id = ?", bankID).First(&b).Error
	return &b, err
}

func (r *PengangkutanRepo) FindBankAktif(bankID string) (*models.BankSampah, error) {
	var b models.BankSampah
	err := r.db.Where("bank_id = ? AND is_active = ?", bankID, true).First(&b).Error
	return &b, err
}

func (r *PengangkutanRepo) FindBSUAktifByParent(bsuID, bsiID string) (*models.BankSampah, error) {
	var b models.BankSampah
	err := r.db.Where("bank_id = ? AND parent_bank_id = ? AND is_active = ? AND jenis_bank = ?",
		bsuID, bsiID, true, models.BSU).First(&b).Error
	return &b, err
}

func (r *PengangkutanRepo) FindJadwalStart(bsiID, bsuID string, todayHari models.HariEnum, mingguKe int, todayDate string) (*models.Jadwal, error) {
	var j models.Jadwal
	err := r.db.Where("bank_id = ? AND target_bank_id = ? AND jenis_jadwal = ? AND ("+
		"(is_rutin = true AND hari = ? AND (minggu_ke = ? OR minggu_ke = 0)) OR "+
		"(is_rutin = false AND tanggal = ?))",
		bsiID, bsuID, models.JadwalPengangkutan, todayHari, mingguKe, todayDate).First(&j).Error
	return &j, err
}

func (r *PengangkutanRepo) GetListByBSI(bsiID, startDate, endDate string) ([]PengangkutanListRow, error) {
	return r.applyDateFilter(r.listBaseQuery().Where("ps.bsi_id = ?", bsiID), startDate, endDate)
}

func (r *PengangkutanRepo) GetListByBSU(bsuID, startDate, endDate string) ([]PengangkutanListRow, error) {
	return r.applyDateFilter(r.listBaseQuery().Where("ps.bsu_id = ?", bsuID), startDate, endDate)
}

func (r *PengangkutanRepo) FindPengangkutan(pengangkutanID string) (*models.PengangkutanSampah, error) {
	var p models.PengangkutanSampah
	err := r.db.Where("pengangkutan_id = ?", pengangkutanID).First(&p).Error
	return &p, err
}

func (r *PengangkutanRepo) FindAdminByBank(adminID, bankID string) (*models.Admin, error) {
	var a models.Admin
	err := r.db.Where("admin_id = ? AND bank_id = ?", adminID, bankID).First(&a).Error
	return &a, err
}

// FindLastRiwayat menerima *gorm.DB agar bisa dipakai di dalam maupun luar transaksi.
func (r *PengangkutanRepo) FindLastRiwayat(db *gorm.DB, pengangkutanID string) (*models.RiwayatPengangkutan, error) {
	var rw models.RiwayatPengangkutan
	err := db.Where("pengangkutan_id = ?", pengangkutanID).Order("changed_at DESC").First(&rw).Error
	return &rw, err
}

func (r *PengangkutanRepo) SetAdminBSI(db *gorm.DB, pgk *models.PengangkutanSampah, adminID string) error {
	return db.Model(pgk).Update("admin_bsi_id", adminID).Error
}

// CheckJadwalConflict mengembalikan true jika ada jadwal yang bertabrakan.
func (r *PengangkutanRepo) CheckJadwalConflict(bsiID, bsuID string, reqHari models.HariEnum, mingguKe int, reqDateStr, jamMulai, jamSelesai string) (bool, error) {
	var j models.Jadwal
	err := r.db.Where("bank_id = ? AND target_bank_id = ? AND jenis_jadwal = ? AND is_active = ? AND ("+
		"(is_rutin = true AND hari = ? AND (minggu_ke = ? OR minggu_ke = 0)) OR "+
		"(is_rutin = false AND tanggal = ?)"+
		") AND jam_mulai < ? AND jam_selesai > ?",
		bsiID, bsuID, models.JadwalPengangkutan, true, reqHari, mingguKe, reqDateStr, jamSelesai, jamMulai).First(&j).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	return err == nil, err
}

func (r *PengangkutanRepo) FindLatestPengangkutan(bsuID, bsiID string) (*models.PengangkutanSampah, error) {
	var p models.PengangkutanSampah
	err := r.db.Where("bsu_id = ? AND bsi_id = ?", bsuID, bsiID).Order("pengangkutan_id DESC").First(&p).Error
	return &p, err
}

func (r *PengangkutanRepo) FindBSUList(bsiID string) ([]models.BankSampah, error) {
	var list []models.BankSampah
	err := r.db.Where("parent_bank_id = ? AND jenis_bank = ?", bsiID, models.BSU).Find(&list).Error
	return list, err
}

func (r *PengangkutanRepo) FindJadwal(jadwalID uuid.UUID) (*models.Jadwal, error) {
	var j models.Jadwal
	err := r.db.Where("jadwal_id = ?", jadwalID).First(&j).Error
	return &j, err
}

func (r *PengangkutanRepo) GetDetailHeader(pengangkutanID string) (*PengangkutanDetailHeader, error) {
	var h PengangkutanDetailHeader
	err := r.db.Table("pengangkutan_sampah ps").
		Select(`ps.pengangkutan_id, ps.total_item, ps.is_mandiri, ps.bukti_foto,
			bsi.nama_bank AS nama_bsi, bsu.nama_bank AS nama_bsu,
			u_bsi.nama AS nama_admin_bsi, u_bsu.nama AS nama_admin_bsu,
			rp.status_pengangkutan AS status_terkini`).
		Joins("LEFT JOIN bank_sampah bsi ON bsi.bank_id = ps.bsi_id").
		Joins("LEFT JOIN bank_sampah bsu ON bsu.bank_id = ps.bsu_id").
		Joins("LEFT JOIN admin a_bsi ON a_bsi.admin_id = ps.admin_bsi_id").
		Joins("LEFT JOIN users u_bsi ON u_bsi.user_id = a_bsi.user_id").
		Joins("LEFT JOIN admin a_bsu ON a_bsu.admin_id = ps.admin_bsu_id").
		Joins("LEFT JOIN users u_bsu ON u_bsu.user_id = a_bsu.user_id").
		Joins(`LEFT JOIN riwayat_pengangkutan rp ON rp.riwayat_pengangkutan_id = (
			SELECT MAX(riwayat_pengangkutan_id)
			FROM riwayat_pengangkutan
			WHERE pengangkutan_id = ps.pengangkutan_id
		)`).
		Where("ps.pengangkutan_id = ?", pengangkutanID).
		First(&h).Error
	return &h, err
}

func (r *PengangkutanRepo) GetDetailItems(pengangkutanID string) ([]PengangkutanDetailItem, error) {
	var items []PengangkutanDetailItem
	err := r.db.Table("detail_pengangkutan dp").
		Select("dp.sampah_id, s.nama_sampah, s.satuan, dp.qty").
		Joins("LEFT JOIN katalog_sampah ks ON ks.sampah_id = dp.sampah_id").
		Joins("LEFT JOIN sampah s ON s.sarok_id = ks.sarok_id").
		Where("dp.pengangkutan_id = ?", pengangkutanID).
		Find(&items).Error
	return items, err
}

func (r *PengangkutanRepo) GetSampahList(bsiID, bsuID string) ([]SampahPengangkutanRow, error) {
	var rows []SampahPengangkutanRow
	err := r.db.Table("katalog_sampah ks").
		Select("ks.sampah_id, sampah.nama_sampah, ks.photo_url AS foto_sampah, sampah.satuan, r.nama_reward, COALESCE(ss.stok, 0) AS stok").
		Joins("LEFT JOIN sampah ON sampah.sarok_id = ks.sarok_id").
		Joins("LEFT JOIN stok_sampah ss ON ss.sampah_id = ks.sampah_id AND ss.bank_id = ?", bsuID).
		Joins("LEFT JOIN reward r ON r.reward_id = ks.reward_id").
		Where("ks.bank_id = ?", bsiID).
		Find(&rows).Error
	return rows, err
}

func (r *PengangkutanRepo) GetRiwayatDetail(pengangkutanID string) ([]RiwayatDetailRow, error) {
	var rows []RiwayatDetailRow
	err := r.db.Table("riwayat_pengangkutan").
		Select("riwayat_pengangkutan.status_pengangkutan, riwayat_pengangkutan.changed_at, riwayat_pengangkutan.changed_by, riwayat_pengangkutan.notes, users.nama").
		Joins("LEFT JOIN admin ON admin.admin_id = riwayat_pengangkutan.changed_by").
		Joins("LEFT JOIN users ON users.user_id = admin.user_id").
		Where("riwayat_pengangkutan.pengangkutan_id = ?", pengangkutanID).
		Order("riwayat_pengangkutan.changed_at DESC").
		Find(&rows).Error
	return rows, err
}

// FindAdminsByBank mengambil semua admin suatu bank (tanpa filter status).
func (r *PengangkutanRepo) FindAdminsByBank(bankID string) ([]PengangkutanAdminUser, error) {
	var users []PengangkutanAdminUser
	err := r.db.Table("admin").
		Select("users.user_id, users.fcm_token").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("admin.bank_id = ?", bankID).
		Scan(&users).Error
	return users, err
}

// FindAdminsBankAktif mengambil admin aktif suatu bank.
func (r *PengangkutanRepo) FindAdminsBankAktif(bankID string) ([]PengangkutanAdminUser, error) {
	var users []PengangkutanAdminUser
	err := r.db.Table("admin").
		Select("users.user_id, users.fcm_token").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("admin.bank_id = ? AND admin.status_admin = ?", bankID, models.Aktif).
		Scan(&users).Error
	return users, err
}

func (r *PengangkutanRepo) FindStok(bankID, sampahID string) (float64, error) {
	var stok models.StokSampah
	err := r.db.Where("bank_id = ? AND sampah_id = ?", bankID, sampahID).First(&stok).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	return stok.Stok, err
}

func (r *PengangkutanRepo) FindKatalogWithAll(sampahID string) (*models.KatalogSampah, error) {
	var k models.KatalogSampah
	err := r.db.Preload("Sarok").Preload("Reward").Where("sampah_id = ?", sampahID).First(&k).Error
	return &k, err
}

// ── Tx methods (caller menyediakan *gorm.DB) ──────────────────────────────

func (r *PengangkutanRepo) CreatePengangkutan(tx *gorm.DB, p *models.PengangkutanSampah) error {
	return tx.Create(p).Error
}

func (r *PengangkutanRepo) CreateRiwayat(tx *gorm.DB, rw *models.RiwayatPengangkutan) error {
	return tx.Create(rw).Error
}

func (r *PengangkutanRepo) CreateJadwal(tx *gorm.DB, j *models.Jadwal) error {
	return tx.Create(j).Error
}

func (r *PengangkutanRepo) UpdatePengangkutanMeta(tx *gorm.DB, pengangkutanID string, updates map[string]interface{}) error {
	return tx.Model(&models.PengangkutanSampah{}).Where("pengangkutan_id = ?", pengangkutanID).Updates(updates).Error
}

func (r *PengangkutanRepo) CreateDetailPengangkutan(tx *gorm.DB, d *models.DetailPengangkutan) error {
	return tx.Create(d).Error
}

// DecrStokBSU mengurangi stok BSU; mengembalikan error jika stok tidak cukup.
func (r *PengangkutanRepo) DecrStokBSU(tx *gorm.DB, bankID, sampahID string, qty float64) error {
	var stok models.StokSampah
	if err := tx.Where("bank_id = ? AND sampah_id = ?", bankID, sampahID).First(&stok).Error; err != nil {
		return fmt.Errorf("stok BSU untuk sampah %s tidak ditemukan: %w", sampahID, err)
	}
	if stok.Stok < qty {
		return fmt.Errorf("stok BSU tidak mencukupi untuk sampah %s (stok: %g, diminta: %g)", sampahID, stok.Stok, qty)
	}
	return tx.Model(&stok).Update("stok", gorm.Expr("stok - ?", qty)).Error
}

// UpsertStokBSI menambah stok BSI; membuat baris baru jika belum ada.
func (r *PengangkutanRepo) UpsertStokBSI(tx *gorm.DB, bankID, sampahID string, qty float64) error {
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

func (r *PengangkutanRepo) FindKatalogTx(tx *gorm.DB, sampahID string) (*models.KatalogSampah, error) {
	var k models.KatalogSampah
	err := tx.Preload("Reward").Where("sampah_id = ?", sampahID).First(&k).Error
	return &k, err
}

func (r *PengangkutanRepo) CreateTabunganSampah(tx *gorm.DB, t *models.TabunganSampah) error {
	return tx.Create(t).Error
}

// ── Private query helpers ──────────────────────────────────────────────────

func (r *PengangkutanRepo) listBaseQuery() *gorm.DB {
	return r.db.Table("pengangkutan_sampah ps").
		Select(`ps.pengangkutan_id, ps.bsi_id, ps.bsu_id, ps.admin_bsi_id, ps.admin_bsu_id, ps.jadwal_id,
			bsi.nama_bank AS nama_bsi, bsu.nama_bank AS nama_bsu,
			u_bsi.nama AS nama_admin_bsi, u_bsu.nama AS nama_admin_bsu,
			rp.status_pengangkutan, rp.changed_at`).
		Joins("LEFT JOIN bank_sampah bsi ON bsi.bank_id = ps.bsi_id").
		Joins("LEFT JOIN bank_sampah bsu ON bsu.bank_id = ps.bsu_id").
		Joins("LEFT JOIN admin a_bsi ON a_bsi.admin_id = ps.admin_bsi_id").
		Joins("LEFT JOIN users u_bsi ON u_bsi.user_id = a_bsi.user_id").
		Joins("LEFT JOIN admin a_bsu ON a_bsu.admin_id = ps.admin_bsu_id").
		Joins("LEFT JOIN users u_bsu ON u_bsu.user_id = a_bsu.user_id").
		Joins(`LEFT JOIN riwayat_pengangkutan rp ON rp.riwayat_pengangkutan_id = (
			SELECT MAX(riwayat_pengangkutan_id)
			FROM riwayat_pengangkutan
			WHERE pengangkutan_id = ps.pengangkutan_id
		)`).
		Order("rp.changed_at DESC")
}

func (r *PengangkutanRepo) applyDateFilter(q *gorm.DB, startDate, endDate string) ([]PengangkutanListRow, error) {
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			q = q.Where("rp.changed_at >= ?", t)
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			q = q.Where("rp.changed_at <= ?", t.Add(24*time.Hour-time.Second))
		}
	}
	var rows []PengangkutanListRow
	return rows, q.Find(&rows).Error
}
