package repositories

import (
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
)

type PencairanTabunganRow struct {
	TabunganID  string    `gorm:"column:tabungan_id"`
	TanggalCair time.Time `gorm:"column:tanggal_cair"`
	QtyDipakai  float64   `gorm:"column:qty_dipakai"`
	HargaItem   float64   `gorm:"column:harga_item"`
}

type TabunganRepo struct {
	db *gorm.DB
}

func NewTabunganRepo(db *gorm.DB) *TabunganRepo {
	return &TabunganRepo{db: db}
}

func (r *TabunganRepo) listTabungan(kolomPemilik, pemilikID string, entitas models.EntitasEnum, start, end *time.Time) ([]models.TabunganSampah, error) {
	q := r.db.
		Where(kolomPemilik+" = ? AND entitas = ?", pemilikID, entitas).
		Preload("KatalogSampah.Reward").
		Preload("KatalogSampah.Sarok")

	if start != nil {
		q = q.Where("created_at >= ?", *start)
	}
	if end != nil {
		q = q.Where("created_at <= ?", *end)
	}

	var tabungans []models.TabunganSampah
	err := q.Order("created_at ASC").Find(&tabungans).Error
	return tabungans, err
}

func (r *TabunganRepo) ListTabunganNasabah(nasabahID string, start, end *time.Time) ([]models.TabunganSampah, error) {
	return r.listTabungan("nasabah_id", nasabahID, models.EntitasNasabah, start, end)
}

func (r *TabunganRepo) ListTabunganBank(bankID string, start, end *time.Time) ([]models.TabunganSampah, error) {
	return r.listTabungan("bank_id", bankID, models.EntitasBankSampah, start, end)
}

func (r *TabunganRepo) ListPencairanTabungan(tabunganIDs []string) ([]PencairanTabunganRow, error) {
	if len(tabunganIDs) == 0 {
		return nil, nil
	}

	var rows []PencairanTabunganRow
	err := r.db.Raw(`
		SELECT
			pt.tabungan_id,
			pbh.created_at              AS tanggal_cair,
			COALESCE(pt.qty_dipakai, 0) AS qty_dipakai,
			dbh.harga_item
		FROM pencairan_tabungan pt
		JOIN penerima_bagi_hasil pbh ON pbh.penerima_id = pt.penerima_id
		JOIN detail_bagi_hasil   dbh ON dbh.penerima_id = pbh.penerima_id
		JOIN tabungan_sampah     ts  ON ts.tabungan_id  = pt.tabungan_id
		                            AND ts.sampah_id    = dbh.sampah_id
		WHERE pt.tabungan_id IN ?
		ORDER BY pt.tabungan_id, pbh.created_at ASC, pt.penerima_id ASC
	`, tabunganIDs).Scan(&rows).Error
	return rows, err
}

func (r *TabunganRepo) ListTanggalPengangkutanCompleted(pengangkutanIDs []string) (map[string]string, error) {
	hasil := make(map[string]string, len(pengangkutanIDs))
	if len(pengangkutanIDs) == 0 {
		return hasil, nil
	}

	type tanggalRow struct {
		SourceID  string `gorm:"column:source_id"`
		ChangedAt string `gorm:"column:changed_at"`
	}

	var rows []tanggalRow
	if err := r.db.Raw(`
		SELECT DISTINCT ON (pengangkutan_id)
			pengangkutan_id AS source_id,
			TO_CHAR(changed_at, 'YYYY-MM-DD"T"HH24:MI:SSOF') AS changed_at
		FROM riwayat_pengangkutan
		WHERE pengangkutan_id IN ?
		  AND status_pengangkutan = ?
		ORDER BY pengangkutan_id, changed_at DESC
	`, pengangkutanIDs, models.StatusCompleted).Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		hasil[row.SourceID] = row.ChangedAt
	}
	return hasil, nil
}
