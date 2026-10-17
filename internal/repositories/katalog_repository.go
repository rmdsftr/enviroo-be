package repositories

import (
	"errors"
	"regexp"
	"time"

	"enviroo-be/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrKatalogMasihDipakai = errors.New("katalog sampah masih dipakai transaksi")

type HistoryHargaRow struct {
	HistoryID     string           `json:"history_id"      gorm:"column:history_sampah_id"`
	SchemaID      string           `json:"schema_id"       gorm:"column:schema_id"`
	LevelUser     models.LevelUser `json:"level_user"      gorm:"column:level_user"`
	HargaLama     float64          `json:"harga_lama"      gorm:"column:harga_lama"`
	HargaBaru     float64          `json:"harga_baru"      gorm:"column:harga_baru"`
	ChangedAt     string           `json:"changed_at"      gorm:"column:changed_at"`
	ChangedByID   string           `json:"changed_by_id"   gorm:"column:changed_by"`
	ChangedByNama string           `json:"changed_by_nama" gorm:"column:changed_by_nama"`
}

type KatalogRepo struct {
	db *gorm.DB
}

func NewKatalogRepo(db *gorm.DB) *KatalogRepo {
	return &KatalogRepo{db: db}
}

func (r *KatalogRepo) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *KatalogRepo) FindBank(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.db.Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *KatalogRepo) ListMasterSampah(q string) ([]models.Sampah, error) {
	db := r.db.Model(&models.Sampah{}).Preload("Satuan")
	if q != "" {
		db = db.Where("nama_sampah ~* ?", q)
	}

	var results []models.Sampah
	err := db.Limit(10).Find(&results).Error
	return results, err
}

func (r *KatalogRepo) FindMasterSampah(sarokID int) (*models.Sampah, error) {
	var sampah models.Sampah
	err := r.db.First(&sampah, "sarok_id = ?", sarokID).Error
	return &sampah, err
}

func (r *KatalogRepo) AdaNilaiRewardAktif(bankID string, rewardID int) (bool, error) {
	var n int64
	err := nilaiRewardAktifSah(r.db, bankID).
		Where("nilai_reward_bank.reward_id = ?", rewardID).
		Count(&n).Error
	return n > 0, err
}

func (r *KatalogRepo) FindKatalog(sampahID string) (*models.KatalogSampah, error) {
	var katalog models.KatalogSampah
	err := r.db.Where("sampah_id = ?", sampahID).First(&katalog).Error
	return &katalog, err
}

func (r *KatalogRepo) FindKatalogWithRelasi(sampahID string) (*models.KatalogSampah, error) {
	var katalog models.KatalogSampah
	err := r.db.
		Preload("Sarok.Satuan").
		Preload("Kategori").
		Preload("Reward").
		Preload("Bank").
		Where("sampah_id = ?", sampahID).
		First(&katalog).Error
	return &katalog, err
}

func (r *KatalogRepo) FindKatalogSejenis(sarokID int, bankID string) (*models.KatalogSampah, error) {
	var katalog models.KatalogSampah
	err := r.db.Preload("Kategori").
		Where("sarok_id = ? AND bank_id = ?", sarokID, bankID).
		Order("created_at ASC").
		First(&katalog).Error
	return &katalog, err
}

func (r *KatalogRepo) ListKatalogByBank(bankID string, limit, offset int) ([]models.KatalogSampah, error) {
	q := r.db.Preload("Sarok.Satuan").Preload("Kategori").Preload("Reward").
		Where("bank_id = ?", bankID)

	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}

	var katalogs []models.KatalogSampah
	err := q.Find(&katalogs).Error
	return katalogs, err
}

func (r *KatalogRepo) CountKatalogByBank(bankID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.KatalogSampah{}).Where("bank_id = ?", bankID).Count(&count).Error
	return count, err
}

func (r *KatalogRepo) CreateKatalog(tx *gorm.DB, katalog *models.KatalogSampah) error {
	return r.conn(tx).Create(katalog).Error
}

func (r *KatalogRepo) SaveKatalog(tx *gorm.DB, katalog *models.KatalogSampah) error {
	return r.conn(tx).Save(katalog).Error
}

func (r *KatalogRepo) UpdateKategoriSejenis(tx *gorm.DB, sarokID int, bankID string, kategoriID int) error {
	return r.conn(tx).Model(&models.KatalogSampah{}).
		Where("sarok_id = ? AND bank_id = ?", sarokID, bankID).
		Update("kategori_id", kategoriID).Error
}

func (r *KatalogRepo) DeleteKatalog(katalog *models.KatalogSampah) error {
	if err := r.db.Delete(katalog).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrKatalogMasihDipakai
		}
		return err
	}
	return nil
}

var tabelPemakaiKatalog = []struct {
	Tabel string
	Label string
}{
	{"detail_setoran_nasabah", "setoran nasabah"},
	{"tabungan_sampah", "tabungan sampah"},
	{"detail_penjualan", "penjualan"},
	{"detail_bagi_hasil", "bagi hasil"},
	{"detail_pengangkutan", "pengangkutan"},
}

func (r *KatalogRepo) CountPemakaianKatalog(sampahID string) (string, error) {
	for _, t := range tabelPemakaiKatalog {
		var count int64
		if err := r.db.Table(t.Tabel).Where("sampah_id = ?", sampahID).Count(&count).Error; err != nil {
			return "", err
		}
		if count > 0 {
			return t.Label, nil
		}
	}
	return "", nil
}

func (r *KatalogRepo) ListStokByBank(bankID string, sampahIDs []string) (map[string]float64, error) {
	hasil := make(map[string]float64, len(sampahIDs))
	if len(sampahIDs) == 0 {
		return hasil, nil
	}

	var rows []models.StokSampah
	if err := r.db.Where("bank_id = ? AND sampah_id IN ?", bankID, sampahIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		hasil[row.SampahID] = row.Stok
	}
	return hasil, nil
}

func (r *KatalogRepo) FindStok(bankID, sampahID string) (float64, error) {
	var stok models.StokSampah
	if err := r.db.Where("bank_id = ? AND sampah_id = ?", bankID, sampahID).Limit(1).Find(&stok).Error; err != nil {
		return 0, err
	}
	return stok.Stok, nil
}

func (r *KatalogRepo) CreateStok(tx *gorm.DB, stok *models.StokSampah) error {
	return r.conn(tx).Create(stok).Error
}

func (r *KatalogRepo) ListSchemaHarga(sampahID string) ([]models.SchemaHargaSampah, error) {
	var schemas []models.SchemaHargaSampah
	err := r.db.Where("sampah_id = ?", sampahID).Find(&schemas).Error
	return schemas, err
}

func (r *KatalogRepo) ListHistoryHarga(sampahID string, start, end *time.Time) ([]HistoryHargaRow, error) {
	q := r.db.Model(&models.KatalogSampahHistory{}).
		Select(`katalog_sampah_history.history_sampah_id,
			katalog_sampah_history.schema_id,
			schema_harga_sampah.level_user,
			katalog_sampah_history.harga_lama,
			katalog_sampah_history.harga_baru,
			katalog_sampah_history.changed_at,
			katalog_sampah_history.changed_by,
			users.nama as changed_by_nama`).
		Joins("JOIN schema_harga_sampah ON schema_harga_sampah.schema_id = katalog_sampah_history.schema_id").
		Joins("JOIN admin ON admin.admin_id = katalog_sampah_history.changed_by").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("schema_harga_sampah.sampah_id = ?", sampahID)

	if start != nil {
		q = q.Where("katalog_sampah_history.changed_at >= ?", *start)
	}
	if end != nil {
		q = q.Where("katalog_sampah_history.changed_at <= ?", *end)
	}

	var histories []HistoryHargaRow
	err := q.Order("katalog_sampah_history.changed_at DESC").Find(&histories).Error
	return histories, err
}

func (r *KatalogRepo) ListKategori() ([]models.KategoriSampah, error) {
	var kategori []models.KategoriSampah
	err := r.db.Find(&kategori).Error
	return kategori, err
}

func (r *KatalogRepo) FindKategori(kategoriID string) (*models.KategoriSampah, error) {
	var kategori models.KategoriSampah
	err := r.db.Where("kategori_id = ?", kategoriID).First(&kategori).Error
	return &kategori, err
}

func (r *KatalogRepo) CountKategoriDenganNama(namaNormal, exceptID string) (int64, error) {
	q := r.db.Model(&models.KategoriSampah{}).
		Where("LOWER(REGEXP_REPLACE(kategori, '[^a-zA-Z0-9]', '', 'g')) = ?", namaNormal)
	if exceptID != "" {
		q = q.Where("kategori_id != ?", exceptID)
	}

	var count int64
	err := q.Count(&count).Error
	return count, err
}

func (r *KatalogRepo) CreateKategori(kategori *models.KategoriSampah) error {
	return r.db.Create(kategori).Error
}

func (r *KatalogRepo) SaveKategori(kategori *models.KategoriSampah) error {
	return r.db.Save(kategori).Error
}

func (r *KatalogRepo) DeleteKategori(kategori *models.KategoriSampah) error {
	return r.db.Delete(kategori).Error
}

func (r *KatalogRepo) CountKatalogByKategori(kategoriID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.KatalogSampah{}).Where("kategori_id = ?", kategoriID).Count(&count).Error
	return count, err
}

func (r *KatalogRepo) ListSatuan() ([]models.Satuan, error) {
	var satuan []models.Satuan
	err := r.db.Find(&satuan).Error
	return satuan, err
}

func (r *KatalogRepo) FindSatuan(satuanID string) (*models.Satuan, error) {
	var satuan models.Satuan
	err := r.db.Where("satuan_id = ?", satuanID).First(&satuan).Error
	return &satuan, err
}

func (r *KatalogRepo) CountSatuanDenganNama(namaNormal, exceptID string) (int64, error) {
	q := r.db.Model(&models.Satuan{}).
		Where("LOWER(REGEXP_REPLACE(satuan, '[^a-zA-Z0-9]', '', 'g')) = ?", namaNormal)
	if exceptID != "" {
		q = q.Where("satuan_id != ?", exceptID)
	}

	var count int64
	err := q.Count(&count).Error
	return count, err
}

func (r *KatalogRepo) CreateSatuan(satuan *models.Satuan) error {
	return r.db.Create(satuan).Error
}

func (r *KatalogRepo) SaveSatuan(satuan *models.Satuan) error {
	return r.db.Save(satuan).Error
}

func (r *KatalogRepo) DeleteSatuan(satuan *models.Satuan) error {
	return r.db.Delete(satuan).Error
}

func (r *KatalogRepo) CountSampahBySatuan(satuanID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Sampah{}).Where("satuan_id = ?", satuanID).Count(&count).Error
	return count, err
}

func (r *KatalogRepo) FindOrCreateMasterSampah(tx *gorm.DB, namaSampah string, satuanID int) (*models.Sampah, error) {
	if namaSampah == "" {
		return nil, errors.New("nama sampah tidak boleh kosong")
	}
	db := r.conn(tx)

	pattern := "^" + regexp.QuoteMeta(namaSampah) + "$"

	var existing models.Sampah
	err := db.Where("nama_sampah ~* ? AND satuan_id = ?", pattern, satuanID).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	baru := models.Sampah{NamaSampah: namaSampah, SatuanID: satuanID}
	if err := db.Create(&baru).Error; err != nil {
		return nil, err
	}
	return &baru, nil
}
