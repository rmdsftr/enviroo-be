package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/utils"

	"gorm.io/gorm"
)

const barangPageLimit = 12

type BarangRepoIface interface {
	FindBank(bankID string) (*models.BankSampah, error)
	ListMasterBarang(q string) ([]models.Barang, error)
	FindMasterBarang(barangID int) (*models.Barang, error)
	FindOrCreateMasterBarang(tx *gorm.DB, namaBarang string) (*models.Barang, error)
	AdaRewardBarangAktif(bankID string) (bool, error)
	FindKatalogBarang(produkID string) (*models.KatalogBarang, error)
	FindKatalogBarangByBank(tx *gorm.DB, produkID, bankID string) (*models.KatalogBarang, error)
	FindKatalogBarangWithMaster(produkID, bankID string) (*models.KatalogBarang, error)
	CreateKatalogBarang(tx *gorm.DB, katalog *models.KatalogBarang) error
	UpdateKatalogBarang(tx *gorm.DB, produkID string, fields map[string]any) error
	DeleteKatalogBarang(tx *gorm.DB, produkID string) error
	ListBarangBank(bankID, katalogBankID string, limit, offset int) ([]repositories.BarangBankRow, error)
	CountPemakaianBarang(produkID string) (int64, error)
	FindStok(tx *gorm.DB, produkID, bankID string) (*models.StokBarang, error)
	LockStok(tx *gorm.DB, produkID, bankID string) (*models.StokBarang, error)
	CreateStok(tx *gorm.DB, stok *models.StokBarang) error
	SetStok(tx *gorm.DB, produkID, bankID string, nilai float64) error
	TambahStok(tx *gorm.DB, produkID, bankID string, delta float64) error
	DeleteStokByProduk(tx *gorm.DB, produkID string) error
	FindDistribusi(disbaID string) (*models.DistribusiBarangBsu, error)
	CreateDistribusi(tx *gorm.DB, disba *models.DistribusiBarangBsu) error
	UpdateDistribusi(tx *gorm.DB, disbaID string, fields map[string]any) error
	CreateDetailDistribusi(tx *gorm.DB, detail *models.DetailDistribusiBarang) error
	ListDetailDistribusi(disbaID string) ([]models.DetailDistribusiBarang, error)
	UpdateStokDetail(tx *gorm.DB, disbaID, produkID string, fields map[string]any) error
	ListHeaderDistribusi(filterCol, bankID string, start, end *time.Time) ([]repositories.DistribusiHeaderRow, error)
	FindHeaderDistribusi(disbaID string) (*repositories.DistribusiHeaderRow, error)
	ListItemDistribusi(disbaID string) ([]repositories.DistribusiItemRow, error)
	ListRiwayatDistribusiProduk(produkID, bankID string, sebagaiBSU bool) ([]repositories.RiwayatDistribusiRow, error)
	FindAdminTargets(bankID string) ([]repositories.NotifTarget, error)
}

type AksesBarangInput struct {
	ActorBankID string
	ActorRole   models.RoleAdmin
}

type TambahBarangInput struct {
	BankID     string
	NamaBarang string
	BarangID   int
	NilaiPoin  float64
	StokAwal   float64
	CreatedBy  string
	FotoURL    *string
}

type TambahBarangResult struct {
	Katalog models.KatalogBarang `json:"katalog"`
	Stok    models.StokBarang    `json:"stok"`
}

type EditBarangInput struct {
	NilaiPoin    float64
	SetNilaiPoin bool
	Stok         float64
	SetStok      bool
	TambahStok   float64
	UpdatedBy    string
	FotoURL      *string
	Akses        AksesBarangInput
}

type HapusBarangInput struct {
	ProdukID string
	Akses    AksesBarangInput
}

type ItemDistribusiInput struct {
	ProdukID  string
	StokKirim float64
}

type BuatSesiDistribusiInput struct {
	BSIID      string
	BSUID      string
	AdminBSIID string
	Items      []ItemDistribusiInput
	Akses      AksesBarangInput
}

type PreviewDistribusiInput struct {
	BSIID string
	BSUID string
	Items []ItemDistribusiInput
	Akses AksesBarangInput
}

type PreviewDistribusiItem struct {
	ProdukID       string  `json:"produk_id"`
	NamaBarang     string  `json:"nama_barang"`
	PhotoURL       *string `json:"photo_url"`
	NilaiPoin      float64 `json:"nilai_poin"`
	StokKirim      float64 `json:"stok_kirim"`
	StokBSISebelum float64 `json:"stok_bsi_sebelum"`
	StokBSISesudah float64 `json:"stok_bsi_sesudah"`
	StokBSUSebelum float64 `json:"stok_bsu_sebelum"`
	StokBSUSesudah float64 `json:"stok_bsu_sesudah"`
	BSUItemBaru    bool    `json:"bsu_item_baru"`
}

type SelesaikanDistribusiInput struct {
	DisbaID    string
	BSUID      string
	AdminBSUID string
	Akses      AksesBarangInput
}

type DetailDistribusiResult struct {
	Header *repositories.DistribusiHeaderRow `json:"header"`
	Items  []repositories.DistribusiItemRow  `json:"items"`
}

type mutasiStok struct {
	BSISebelum  float64
	BSISesudah  float64
	BSUSebelum  float64
	BSUSesudah  float64
	BSUItemBaru bool
}

type BarangService interface {
	ListMasterBarang(q string) ([]models.Barang, error)
	TambahBarang(in TambahBarangInput) (*TambahBarangResult, error)
	ListBarangBank(bankID string, page int) ([]repositories.BarangBankRow, *utils.PageMeta, error)
	RiwayatDistribusiBarang(produkID, bankID string, in AksesBarangInput) ([]repositories.RiwayatDistribusiRow, error)
	EditBarang(produkID string, in EditBarangInput) (*models.KatalogBarang, error)
	HapusBarang(in HapusBarangInput) error

	BuatSesiDistribusi(in BuatSesiDistribusiInput) (string, error)
	PreviewDistribusi(in PreviewDistribusiInput) ([]PreviewDistribusiItem, []string, error)
	SelesaikanDistribusi(in SelesaikanDistribusiInput) error
	ListDistribusi(bankID, startDate, endDate string) ([]repositories.DistribusiHeaderRow, error)
	DetailDistribusi(disbaID string, in AksesBarangInput) (*DetailDistribusiResult, error)
}

type barangService struct {
	db       *gorm.DB
	repo     BarangRepoIface
	notifSvc NotifikasiService
}

func NewBarangService(db *gorm.DB, repo BarangRepoIface, notifSvc NotifikasiService) BarangService {
	return &barangService{db: db, repo: repo, notifSvc: notifSvc}
}

func (s *barangService) ListMasterBarang(q string) ([]models.Barang, error) {
	results, err := s.repo.ListMasterBarang(q)
	if err != nil {
		return nil, errInternal("Gagal mengambil data barang")
	}
	return results, nil
}

func (s *barangService) TambahBarang(in TambahBarangInput) (*TambahBarangResult, error) {
	adaReward, err := s.repo.AdaRewardBarangAktif(in.BankID)
	if err != nil {
		return nil, errInternal("Gagal memeriksa nilai reward bank: " + err.Error())
	}
	if !adaReward {
		return nil, errBadRequest("Bank ini belum/tidak lagi menyediakan reward Barang. Atur nilai reward terlebih dahulu")
	}

	var barangID int
	if in.BarangID > 0 {
		master, err := s.repo.FindMasterBarang(in.BarangID)
		if err != nil {
			return nil, errBadRequest("barang_id tidak valid")
		}
		barangID = master.BarangID
	} else {
		master, err := s.repo.FindOrCreateMasterBarang(nil, in.NamaBarang)
		if err != nil {
			return nil, errInternal("Gagal memproses master barang: " + err.Error())
		}
		barangID = master.BarangID
	}

	produkID, err := idgen.GenerateProdukID(in.BankID, barangID)
	if err != nil {
		return nil, errInternal("Gagal generate barang ID: " + err.Error())
	}

	bankID := in.BankID
	createdBy := in.CreatedBy
	katalog := models.KatalogBarang{
		ProdukID:  produkID,
		BankID:    &bankID,
		BarangID:  barangID,
		PhotoURL:  in.FotoURL,
		NilaiPoin: in.NilaiPoin,
		CreatedBy: &createdBy,
		UpdatedBy: &createdBy,
	}
	stok := models.StokBarang{ProdukID: produkID, BankID: in.BankID, Stok: in.StokAwal}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.CreateKatalogBarang(tx, &katalog); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return errConflict("Katalog untuk barang ini sudah ada di bank ini", nil)
			}
			return errInternal("Gagal membuat katalog: " + err.Error())
		}
		if err := s.repo.CreateStok(tx, &stok); err != nil {
			return errInternal("Gagal membuat stok awal: " + err.Error())
		}
		return nil
	})
	if txErr != nil {
		return nil, bungkusTxError(txErr, "Gagal menyimpan transaksi")
	}

	return &TambahBarangResult{Katalog: katalog, Stok: stok}, nil
}

func (s *barangService) ListBarangBank(bankID string, page int) ([]repositories.BarangBankRow, *utils.PageMeta, error) {
	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		return nil, nil, errNotFound("Bank tidak ditemukan")
	}

	katalogBankID := bankID
	if bank.JenisBank == models.BSU && bank.ParentBankID != nil {
		katalogBankID = *bank.ParentBankID
	}

	limit, offset := 0, 0
	if page > 0 {
		limit = barangPageLimit
		offset = utils.Offset(page, limit)
	}

	rows, err := s.repo.ListBarangBank(bankID, katalogBankID, limit, offset)
	if err != nil {
		return nil, nil, errInternal("Gagal mengambil data barang")
	}
	if rows == nil {
		rows = []repositories.BarangBankRow{}
	}

	if page <= 0 {
		return rows, nil, nil
	}

	total := 0
	if len(rows) > 0 {
		total = rows[0].TotalCount
	}
	return rows, utils.NewPageMeta(page, limit, total), nil
}

func (s *barangService) RiwayatDistribusiBarang(produkID, bankID string, in AksesBarangInput) ([]repositories.RiwayatDistribusiRow, error) {
	if bankID == "" {
		return nil, errBadRequest("bank_id wajib diisi")
	}

	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if err := pastikanAksesBarang(in, bankID); err != nil {
		return nil, err
	}

	rows, err := s.repo.ListRiwayatDistribusiProduk(produkID, bankID, bank.JenisBank == models.BSU)
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat distribusi")
	}
	if rows == nil {
		rows = []repositories.RiwayatDistribusiRow{}
	}
	return rows, nil
}

func (s *barangService) EditBarang(produkID string, in EditBarangInput) (*models.KatalogBarang, error) {
	katalog, err := s.repo.FindKatalogBarang(produkID)
	if err != nil {
		return nil, errNotFound("Barang tidak ditemukan")
	}
	if katalog.BankID == nil {
		return nil, errInternal("Katalog barang tidak memiliki bank pemilik")
	}
	if err := pastikanAksesBarang(in.Akses, *katalog.BankID); err != nil {
		return nil, err
	}

	if in.TambahStok != 0 && in.SetStok {
		return nil, errBadRequest("tambah stok dan koreksi stok tidak boleh diisi bersamaan. Isi salah satu, tambah stok atau koreksi stok")
	}

	if in.FotoURL != nil {
		katalog.PhotoURL = in.FotoURL
	}

	katalogUpdates := map[string]any{
		"updated_by": in.UpdatedBy,
		"photo_url":  katalog.PhotoURL,
	}
	if in.SetNilaiPoin {
		katalogUpdates["nilai_poin"] = in.NilaiPoin
	}

	bankID := *katalog.BankID
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.UpdateKatalogBarang(tx, produkID, katalogUpdates); err != nil {
			return errInternal("Gagal mengupdate katalog: " + err.Error())
		}

		switch {
		case in.TambahStok != 0:
			if err := s.repo.TambahStok(tx, produkID, bankID, in.TambahStok); err != nil {
				return errInternal("Gagal menambah stok: " + err.Error())
			}
		case in.SetStok:
			if err := s.repo.SetStok(tx, produkID, bankID, in.Stok); err != nil {
				return errInternal("Gagal mengupdate stok: " + err.Error())
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, bungkusTxError(txErr, "Gagal menyimpan transaksi")
	}

	hasil, err := s.repo.FindKatalogBarang(produkID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data barang setelah update")
	}
	return hasil, nil
}

func (s *barangService) HapusBarang(in HapusBarangInput) error {
	katalog, err := s.repo.FindKatalogBarang(in.ProdukID)
	if err != nil {
		return errNotFound("Barang tidak ditemukan")
	}
	if katalog.BankID == nil {
		return errInternal("Katalog barang tidak memiliki bank pemilik")
	}
	if err := pastikanAksesBarang(in.Akses, *katalog.BankID); err != nil {
		return err
	}

	count, err := s.repo.CountPemakaianBarang(in.ProdukID)
	if err != nil {
		return errInternal("Gagal memvalidasi data barang")
	}
	if count > 0 {
		return errConflict("Barang tidak dapat dihapus karena sudah digunakan dalam transaksi.", nil)
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.DeleteStokByProduk(tx, in.ProdukID); err != nil {
			return err
		}
		return s.repo.DeleteKatalogBarang(tx, in.ProdukID)
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return errNotFound("Barang tidak ditemukan")
		}
		return errInternal("Gagal menghapus barang")
	}
	return nil
}

func (s *barangService) BuatSesiDistribusi(in BuatSesiDistribusiInput) (string, error) {
	if err := pastikanAksesBarang(in.Akses, in.BSIID); err != nil {
		return "", err
	}
	if err := s.pastikanBSUCabangDari(in.BSUID, in.BSIID); err != nil {
		return "", err
	}

	var disbaID string
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		id, err := idgen.GenerateDisbaID(tx, in.BSIID)
		if err != nil {
			return errInternal("gagal generate disba ID: " + err.Error())
		}
		disbaID = id

		if err := s.repo.CreateDistribusi(tx, &models.DistribusiBarangBsu{
			DisbaID:          disbaID,
			BsiID:            in.BSIID,
			BsuID:            in.BSUID,
			AdminBsiID:       in.AdminBSIID,
			StatusDistribusi: models.DisbaPending,
		}); err != nil {
			return errInternal("gagal membuat sesi distribusi: " + err.Error())
		}

		var totalItem, totalPoin float64
		for _, item := range in.Items {
			katalog, err := s.repo.FindKatalogBarangByBank(tx, item.ProdukID, in.BSIID)
			if err != nil {
				return errNotFound("barang BSI tidak ditemukan: " + item.ProdukID)
			}

			subtotal := katalog.NilaiPoin * item.StokKirim
			totalItem += item.StokKirim
			totalPoin += subtotal

			if err := s.repo.CreateDetailDistribusi(tx, &models.DetailDistribusiBarang{
				ProdukID:     item.ProdukID,
				DisbaID:      disbaID,
				Item:         item.StokKirim,
				Poin:         katalog.NilaiPoin,
				SubtotalPoin: subtotal,
			}); err != nil {
				return errInternal("gagal menyimpan detail distribusi: " + err.Error())
			}
		}

		if err := s.repo.UpdateDistribusi(tx, disbaID, map[string]any{
			"total_item": totalItem,
			"total_poin": totalPoin,
		}); err != nil {
			return errInternal("gagal mengupdate total distribusi: " + err.Error())
		}
		return nil
	})
	if txErr != nil {
		return "", bungkusTxError(txErr, "Gagal membuat sesi distribusi")
	}

	return disbaID, nil
}

func (s *barangService) PreviewDistribusi(in PreviewDistribusiInput) ([]PreviewDistribusiItem, []string, error) {
	if len(in.Items) == 0 {
		return nil, nil, errBadRequest("Items tidak boleh kosong")
	}
	if err := pastikanAksesBarang(in.Akses, in.BSIID); err != nil {
		return nil, nil, err
	}
	if err := s.pastikanBSUCabangDari(in.BSUID, in.BSIID); err != nil {
		return nil, nil, err
	}

	var previews []PreviewDistribusiItem
	var validationErrors []string

	for _, item := range in.Items {
		katalog, err := s.repo.FindKatalogBarangWithMaster(item.ProdukID, in.BSIID)
		if err != nil {
			validationErrors = append(validationErrors,
				fmt.Sprintf("produk_id '%s' tidak ditemukan di BSI", item.ProdukID))
			continue
		}

		mutasi, err := s.hitungMutasiStok(nil, item.ProdukID, in.BSIID, in.BSUID, item.StokKirim)
		if err != nil {
			svcErr, ok := err.(*ServiceError)
			pesan := err.Error()
			if ok {
				pesan = svcErr.Message
			}
			validationErrors = append(validationErrors,
				fmt.Sprintf("%s: %s", katalog.MasterBarang.NamaBarang, pesan))
			continue
		}

		previews = append(previews, PreviewDistribusiItem{
			ProdukID:       katalog.ProdukID,
			NamaBarang:     katalog.MasterBarang.NamaBarang,
			PhotoURL:       katalog.PhotoURL,
			NilaiPoin:      katalog.NilaiPoin,
			StokKirim:      item.StokKirim,
			StokBSISebelum: mutasi.BSISebelum,
			StokBSISesudah: mutasi.BSISesudah,
			StokBSUSebelum: mutasi.BSUSebelum,
			StokBSUSesudah: mutasi.BSUSesudah,
			BSUItemBaru:    mutasi.BSUItemBaru,
		})
	}

	return previews, validationErrors, nil
}

func (s *barangService) SelesaikanDistribusi(in SelesaikanDistribusiInput) error {
	disba, err := s.repo.FindDistribusi(in.DisbaID)
	if err != nil {
		return errNotFound("Sesi distribusi tidak ditemukan")
	}
	if disba.StatusDistribusi != models.DisbaPending {
		return errBadRequest("Sesi distribusi sudah diproses atau tidak valid")
	}
	if disba.BsuID != in.BSUID {
		return errBadRequest("BSU tidak sesuai dengan sesi distribusi")
	}
	if err := pastikanAksesBarang(in.Akses, disba.BsuID); err != nil {
		return err
	}

	details, err := s.repo.ListDetailDistribusi(in.DisbaID)
	if err != nil {
		return errInternal("Gagal memuat detail distribusi")
	}
	if len(details) == 0 {
		return errBadRequest("Tidak ada item pada sesi distribusi ini")
	}

	bsiID := disba.BsiID
	bsuID := in.BSUID

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		for _, detail := range details {
			mutasi, err := s.hitungMutasiStok(tx, detail.ProdukID, bsiID, bsuID, detail.Item)
			if err != nil {
				return err
			}

			if mutasi.BSUItemBaru {
				if err := s.repo.CreateStok(tx, &models.StokBarang{
					ProdukID: detail.ProdukID,
					BankID:   bsuID,
					Stok:     0,
				}); err != nil {
					return errInternal("gagal membuat stok BSU: " + err.Error())
				}
			}

			if err := s.repo.SetStok(tx, detail.ProdukID, bsiID, mutasi.BSISesudah); err != nil {
				return errInternal("gagal update stok BSI: " + err.Error())
			}
			if err := s.repo.SetStok(tx, detail.ProdukID, bsuID, mutasi.BSUSesudah); err != nil {
				return errInternal("gagal update stok BSU: " + err.Error())
			}

			if err := s.repo.UpdateStokDetail(tx, in.DisbaID, detail.ProdukID, map[string]any{
				"stok_bsi_sebelum": mutasi.BSISebelum,
				"stok_bsi_sesudah": mutasi.BSISesudah,
				"stok_bsu_sebelum": mutasi.BSUSebelum,
				"stok_bsu_sesudah": mutasi.BSUSesudah,
			}); err != nil {
				return errInternal("gagal update stok detail distribusi: " + err.Error())
			}
		}

		if err := s.repo.UpdateDistribusi(tx, in.DisbaID, map[string]any{
			"admin_bsu_id":      in.AdminBSUID,
			"status_distribusi": models.DisbaSelesai,
		}); err != nil {
			return errInternal("gagal mengupdate header distribusi: " + err.Error())
		}
		return nil
	})
	if txErr != nil {
		return bungkusTxError(txErr, "Gagal memproses distribusi barang")
	}

	s.kirimNotifDistribusi(bsiID, bsuID, len(details))
	return nil
}

func (s *barangService) ListDistribusi(bankID, startDate, endDate string) ([]repositories.DistribusiHeaderRow, error) {
	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	var filterCol string
	switch bank.JenisBank {
	case models.BSI:
		filterCol = "d.bsi_id"
	case models.BSU:
		filterCol = "d.bsu_id"
	default:
		return nil, errForbidden("Akses ditolak: jenis bank tidak diizinkan")
	}

	var start, end *time.Time
	if t, err := time.Parse("2006-01-02", startDate); err == nil {
		start = &t
	}
	if t, err := time.Parse("2006-01-02", endDate); err == nil {
		akhirHari := t.Add(24*time.Hour - time.Second)
		end = &akhirHari
	}

	rows, err := s.repo.ListHeaderDistribusi(filterCol, bankID, start, end)
	if err != nil {
		return nil, errInternal("Gagal mengambil data distribusi")
	}
	if rows == nil {
		rows = []repositories.DistribusiHeaderRow{}
	}
	return rows, nil
}

func (s *barangService) DetailDistribusi(disbaID string, in AksesBarangInput) (*DetailDistribusiResult, error) {
	hdr, err := s.repo.FindHeaderDistribusi(disbaID)
	if err != nil || hdr.DisbaID == "" {
		return nil, errNotFound("Distribusi tidak ditemukan")
	}

	items, err := s.repo.ListItemDistribusi(disbaID)
	if err != nil {
		return nil, errInternal("Gagal mengambil detail distribusi: " + err.Error())
	}
	if items == nil {
		items = []repositories.DistribusiItemRow{}
	}

	return &DetailDistribusiResult{Header: hdr, Items: items}, nil
}

func (s *barangService) hitungMutasiStok(tx *gorm.DB, produkID, bsiID, bsuID string, jumlah float64) (*mutasiStok, error) {
	var stokBSI *models.StokBarang
	var err error
	if tx != nil {
		stokBSI, err = s.repo.LockStok(tx, produkID, bsiID)
	} else {
		stokBSI, err = s.repo.FindStok(nil, produkID, bsiID)
	}
	if err != nil {
		return nil, errNotFound("stok BSI tidak ditemukan")
	}
	if stokBSI.Stok < jumlah {
		return nil, errConflict(fmt.Sprintf("stok BSI tidak mencukupi (tersedia: %.2f, diminta: %.2f)", stokBSI.Stok, jumlah), nil)
	}

	var stokBSU *models.StokBarang
	if tx != nil {
		stokBSU, err = s.repo.LockStok(tx, produkID, bsuID)
	} else {
		stokBSU, err = s.repo.FindStok(nil, produkID, bsuID)
	}

	bsuSebelum := 0.0
	bsuItemBaru := false
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		bsuItemBaru = true
	case err != nil:
		return nil, errInternal("gagal mengecek stok BSU: " + err.Error())
	default:
		bsuSebelum = stokBSU.Stok
	}

	return &mutasiStok{
		BSISebelum:  stokBSI.Stok,
		BSISesudah:  stokBSI.Stok - jumlah,
		BSUSebelum:  bsuSebelum,
		BSUSesudah:  bsuSebelum + jumlah,
		BSUItemBaru: bsuItemBaru,
	}, nil
}

func (s *barangService) pastikanBSUCabangDari(bsuID, bsiID string) error {
	bsu, err := s.repo.FindBank(bsuID)
	if err != nil || bsu.JenisBank != models.BSU {
		return errNotFound("BSU tidak ditemukan")
	}
	if bsu.ParentBankID == nil || *bsu.ParentBankID != bsiID {
		return errBadRequest("BSU ini bukan cabang dari BSI tersebut")
	}
	return nil
}

func (s *barangService) kirimNotifDistribusi(bsiID, bsuID string, totalJenisBarang int) {
	go func() {
		bankBSI, err := s.repo.FindBank(bsiID)
		if err != nil {
			return
		}
		bankBSU, err := s.repo.FindBank(bsuID)
		if err != nil {
			return
		}

		targets, err := s.repo.FindAdminTargets(bsuID)
		if err != nil {
			log.Printf("[Notif] Gagal query admin BSU %s: %v", bsuID, err)
			return
		}

		refID := fmt.Sprintf("%s-to-%s", bsiID, bsuID)
		ctx := context.Background()
		for _, t := range targets {
			if err := s.notifSvc.NotifDistribusiBarangBerhasil(ctx, t.UserID, t.FCMToken,
				bankBSU.NamaBank, totalJenisBarang, bankBSI.NamaBank, refID); err != nil {
				log.Printf("[Notif] Gagal kirim notif distribusi barang ke user %s: %v", t.UserID, err)
			}
		}
	}()
}

func pastikanAksesBarang(in AksesBarangInput, bankID string) error {
	if in.ActorBankID == "" {
		return nil
	}
	if in.ActorRole == models.AdminBSI || in.ActorRole == models.PetugasBSI {
		return nil
	}
	if in.ActorBankID != bankID {
		return errForbidden("Akses ditolak. Anda tidak memiliki akses ke data barang bank ini")
	}
	return nil
}
