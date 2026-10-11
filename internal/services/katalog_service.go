package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/utils"

	"gorm.io/gorm"
)

const katalogPageLimit = 12

type KatalogRepoIface interface {
	FindBank(bankID string) (*models.BankSampah, error)
	ListMasterSampah(q string) ([]models.Sampah, error)
	FindMasterSampah(sarokID int) (*models.Sampah, error)
	FindOrCreateMasterSampah(tx *gorm.DB, namaSampah string, satuan models.SatuanEnum) (*models.Sampah, error)
	AdaNilaiRewardAktif(bankID string, rewardID int) (bool, error)
	FindKatalog(sampahID string) (*models.KatalogSampah, error)
	FindKatalogWithRelasi(sampahID string) (*models.KatalogSampah, error)
	FindKatalogSejenis(sarokID int, bankID string) (*models.KatalogSampah, error)
	ListKatalogByBank(bankID string, limit, offset int) ([]models.KatalogSampah, error)
	CountKatalogByBank(bankID string) (int64, error)
	CreateKatalog(tx *gorm.DB, katalog *models.KatalogSampah) error
	SaveKatalog(tx *gorm.DB, katalog *models.KatalogSampah) error
	UpdateKategoriSejenis(tx *gorm.DB, sarokID int, bankID string, kategoriID int) error
	DeleteKatalog(katalog *models.KatalogSampah) error
	CountPemakaianKatalog(sampahID string) (string, error)
	ListStokByBank(bankID string, sampahIDs []string) (map[string]float64, error)
	FindStok(bankID, sampahID string) (float64, error)
	CreateStok(tx *gorm.DB, stok *models.StokSampah) error
	ListSchemaHarga(sampahID string) ([]models.SchemaHargaSampah, error)
	ListHistoryHarga(sampahID string) ([]repositories.HistoryHargaRow, error)
	ListKategori() ([]models.KategoriSampah, error)
	FindKategori(kategoriID string) (*models.KategoriSampah, error)
	CountKategoriDenganNama(namaNormal, exceptID string) (int64, error)
	CreateKategori(kategori *models.KategoriSampah) error
	SaveKategori(kategori *models.KategoriSampah) error
	DeleteKategori(kategori *models.KategoriSampah) error
	CountKatalogByKategori(kategoriID string) (int64, error)
}

type TambahKatalogInput struct {
	BankID          string
	NamaSampah      string
	SarokID         int
	Satuan          models.SatuanEnum
	KategoriID      int
	RewardID        int
	SyaratPemilahan string
	FotoURL         string
}

type TambahKatalogResult struct {
	Katalog models.KatalogSampah `json:"katalog"`
	Stok    models.StokSampah    `json:"stok"`
}

type EditKatalogInput struct {
	SyaratPemilahan string
	KategoriID      int
	FotoURL         string
	AdaFileFoto     bool
	Akses           AksesKatalogInput
}

type HapusKatalogInput struct {
	SampahID string
	Akses    AksesKatalogInput
}

type AksesKatalogInput struct {
	ActorBankID string
	ActorRole   models.RoleAdmin
}

type KatalogItem struct {
	SampahID        string                `json:"sampah_id"`
	SarokID         int                   `json:"sarok_id"`
	NamaSampah      string                `json:"nama_sampah"`
	PhotoURL        string                `json:"photo_url"`
	Satuan          models.SatuanEnum     `json:"satuan"`
	BankID          string                `json:"bank_id"`
	KategoriID      int                   `json:"kategori_id"`
	RewardID        int                   `json:"reward_id"`
	SyaratPemilahan string                `json:"syarat_pemilahan"`
	Stok            float64               `json:"stok"`
	Kategori        models.KategoriSampah `json:"kategori"`
	Reward          models.Reward         `json:"reward"`
}

type HargaLevelItem struct {
	SchemaID     string                  `json:"schema_id"`
	LevelUser    models.LevelUser        `json:"level_user"`
	Harga        float64                 `json:"harga"`
	SatuanReward models.SatuanRewardEnum `json:"satuan_reward"`
}

type DetailKatalogResult struct {
	SampahID        string                         `json:"sampah_id"`
	SarokID         int                            `json:"sarok_id"`
	NamaSampah      string                         `json:"nama_sampah"`
	PhotoURL        string                         `json:"photo_url"`
	Satuan          models.SatuanEnum              `json:"satuan"`
	SyaratPemilahan string                         `json:"syarat_pemilahan"`
	BankID          string                         `json:"bank_id"`
	Kategori        models.KategoriSampah          `json:"kategori"`
	Reward          models.Reward                  `json:"reward"`
	Stok            float64                        `json:"stok"`
	HargaPerLevel   []HargaLevelItem               `json:"harga_per_level"`
	HistoryHarga    []repositories.HistoryHargaRow `json:"history_harga"`
}

type KatalogService interface {
	ListMasterSampah(q string) ([]models.Sampah, error)

	TambahKatalog(in TambahKatalogInput) (*TambahKatalogResult, error)
	EditKatalog(sampahID string, in EditKatalogInput) (*models.KatalogSampah, error)
	HapusKatalog(in HapusKatalogInput) error
	ListKatalogBank(bankID string, page int) ([]KatalogItem, *utils.PageMeta, error)
	DetailKatalog(sampahID string, in AksesKatalogInput) (*DetailKatalogResult, error)

	ListKategori() ([]models.KategoriSampah, error)
	TambahKategori(nama string) (*models.KategoriSampah, error)
	UbahKategori(kategoriID, nama string) (*models.KategoriSampah, error)
	HapusKategori(kategoriID string) error
}

type katalogService struct {
	db   *gorm.DB
	repo KatalogRepoIface
}

func NewKatalogService(db *gorm.DB, repo KatalogRepoIface) KatalogService {
	return &katalogService{db: db, repo: repo}
}

func (s *katalogService) ListMasterSampah(q string) ([]models.Sampah, error) {
	results, err := s.repo.ListMasterSampah(q)
	if err != nil {
		return nil, errInternal("Gagal mengambil data sampah")
	}
	return results, nil
}

func (s *katalogService) TambahKatalog(in TambahKatalogInput) (*TambahKatalogResult, error) {
	bank, err := s.repo.FindBank(in.BankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank == models.BSU {
		return nil, errForbidden("BSU tidak dapat menambah katalog sampah. Katalog diwarisi dari BSI induk.")
	}
	if err := validasiSatuan(in.Satuan); err != nil {
		return nil, err
	}

	adaReward, err := s.repo.AdaNilaiRewardAktif(in.BankID, in.RewardID)
	if err != nil {
		return nil, errInternal("Gagal memeriksa nilai reward bank: " + err.Error())
	}
	if !adaReward {
		return nil, errBadRequest("Bank ini belum/tidak lagi menyediakan reward tersebut, atau jenis reward tidak valid. Atur nilai reward terlebih dahulu")
	}

	var sarokID int
	if in.SarokID > 0 {
		master, err := s.repo.FindMasterSampah(in.SarokID)
		if err != nil {
			return nil, errBadRequest("sarok_id tidak valid")
		}
		if master.Satuan != in.Satuan {
			master, err = s.repo.FindOrCreateMasterSampah(nil, master.NamaSampah, in.Satuan)
			if err != nil {
				return nil, errInternal("Gagal memproses master sampah: " + err.Error())
			}
		}
		sarokID = master.SarokID
	} else {
		master, err := s.repo.FindOrCreateMasterSampah(nil, in.NamaSampah, in.Satuan)
		if err != nil {
			return nil, errInternal("Gagal memproses master sampah: " + err.Error())
		}
		sarokID = master.SarokID
	}

	sampahID, err := idgen.GenerateSampahID(in.BankID, sarokID, in.RewardID)
	if err != nil {
		return nil, errInternal("Gagal generate sampah ID: " + err.Error())
	}

	fotoURL, syarat := in.FotoURL, in.SyaratPemilahan

	sejenis, err := s.repo.FindKatalogSejenis(sarokID, in.BankID)
	switch {
	case err == nil:
		if sejenis.KategoriID != in.KategoriID {
			return nil, errConflict(fmt.Sprintf(
				"Jenis sampah ini sudah terdaftar dengan kategori %s di bank ini. Kategori untuk jenis sampah yang sama harus sama",
				sejenis.Kategori.Kategori), nil)
		}

		if fotoURL == "" {
			fotoURL = sejenis.PhotoURL
		}
		if strings.TrimSpace(syarat) == "" {
			syarat = sejenis.SyaratPemilahan
		}
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, errInternal("Gagal memeriksa katalog yang sudah ada: " + err.Error())
	}

	katalog := models.KatalogSampah{
		SampahID:        sampahID,
		SarokID:         sarokID,
		BankID:          in.BankID,
		PhotoURL:        fotoURL,
		KategoriID:      in.KategoriID,
		RewardID:        in.RewardID,
		SyaratPemilahan: syarat,
	}
	stok := models.StokSampah{BankID: in.BankID, SampahID: sampahID, Stok: 0}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.CreateKatalog(tx, &katalog); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return errConflict("Katalog untuk jenis sampah dan reward ini sudah ada di bank ini", nil)
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

	return &TambahKatalogResult{Katalog: katalog, Stok: stok}, nil
}

func (s *katalogService) EditKatalog(sampahID string, in EditKatalogInput) (*models.KatalogSampah, error) {
	katalog, bank, err := s.ambilKatalogDanBank(sampahID)
	if err != nil {
		return nil, err
	}
	if err := pastikanAksesKatalog(in.Akses, katalog.BankID); err != nil {
		return nil, err
	}
	if bank.JenisBank == models.BSU {
		return nil, errForbidden("BSU tidak dapat mengedit katalog sampah")
	}

	ubahKategori := in.KategoriID != 0 && in.KategoriID != katalog.KategoriID
	if ubahKategori {
		if _, err := s.repo.FindKategori(strconv.Itoa(in.KategoriID)); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errBadRequest("kategori_id tidak valid")
			}
			return nil, errInternal("Gagal memeriksa kategori: " + err.Error())
		}
		katalog.KategoriID = in.KategoriID
	}

	katalog.SyaratPemilahan = in.SyaratPemilahan
	if in.AdaFileFoto {
		katalog.PhotoURL = in.FotoURL
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.SaveKatalog(tx, katalog); err != nil {
			return errInternal("Gagal mengupdate katalog: " + err.Error())
		}
		if ubahKategori {
			if err := s.repo.UpdateKategoriSejenis(tx, katalog.SarokID, katalog.BankID, in.KategoriID); err != nil {
				return errInternal("Gagal mengupdate kategori katalog sejenis: " + err.Error())
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, bungkusTxError(txErr, "Gagal menyimpan transaksi")
	}
	return katalog, nil
}

func (s *katalogService) HapusKatalog(in HapusKatalogInput) error {
	katalog, bank, err := s.ambilKatalogDanBank(in.SampahID)
	if err != nil {
		return err
	}
	if err := pastikanAksesKatalog(in.Akses, katalog.BankID); err != nil {
		return err
	}
	if bank.JenisBank == models.BSU {
		return errForbidden("BSU tidak dapat menghapus katalog sampah")
	}

	pemakai, err := s.repo.CountPemakaianKatalog(in.SampahID)
	if err != nil {
		return errInternal("Gagal memvalidasi pemakaian katalog: " + err.Error())
	}
	if pemakai != "" {
		return errConflict("Sampah ini tidak dapat dihapus karena sudah digunakan dalam data "+pemakai+".", nil)
	}

	if err := s.repo.DeleteKatalog(katalog); err != nil {
		if errors.Is(err, repositories.ErrKatalogMasihDipakai) {
			return errConflict("Sampah ini tidak dapat dihapus karena sudah digunakan dalam transaksi.", nil)
		}
		return errInternal("Gagal menghapus katalog: " + err.Error())
	}
	return nil
}

func (s *katalogService) ListKatalogBank(bankID string, page int) ([]KatalogItem, *utils.PageMeta, error) {
	katalogBankID, err := s.resolveBankKatalog(bankID)
	if err != nil {
		return nil, nil, err
	}

	limit, offset := 0, 0
	if page > 0 {
		limit = katalogPageLimit
		offset = utils.Offset(page, limit)
	}

	katalogs, err := s.repo.ListKatalogByBank(katalogBankID, limit, offset)
	if err != nil {
		return nil, nil, errInternal("Gagal mengambil katalog")
	}

	sampahIDs := make([]string, 0, len(katalogs))
	for _, k := range katalogs {
		sampahIDs = append(sampahIDs, k.SampahID)
	}
	stokMap, err := s.repo.ListStokByBank(bankID, sampahIDs)
	if err != nil {
		return nil, nil, errInternal("Gagal mengambil stok katalog")
	}

	items := make([]KatalogItem, 0, len(katalogs))
	for _, k := range katalogs {
		items = append(items, KatalogItem{
			SampahID:        k.SampahID,
			SarokID:         k.SarokID,
			NamaSampah:      k.Sarok.NamaSampah,
			PhotoURL:        k.PhotoURL,
			Satuan:          k.Sarok.Satuan,
			BankID:          k.BankID,
			KategoriID:      k.KategoriID,
			RewardID:        k.RewardID,
			SyaratPemilahan: k.SyaratPemilahan,
			Stok:            stokMap[k.SampahID],
			Kategori:        k.Kategori,
			Reward:          k.Reward,
		})
	}

	if page <= 0 {
		return items, nil, nil
	}

	total, err := s.repo.CountKatalogByBank(katalogBankID)
	if err != nil {
		return nil, nil, errInternal("Gagal menghitung katalog")
	}
	return items, utils.NewPageMeta(page, limit, int(total)), nil
}

func (s *katalogService) DetailKatalog(sampahID string, in AksesKatalogInput) (*DetailKatalogResult, error) {
	katalog, err := s.repo.FindKatalogWithRelasi(sampahID)
	if err != nil {
		return nil, errNotFound("Katalog sampah tidak ditemukan")
	}
	if err := pastikanAksesKatalog(in, katalog.BankID); err != nil {
		return nil, err
	}

	stok, err := s.repo.FindStok(katalog.BankID, sampahID)
	if err != nil {
		return nil, errInternal("Gagal mengambil stok sampah")
	}

	schemas, err := s.repo.ListSchemaHarga(sampahID)
	if err != nil {
		return nil, errInternal("Gagal mengambil harga sampah")
	}
	hargaPerLevel := make([]HargaLevelItem, 0, len(schemas))
	for _, sc := range schemas {
		hargaPerLevel = append(hargaPerLevel, HargaLevelItem{
			SchemaID:     sc.SchemaID,
			LevelUser:    sc.LevelUser,
			Harga:        sc.Harga,
			SatuanReward: sc.SatuanReward,
		})
	}

	histories, err := s.repo.ListHistoryHarga(sampahID)
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat harga")
	}
	if histories == nil {
		histories = []repositories.HistoryHargaRow{}
	}

	return &DetailKatalogResult{
		SampahID:        katalog.SampahID,
		SarokID:         katalog.Sarok.SarokID,
		NamaSampah:      katalog.Sarok.NamaSampah,
		PhotoURL:        katalog.PhotoURL,
		Satuan:          katalog.Sarok.Satuan,
		SyaratPemilahan: katalog.SyaratPemilahan,
		BankID:          katalog.BankID,
		Kategori:        katalog.Kategori,
		Reward:          katalog.Reward,
		Stok:            stok,
		HargaPerLevel:   hargaPerLevel,
		HistoryHarga:    histories,
	}, nil
}

func (s *katalogService) ListKategori() ([]models.KategoriSampah, error) {
	kategori, err := s.repo.ListKategori()
	if err != nil {
		return nil, errInternal("Gagal mengambil kategori: " + err.Error())
	}
	return kategori, nil
}

func (s *katalogService) TambahKategori(nama string) (*models.KategoriSampah, error) {
	if err := s.pastikanNamaKategoriUnik(nama, ""); err != nil {
		return nil, err
	}

	kategori := &models.KategoriSampah{Kategori: nama}
	if err := s.repo.CreateKategori(kategori); err != nil {
		return nil, errInternal("Gagal membuat kategori: " + err.Error())
	}
	return kategori, nil
}

func (s *katalogService) UbahKategori(kategoriID, nama string) (*models.KategoriSampah, error) {
	kategori, err := s.repo.FindKategori(kategoriID)
	if err != nil {
		return nil, errNotFound("Kategori tidak ditemukan")
	}
	if err := s.pastikanNamaKategoriUnik(nama, kategoriID); err != nil {
		return nil, err
	}

	kategori.Kategori = nama
	if err := s.repo.SaveKategori(kategori); err != nil {
		return nil, errInternal("Gagal mengupdate kategori: " + err.Error())
	}
	return kategori, nil
}

func (s *katalogService) HapusKategori(kategoriID string) error {
	kategori, err := s.repo.FindKategori(kategoriID)
	if err != nil {
		return errNotFound("Kategori tidak ditemukan")
	}

	count, err := s.repo.CountKatalogByKategori(kategoriID)
	if err != nil {
		return errInternal("Gagal memvalidasi pemakaian kategori: " + err.Error())
	}
	if count > 0 {
		return errConflict("Kategori masih digunakan oleh katalog sampah dan tidak dapat dihapus", nil)
	}

	if err := s.repo.DeleteKategori(kategori); err != nil {
		return errInternal("Gagal menghapus kategori: " + err.Error())
	}
	return nil
}

func validasiSatuan(satuan models.SatuanEnum) error {
	switch satuan {
	case models.SatuanKG, models.SatuanPCS, models.SatuanLITER:
		return nil
	}
	return errBadRequest("satuan harus salah satu dari: kg, pcs, liter")
}

func (s *katalogService) resolveBankKatalog(bankID string) (string, error) {
	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		return "", errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank != models.BSU {
		return bankID, nil
	}
	if bank.ParentBankID == nil {
		return "", errBadRequest("BSU ini tidak memiliki BSI induk")
	}
	return *bank.ParentBankID, nil
}

func (s *katalogService) ambilKatalogDanBank(sampahID string) (*models.KatalogSampah, *models.BankSampah, error) {
	katalog, err := s.repo.FindKatalog(sampahID)
	if err != nil {
		return nil, nil, errNotFound("Katalog sampah tidak ditemukan")
	}
	bank, err := s.repo.FindBank(katalog.BankID)
	if err != nil {
		return nil, nil, errNotFound("Bank pemilik katalog tidak ditemukan")
	}
	return katalog, bank, nil
}

func (s *katalogService) pastikanNamaKategoriUnik(nama, exceptID string) error {
	count, err := s.repo.CountKategoriDenganNama(utils.NormalizeString(nama), exceptID)
	if err != nil {
		return errInternal("Gagal memvalidasi nama kategori: " + err.Error())
	}
	if count > 0 {
		return errConflict("Kategori dengan nama tersebut sudah ada", nil)
	}
	return nil
}

func pastikanAksesKatalog(in AksesKatalogInput, katalogBankID string) error {
	if in.ActorBankID == "" {
		return nil
	}
	if in.ActorRole == models.AdminBSI || in.ActorRole == models.PetugasBSI {
		return nil
	}
	if in.ActorBankID != katalogBankID {
		return errForbidden("Akses ditolak. Anda tidak memiliki akses ke katalog sampah ini")
	}
	return nil
}
