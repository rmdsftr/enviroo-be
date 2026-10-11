package repositories

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BarangBankRow struct {
	ProdukID   string  `json:"produk_id" gorm:"column:produk_id"`
	NamaBarang string  `json:"nama_barang" gorm:"column:nama_barang"`
	BarangID   int     `json:"barang_id" gorm:"column:barang_id"`
	PhotoURL   *string `json:"photo_url" gorm:"column:photo_url"`
	NilaiPoin  float64 `json:"nilai_poin" gorm:"column:nilai_poin"`
	Stok       float64 `json:"stok" gorm:"column:stok"`
	TotalCount int     `json:"-" gorm:"column:total_count"`
}

type RiwayatDistribusiRow struct {
	DisbaID      string  `json:"disba_id"      gorm:"column:disba_id"`
	TanggalKirim string  `json:"tanggal_kirim" gorm:"column:tanggal_kirim"`
	Item         float64 `json:"item"          gorm:"column:item"`
	StokSebelum  float64 `json:"stok_sebelum"  gorm:"column:stok_sebelum"`
	StokSesudah  float64 `json:"stok_sesudah"  gorm:"column:stok_sesudah"`
}

type DistribusiHeaderRow struct {
	DisbaID          string    `gorm:"column:disba_id"          json:"disba_id"`
	BsiID            string    `gorm:"column:bsi_id"            json:"bsi_id"`
	NamaBSI          string    `gorm:"column:nama_bsi"          json:"nama_bsi"`
	BsuID            string    `gorm:"column:bsu_id"            json:"bsu_id"`
	NamaBSU          string    `gorm:"column:nama_bsu"          json:"nama_bsu"`
	NamaAdminBSI     string    `gorm:"column:nama_admin_bsi"    json:"nama_admin_bsi"`
	NamaAdminBSU     string    `gorm:"column:nama_admin_bsu"    json:"nama_admin_bsu"`
	CreatedAt        time.Time `gorm:"column:created_at"        json:"created_at"`
	TotalItem        float64   `gorm:"column:total_item"        json:"total_item"`
	TotalPoin        float64   `gorm:"column:total_poin"        json:"total_poin"`
	StatusDistribusi string    `gorm:"column:status_distribusi" json:"status_distribusi"`
}

type DistribusiItemRow struct {
	ProdukID       string  `gorm:"column:produk_id"       json:"produk_id"`
	NamaBarang     string  `gorm:"column:nama_barang"      json:"nama_barang"`
	PhotoURL       *string `gorm:"column:photo_url"        json:"photo_url"`
	NilaiPoin      float64 `gorm:"column:nilai_poin"       json:"nilai_poin"`
	Item           float64 `gorm:"column:item"             json:"item"`
	SubtotalPoin   float64 `gorm:"column:subtotal_poin"    json:"subtotal_poin"`
	StokBsiSebelum float64 `gorm:"column:stok_bsi_sebelum" json:"stok_bsi_sebelum"`
	StokBsiSesudah float64 `gorm:"column:stok_bsi_sesudah" json:"stok_bsi_sesudah"`
	StokBsuSebelum float64 `gorm:"column:stok_bsu_sebelum" json:"stok_bsu_sebelum"`
	StokBsuSesudah float64 `gorm:"column:stok_bsu_sesudah" json:"stok_bsu_sesudah"`
}

const baseSelectDistribusi = `
	SELECT
		d.disba_id,
		d.bsi_id,
		bsi.nama_bank                 AS nama_bsi,
		d.bsu_id,
		bsu.nama_bank                 AS nama_bsu,
		COALESCE(u_bsi.nama, '-')     AS nama_admin_bsi,
		COALESCE(u_bsu.nama, '-')     AS nama_admin_bsu,
		d.created_at,
		d.total_item,
		d.total_poin,
		d.status_distribusi
	FROM distribusi_barang_bsu d
	LEFT JOIN bank_sampah bsi  ON bsi.bank_id  = d.bsi_id
	LEFT JOIN bank_sampah bsu  ON bsu.bank_id  = d.bsu_id
	LEFT JOIN admin a_bsi      ON a_bsi.admin_id = d.admin_bsi_id
	LEFT JOIN users u_bsi      ON u_bsi.user_id  = a_bsi.user_id
	LEFT JOIN admin a_bsu      ON a_bsu.admin_id = d.admin_bsu_id
	LEFT JOIN users u_bsu      ON u_bsu.user_id  = a_bsu.user_id
	%s`

type BarangRepo struct {
	db *gorm.DB
}

func NewBarangRepo(db *gorm.DB) *BarangRepo {
	return &BarangRepo{db: db}
}

func (r *BarangRepo) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *BarangRepo) FindBank(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.db.Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *BarangRepo) ListMasterBarang(q string) ([]models.Barang, error) {
	db := r.db.Model(&models.Barang{})
	if q != "" {
		db = db.Where("nama_barang ~* ?", q)
	}

	var results []models.Barang
	err := db.Limit(10).Find(&results).Error
	return results, err
}

func (r *BarangRepo) FindMasterBarang(barangID int) (*models.Barang, error) {
	var barang models.Barang
	err := r.db.First(&barang, "barang_id = ?", barangID).Error
	return &barang, err
}

func (r *BarangRepo) AdaRewardBarangAktif(bankID string) (bool, error) {
	var n int64
	err := nilaiRewardAktifSah(r.db, bankID).
		Where("r.nama_reward = ?", models.RewardEnumBarang).
		Count(&n).Error
	return n > 0, err
}

func (r *BarangRepo) FindKatalogBarang(produkID string) (*models.KatalogBarang, error) {
	var katalog models.KatalogBarang
	err := r.db.Where("produk_id = ?", produkID).First(&katalog).Error
	return &katalog, err
}

func (r *BarangRepo) FindKatalogBarangByBank(tx *gorm.DB, produkID, bankID string) (*models.KatalogBarang, error) {
	var katalog models.KatalogBarang
	err := r.conn(tx).Where("produk_id = ? AND bank_id = ?", produkID, bankID).First(&katalog).Error
	return &katalog, err
}

func (r *BarangRepo) FindKatalogBarangWithMaster(produkID, bankID string) (*models.KatalogBarang, error) {
	var katalog models.KatalogBarang
	err := r.db.Preload("MasterBarang").
		Where("produk_id = ? AND bank_id = ?", produkID, bankID).
		First(&katalog).Error
	return &katalog, err
}

func (r *BarangRepo) CreateKatalogBarang(tx *gorm.DB, katalog *models.KatalogBarang) error {
	return r.conn(tx).Create(katalog).Error
}

func (r *BarangRepo) UpdateKatalogBarang(tx *gorm.DB, produkID string, fields map[string]any) error {
	return r.conn(tx).Model(&models.KatalogBarang{}).
		Where("produk_id = ?", produkID).
		Updates(fields).Error
}

func (r *BarangRepo) DeleteKatalogBarang(tx *gorm.DB, produkID string) error {
	result := r.conn(tx).Where("produk_id = ?", produkID).Delete(&models.KatalogBarang{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *BarangRepo) ListBarangBank(bankID, katalogBankID string, limit, offset int) ([]BarangBankRow, error) {
	const baseSQL = `
		SELECT
			ks.produk_id,
			s.nama_barang,
			s.barang_id,
			ks.photo_url,
			ks.nilai_poin,
			COALESCE(st.stok, 0) AS stok
			%s
		FROM katalog_barang ks
		JOIN barang s ON s.barang_id = ks.barang_id
		LEFT JOIN stok_barang st
			ON st.produk_id = ks.produk_id
			AND st.bank_id = ?
		WHERE ks.bank_id = ?
		ORDER BY s.nama_barang ASC
		%s`

	var rows []BarangBankRow
	var err error

	if limit <= 0 {
		q := fmt.Sprintf(baseSQL, "", "")
		err = r.db.Raw(q, bankID, katalogBankID).Scan(&rows).Error
	} else {
		q := fmt.Sprintf(baseSQL, ", COUNT(*) OVER() AS total_count", "LIMIT ? OFFSET ?")
		err = r.db.Raw(q, bankID, katalogBankID, limit, offset).Scan(&rows).Error
	}

	return rows, err
}

func (r *BarangRepo) CountPemakaianBarang(produkID string) (int64, error) {
	var count int64
	err := r.db.Raw(`
		SELECT COUNT(*) FROM (
			SELECT produk_id FROM detail_distribusi_barang WHERE produk_id = ?
			UNION ALL
			SELECT produk_id FROM detail_penarikan_barang WHERE produk_id = ?
		) t`, produkID, produkID).Scan(&count).Error
	return count, err
}

func (r *BarangRepo) FindStok(tx *gorm.DB, produkID, bankID string) (*models.StokBarang, error) {
	var stok models.StokBarang
	err := r.conn(tx).Where("produk_id = ? AND bank_id = ?", produkID, bankID).First(&stok).Error
	return &stok, err
}

func (r *BarangRepo) LockStok(tx *gorm.DB, produkID, bankID string) (*models.StokBarang, error) {
	var stok models.StokBarang
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("produk_id = ? AND bank_id = ?", produkID, bankID).
		First(&stok).Error
	return &stok, err
}

func (r *BarangRepo) CreateStok(tx *gorm.DB, stok *models.StokBarang) error {
	return r.conn(tx).Create(stok).Error
}

func (r *BarangRepo) SetStok(tx *gorm.DB, produkID, bankID string, nilai float64) error {
	return r.conn(tx).Model(&models.StokBarang{}).
		Where("produk_id = ? AND bank_id = ?", produkID, bankID).
		UpdateColumn("stok", nilai).Error
}

func (r *BarangRepo) TambahStok(tx *gorm.DB, produkID, bankID string, delta float64) error {
	return r.conn(tx).Model(&models.StokBarang{}).
		Where("produk_id = ? AND bank_id = ?", produkID, bankID).
		UpdateColumn("stok", gorm.Expr("stok + ?", delta)).Error
}

func (r *BarangRepo) DeleteStokByProduk(tx *gorm.DB, produkID string) error {
	return r.conn(tx).Where("produk_id = ?", produkID).Delete(&models.StokBarang{}).Error
}

func (r *BarangRepo) FindDistribusi(disbaID string) (*models.DistribusiBarangBsu, error) {
	var disba models.DistribusiBarangBsu
	err := r.db.Where("disba_id = ?", disbaID).First(&disba).Error
	return &disba, err
}

func (r *BarangRepo) CreateDistribusi(tx *gorm.DB, disba *models.DistribusiBarangBsu) error {
	return r.conn(tx).Create(disba).Error
}

func (r *BarangRepo) UpdateDistribusi(tx *gorm.DB, disbaID string, fields map[string]any) error {
	return r.conn(tx).Model(&models.DistribusiBarangBsu{}).
		Where("disba_id = ?", disbaID).
		Updates(fields).Error
}

func (r *BarangRepo) CreateDetailDistribusi(tx *gorm.DB, detail *models.DetailDistribusiBarang) error {
	return r.conn(tx).Create(detail).Error
}

func (r *BarangRepo) ListDetailDistribusi(disbaID string) ([]models.DetailDistribusiBarang, error) {
	var details []models.DetailDistribusiBarang
	err := r.db.Where("disba_id = ?", disbaID).Find(&details).Error
	return details, err
}

func (r *BarangRepo) UpdateStokDetail(tx *gorm.DB, disbaID, produkID string, fields map[string]any) error {
	return r.conn(tx).Model(&models.DetailDistribusiBarang{}).
		Where("disba_id = ? AND produk_id = ?", disbaID, produkID).
		Updates(fields).Error
}

func (r *BarangRepo) ListHeaderDistribusi(filterCol, bankID string, start, end *time.Time) ([]DistribusiHeaderRow, error) {
	where := fmt.Sprintf("WHERE %s = ?", filterCol)
	args := []interface{}{bankID}

	if start != nil {
		where += " AND d.created_at >= ?"
		args = append(args, *start)
	}
	if end != nil {
		where += " AND d.created_at <= ?"
		args = append(args, *end)
	}

	query := fmt.Sprintf(baseSelectDistribusi, where+"\n\tORDER BY d.created_at DESC")

	var rows []DistribusiHeaderRow
	err := r.db.Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (r *BarangRepo) FindHeaderDistribusi(disbaID string) (*DistribusiHeaderRow, error) {
	query := fmt.Sprintf(baseSelectDistribusi, "WHERE d.disba_id = ?")

	var hdr DistribusiHeaderRow
	err := r.db.Raw(query, disbaID).Scan(&hdr).Error
	return &hdr, err
}

func (r *BarangRepo) ListItemDistribusi(disbaID string) ([]DistribusiItemRow, error) {
	var items []DistribusiItemRow
	err := r.db.Raw(`
		SELECT
			dd.produk_id,
			sm.nama_barang,
			ks.photo_url,
			ks.nilai_poin,
			dd.item,
			dd.subtotal_poin,
			dd.stok_bsi_sebelum,
			dd.stok_bsi_sesudah,
			dd.stok_bsu_sebelum,
			dd.stok_bsu_sesudah
		FROM detail_distribusi_barang dd
		JOIN katalog_barang ks ON ks.produk_id = dd.produk_id
		JOIN barang sm         ON sm.barang_id  = ks.barang_id
		WHERE dd.disba_id = ?
		ORDER BY sm.nama_barang ASC
	`, disbaID).Scan(&items).Error
	return items, err
}

func (r *BarangRepo) ListRiwayatDistribusiProduk(produkID, bankID string, sebagaiBSU bool) ([]RiwayatDistribusiRow, error) {
	stokSebelumCol, stokSesudahCol, filterCol := "d.stok_bsi_sebelum", "d.stok_bsi_sesudah", "dsb.bsi_id"
	if sebagaiBSU {
		stokSebelumCol, stokSesudahCol, filterCol = "d.stok_bsu_sebelum", "d.stok_bsu_sesudah", "dsb.bsu_id"
	}

	query := fmt.Sprintf(`
		SELECT
			d.disba_id,
			TO_CHAR(dsb.created_at, 'YYYY-MM-DD HH24:MI:SS') AS tanggal_kirim,
			d.item,
			%s AS stok_sebelum,
			%s AS stok_sesudah
		FROM detail_distribusi_barang d
		JOIN distribusi_barang_bsu dsb
			ON dsb.disba_id = d.disba_id
			AND %s = ?
		WHERE d.produk_id = ?
		ORDER BY dsb.created_at DESC
	`, stokSebelumCol, stokSesudahCol, filterCol)

	var rows []RiwayatDistribusiRow
	err := r.db.Raw(query, bankID, produkID).Scan(&rows).Error
	return rows, err
}

func (r *BarangRepo) FindAdminTargets(bankID string) ([]NotifTarget, error) {
	var targets []NotifTarget
	err := r.db.Table("admin").
		Select("users.user_id, users.fcm_token").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("admin.bank_id = ? AND admin.status_admin = ?", bankID, models.Aktif).
		Scan(&targets).Error
	return targets, err
}

func (r *BarangRepo) FindOrCreateMasterBarang(tx *gorm.DB, namaBarang string) (*models.Barang, error) {
	if namaBarang == "" {
		return nil, errors.New("nama barang tidak boleh kosong")
	}
	db := r.conn(tx)

	pattern := "^" + regexp.QuoteMeta(namaBarang) + "$"

	var existing models.Barang
	err := db.Where("nama_barang ~* ?", pattern).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	baru := models.Barang{NamaBarang: namaBarang}
	if err := db.Create(&baru).Error; err != nil {
		return nil, err
	}
	return &baru, nil
}
