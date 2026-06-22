package services

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/pkg/utils"
	"time"

	"gorm.io/gorm"
)

type DistribusiSisaRepoIface interface {
	GetBankByID(bankID string) (*models.BankSampah, error)
	GetBankByIDAndJenis(bankID string, jenis []models.JenisBank) (*models.BankSampah, error)
	GetSetting(bankID string) (*models.SettingSisaBagiHasil, error)
	CreateSetting(s *models.SettingSisaBagiHasil) error
	SaveSetting(s *models.SettingSisaBagiHasil) error
	GetBagiHasilWithBank(bagiHasilID string) (*models.BagiHasil, error)
	FindDistribusiByBagiHasil(bagiHasilID string) (*models.DistribusiSisa, bool)
	GetBSUList(db *gorm.DB, bsiID string) ([]models.BankSampah, error)
	GetRewardByNama(nama models.RewardEnum) (*models.Reward, error)
	GetDetailPenjualan(db *gorm.DB, penjualanID string) ([]models.DetailPenjualan, error)
	GetKontribusiPerBSU(db *gorm.DB, bagiHasilID string) ([]repositories.KontribusiBSURow, error)
	GetTransportSplitPerBSU(db *gorm.DB, bagiHasilID string) ([]repositories.TransportSplitRow, error)
	CreateDistribusiSisa(db *gorm.DB, d *models.DistribusiSisa) error
	CreatePenerimaSisa(db *gorm.DB, p *models.PenerimaDistribusiSisa) error
	UpdateSaldoRekening(db *gorm.DB, bankID *string, nasabahID *string, rewardID int, reward models.Reward, nilai float64, entitas models.EntitasEnum, adminID string, now time.Time) error
	SetPenerimaSisaID(db *gorm.DB, bagiHasilID, bsuID, penerimaSisaID string) error
	GetAdminsByBankID(bankID string) ([]models.Admin, error)
	GetDistribusiSisa(distribusiID string) (*models.DistribusiSisa, error)
	GetAdminNama(adminID string) string
	GetPenerimaBSI(distribusiID string) (*models.PenerimaDistribusiSisa, error)
	GetListPenerimaBSU(distribusiID string) ([]models.PenerimaDistribusiSisa, error)
	GetPenerimaByBank(bankID string, jenis models.JenisTerimaEnum, startDate, endDate time.Time) ([]models.PenerimaDistribusiSisa, error)
	GetPenerimaSisaWithDetails(penerimaSisaID string) (*models.PenerimaDistribusiSisa, error)
	GetPerhitunganSisa(penerimaSisaID string) ([]models.PerhitunganSisa, error)
}

type DistribusiSisaService struct {
	db   *gorm.DB
	repo DistribusiSisaRepoIface
}

func NewDistribusiSisaService(db *gorm.DB, repo *repositories.DistribusiSisaRepo) *DistribusiSisaService {
	return &DistribusiSisaService{db: db, repo: repo}
}

func NewDistribusiSisaServiceWithRepo(db *gorm.DB, repo DistribusiSisaRepoIface) *DistribusiSisaService {
	return &DistribusiSisaService{db: db, repo: repo}
}

// ── Request types ───────────────────────────────────────────────────────────

type AddKonfigurasiReq struct {
	PorsiBSU       float64 `json:"porsi_bsu" binding:"required,gt=0"`
	PorsiBSI       float64 `json:"porsi_bsi" binding:"required,gt=0"`
	PorsiTransport float64 `json:"porsi_transport" binding:"required,gt=0"`
}

type UpdateKonfigurasiReq struct {
	PorsiBSU       *float64 `json:"porsi_bsu"`
	PorsiBSI       *float64 `json:"porsi_bsi"`
	PorsiTransport *float64 `json:"porsi_transport"`
}

type SubmitDistribusiReq struct {
	AdminID string `json:"admin_id" binding:"required"`
}

// ── Response types ──────────────────────────────────────────────────────────

type KonfigurasiResp struct {
	SettingID      int64   `json:"setting_id"`
	BankID         string  `json:"bank_id"`
	NamaBank       string  `json:"nama_bank"`
	PorsiBSU       float64 `json:"porsi_bsu"`
	PorsiBSI       float64 `json:"porsi_bsi"`
	PorsiTransport float64 `json:"porsi_transport"`
	TotalPorsi     float64 `json:"total_porsi"`
}

type BSUPreview struct {
	BankID           string  `json:"bank_id"`
	NamaBank         string  `json:"nama_bank"`
	PersenKontribusi float64 `json:"persen_kontribusi"`
	Pokok            float64 `json:"pokok"`
	Transportasi     float64 `json:"transportasi"`
	Nominal          float64 `json:"nominal"`
}

type PreviewDistribusiResp struct {
	BagiHasilID string                  `json:"bagi_hasil_id"`
	TotalSisa   float64                 `json:"total_sisa"`
	Satuan      models.SatuanRewardEnum `json:"satuan"`
	NominalBSI  float64                 `json:"nominal_bsi"`
	PenerimaBSU []BSUPreview            `json:"penerima_bsu"`
}

type RingkasanBSU struct {
	BankID         string  `json:"bank_id"`
	NamaBank       string  `json:"nama_bank"`
	Nominal        float64 `json:"nominal"`
	PenerimaSisaID string  `json:"-"`
}

type SubmitDistribusiResp struct {
	DistribusiID string                  `json:"distribusi_id"`
	BagiHasilID  string                  `json:"bagi_hasil_id"`
	TotalSisa    float64                 `json:"total_sisa"`
	Satuan       models.SatuanRewardEnum `json:"satuan"`
	NominalBSI   float64                 `json:"nominal_bsi"`
	PenerimaBSU  []RingkasanBSU          `json:"penerima_bsu"`
	NamaBSI      string                  `json:"-"`
}

type PenerimaBSIDetailResp struct {
	PenerimaSisaID  string                    `json:"penerima_sisa_id"`
	BankID          string                    `json:"bank_id"`
	NamaBank        string                    `json:"nama_bank"`
	NominalDiterima float64                   `json:"nominal_diterima"`
	Porsi           float64                   `json:"porsi"`
	Transportasi    float64                   `json:"transportasi"`
	SatuanNominal   string                    `json:"satuan_nominal"`
	TransportDetail models.TransportDetail    `json:"transport_detail,omitempty"`
}

type PenerimaBSUDetailResp struct {
	PenerimaSisaID  string  `json:"penerima_sisa_id"`
	BankID          string  `json:"bank_id"`
	NamaBank        string  `json:"nama_bank"`
	NominalDiterima float64 `json:"nominal_diterima"`
	Porsi           float64 `json:"porsi"`
	Transportasi    float64 `json:"transportasi"`
	SatuanNominal   string  `json:"satuan_nominal"`
}

type DetailDistribusiResp struct {
	DistribusiID string                  `json:"distribusi_id"`
	BagiHasilID  string                  `json:"bagi_hasil_id"`
	TotalSisa    float64                 `json:"total_sisa"`
	Satuan       string                  `json:"satuan"`
	CreatedAt    time.Time               `json:"created_at"`
	CreatedBy    string                  `json:"created_by"`
	PenerimaBSI  PenerimaBSIDetailResp   `json:"penerima_bsi"`
	PenerimaBSU  []PenerimaBSUDetailResp `json:"penerima_bsu"`
}

type ListItemResp struct {
	PenerimaSisaID    string    `json:"penerima_sisa_id"`
	DistribusiID      string    `json:"distribusi_id"`
	BagiHasilID       string    `json:"bagi_hasil_id"`
	NominalDiterima   float64   `json:"nominal_diterima"`
	Porsi             float64   `json:"porsi"`
	Transportasi      float64   `json:"transportasi"`
	SatuanNominal     string    `json:"satuan_nominal"`
	TanggalDistribusi time.Time `json:"tanggal_distribusi"`
}

type ListBagiHasilBankResp struct {
	BankID    string           `json:"bank_id"`
	NamaBank  string           `json:"nama_bank"`
	JenisBank models.JenisBank `json:"jenis_bank"`
	Riwayat   []ListItemResp   `json:"riwayat"`
}

type PerhitunganResp struct {
	TabunganID string  `json:"tabungan_id"`
	NamaSampah string  `json:"nama_sampah"`
	QtyDipakai float64 `json:"qty_dipakai"`
	Satuan     string  `json:"satuan"`
}

type DetailBagiHasilBankResp struct {
	PenerimaSisaID    string            `json:"penerima_sisa_id"`
	DistribusiID      string            `json:"distribusi_id"`
	BagiHasilID       string            `json:"bagi_hasil_id"`
	BankID            string            `json:"bank_id"`
	NamaBank          string            `json:"nama_bank"`
	NominalDiterima   float64           `json:"nominal_diterima"`
	Porsi             float64           `json:"porsi"`
	Transportasi      float64           `json:"transportasi"`
	SatuanNominal     string            `json:"satuan_nominal"`
	TanggalDistribusi time.Time         `json:"tanggal_distribusi"`
	PerhitunganSisa   []PerhitunganResp `json:"perhitungan_sisa"`
}

// ── Service Methods ──────────────────────────────────────────────────────────

func (s *DistribusiSisaService) GetKonfigurasi(bankID string) (*KonfigurasiResp, error) {
	bank, err := s.repo.GetBankByID(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank != models.BSI {
		return nil, errForbidden("Hanya bank sampah bertipe BSI yang memiliki konfigurasi ini")
	}
	setting, err := s.repo.GetSetting(bankID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errNotFound("Konfigurasi belum diatur untuk bank ini")
		}
		return nil, errInternal("Gagal mengambil konfigurasi")
	}
	return &KonfigurasiResp{
		SettingID:      setting.SettingID,
		BankID:         setting.BankID,
		NamaBank:       bank.NamaBank,
		PorsiBSU:       setting.PorsiBSU,
		PorsiBSI:       setting.PorsiBSI,
		PorsiTransport: setting.PorsiTransport,
		TotalPorsi:     setting.PorsiBSU + setting.PorsiBSI + setting.PorsiTransport,
	}, nil
}

func (s *DistribusiSisaService) AddKonfigurasi(bankID string, req AddKonfigurasiReq) (*KonfigurasiResp, error) {
	bank, err := s.repo.GetBankByID(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank != models.BSI {
		return nil, errForbidden("Hanya bank sampah bertipe BSI yang dapat mengatur konfigurasi ini")
	}

	total := req.PorsiBSU + req.PorsiBSI + req.PorsiTransport
	if total < 99.99 || total > 100.01 {
		return nil, errBadRequest("Total porsi harus berjumlah 100%")
	}

	if existing, getErr := s.repo.GetSetting(bankID); getErr == nil {
		return nil, errConflict(
			"Konfigurasi untuk bank ini sudah ada, gunakan PATCH untuk mengubahnya",
			map[string]interface{}{"setting_id": existing.SettingID},
		)
	}

	setting := &models.SettingSisaBagiHasil{
		BankID:         bankID,
		PorsiBSU:       req.PorsiBSU,
		PorsiBSI:       req.PorsiBSI,
		PorsiTransport: req.PorsiTransport,
	}
	if err := s.repo.CreateSetting(setting); err != nil {
		return nil, errInternal("Gagal menyimpan konfigurasi")
	}
	return &KonfigurasiResp{
		SettingID:      setting.SettingID,
		BankID:         setting.BankID,
		NamaBank:       bank.NamaBank,
		PorsiBSU:       setting.PorsiBSU,
		PorsiBSI:       setting.PorsiBSI,
		PorsiTransport: setting.PorsiTransport,
		TotalPorsi:     total,
	}, nil
}

func (s *DistribusiSisaService) UpdateKonfigurasi(bankID string, req UpdateKonfigurasiReq) (*KonfigurasiResp, error) {
	bank, err := s.repo.GetBankByID(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank != models.BSI {
		return nil, errForbidden("Hanya bank sampah bertipe BSI yang dapat mengatur konfigurasi ini")
	}

	if req.PorsiBSU == nil && req.PorsiBSI == nil && req.PorsiTransport == nil {
		return nil, errBadRequest("Minimal satu field porsi harus diisi")
	}

	setting, err := s.repo.GetSetting(bankID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errNotFound("Konfigurasi belum diatur, gunakan POST untuk membuat konfigurasi baru")
		}
		return nil, errInternal("Gagal mengambil konfigurasi")
	}

	if req.PorsiBSU != nil {
		setting.PorsiBSU = *req.PorsiBSU
	}
	if req.PorsiBSI != nil {
		setting.PorsiBSI = *req.PorsiBSI
	}
	if req.PorsiTransport != nil {
		setting.PorsiTransport = *req.PorsiTransport
	}

	total := setting.PorsiBSU + setting.PorsiBSI + setting.PorsiTransport
	if total < 99.99 || total > 100.01 {
		return nil, errBadRequest("Total porsi harus berjumlah 100%")
	}

	if err := s.repo.SaveSetting(setting); err != nil {
		return nil, errInternal("Gagal menyimpan konfigurasi")
	}
	return &KonfigurasiResp{
		SettingID:      setting.SettingID,
		BankID:         setting.BankID,
		NamaBank:       bank.NamaBank,
		PorsiBSU:       setting.PorsiBSU,
		PorsiBSI:       setting.PorsiBSI,
		PorsiTransport: setting.PorsiTransport,
		TotalPorsi:     total,
	}, nil
}

func (s *DistribusiSisaService) PreviewDistribusiSisa(bagiHasilID string) (*PreviewDistribusiResp, error) {
	bagiHasil, err := s.repo.GetBagiHasilWithBank(bagiHasilID)
	if err != nil {
		return nil, errNotFound("Data bagi hasil tidak ditemukan")
	}
	if bagiHasil.Bank == nil || bagiHasil.Bank.JenisBank != models.BSI {
		return nil, errBadRequest("Distribusi sisa hanya berlaku untuk bagi hasil BSI")
	}
	bsiID := *bagiHasil.BankID

	if existing, found := s.repo.FindDistribusiByBagiHasil(bagiHasilID); found {
		return nil, &ServiceError{
			Code:    400,
			Message: "Sisa bagi hasil ini sudah pernah didistribusikan",
			Data:    map[string]interface{}{"distribusi_id": existing.DistribusiID},
		}
	}

	totalSisa := bagiHasil.SisaBagiHasil
	if totalSisa <= 0 {
		return nil, errBadRequest("Tidak ada sisa bagi hasil yang perlu didistribusikan")
	}

	setting, err := s.repo.GetSetting(bsiID)
	if err != nil {
		return nil, errBadRequest("Konfigurasi porsi distribusi sisa belum diatur untuk bank ini")
	}

	porsiBSU := setting.PorsiBSU / 100
	porsiBSI := setting.PorsiBSI / 100
	porsiTransport := setting.PorsiTransport / 100

	daftarBSU, err := s.repo.GetBSUList(s.db, bsiID)
	if err != nil {
		return nil, errInternal("Gagal mengambil daftar BSU")
	}
	bsuNamaMap := make(map[string]string)
	for _, bsu := range daftarBSU {
		bsuNamaMap[bsu.BankID] = bsu.NamaBank
	}

	kontribusiRows, err := s.repo.GetKontribusiPerBSU(s.db, bagiHasilID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data kontribusi BSU")
	}
	kontribusiMap := make(map[string]float64)
	var totalNilaiBSU float64
	for _, row := range kontribusiRows {
		kontribusiMap[row.BankID] = row.NilaiDipakai
		totalNilaiBSU += row.NilaiDipakai
	}

	transportSplitRows, err := s.repo.GetTransportSplitPerBSU(s.db, bagiHasilID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data transport split BSU")
	}
	transportSplitMap := make(map[string]repositories.TransportSplitRow)
	for _, row := range transportSplitRows {
		transportSplitMap[row.BankID] = row
	}

	penerimaBSU := make([]BSUPreview, 0)
	var totalPorsiBSI float64
	var totalTransportBSI float64

	for _, bsu := range daftarBSU {
		kontribusi, ada := kontribusiMap[bsu.BankID]
		if !ada || kontribusi == 0 || totalNilaiBSU == 0 {
			continue
		}

		persentaseKontribusi := kontribusi / totalNilaiBSU
		bagianPokokBSU := totalSisa * porsiBSU * persentaseKontribusi
		bagianPokokBSI := totalSisa * porsiBSI * persentaseKontribusi
		bagianTransport := totalSisa * porsiTransport * persentaseKontribusi

		totalPorsiBSI += bagianPokokBSI

		split := transportSplitMap[bsu.BankID]
		totalQty := split.QtyMandiri + split.QtyNonMandiri
		var persenMandiri float64
		if totalQty > 0 {
			persenMandiri = split.QtyMandiri / totalQty
		}
		transportasiBSU := bagianTransport * persenMandiri
		totalTransportBSI += bagianTransport * (1 - persenMandiri)

		penerimaBSU = append(penerimaBSU, BSUPreview{
			BankID:           bsu.BankID,
			NamaBank:         bsuNamaMap[bsu.BankID],
			PersenKontribusi: persentaseKontribusi * 100,
			Pokok:            bagianPokokBSU,
			Transportasi:     transportasiBSU,
			Nominal:          bagianPokokBSU + transportasiBSU,
		})
	}

	return &PreviewDistribusiResp{
		BagiHasilID: bagiHasilID,
		TotalSisa:   totalSisa,
		Satuan:      bagiHasil.SatuanBagiHasil,
		NominalBSI:  totalPorsiBSI + totalTransportBSI,
		PenerimaBSU: penerimaBSU,
	}, nil
}

func (s *DistribusiSisaService) Submit(bagiHasilID string, req SubmitDistribusiReq) (*SubmitDistribusiResp, error) {
	bagiHasil, err := s.repo.GetBagiHasilWithBank(bagiHasilID)
	if err != nil {
		return nil, errNotFound("Data bagi hasil tidak ditemukan")
	}
	if bagiHasil.Bank == nil || bagiHasil.Bank.JenisBank != models.BSI {
		return nil, errBadRequest("Distribusi sisa hanya berlaku untuk bagi hasil BSI")
	}
	bsiID := *bagiHasil.BankID

	if existing, found := s.repo.FindDistribusiByBagiHasil(bagiHasilID); found {
		return nil, &ServiceError{
			Code:    400,
			Message: "Sisa bagi hasil ini sudah pernah didistribusikan",
			Data:    map[string]interface{}{"distribusi_id": existing.DistribusiID},
		}
	}

	totalSisa := bagiHasil.SisaBagiHasil
	if totalSisa <= 0 {
		return nil, errBadRequest("Tidak ada sisa bagi hasil yang perlu didistribusikan")
	}

	setting, err := s.repo.GetSetting(bsiID)
	if err != nil {
		return nil, errBadRequest("Konfigurasi porsi distribusi sisa belum diatur untuk bank ini")
	}

	porsiBSU := setting.PorsiBSU / 100
	porsiBSI := setting.PorsiBSI / 100
	porsiTransport := setting.PorsiTransport / 100

	var namaRewardTarget models.RewardEnum
	if bagiHasil.SatuanBagiHasil == models.SatuanRewardEnumRp {
		namaRewardTarget = models.RewardEnumUang
	} else {
		namaRewardTarget = models.RewardEnumSembako
	}
	rewardDistribusi, err := s.repo.GetRewardByNama(namaRewardTarget)
	if err != nil {
		return nil, errInternal("Gagal menentukan jenis reward untuk distribusi")
	}

	daftarBSU, err := s.repo.GetBSUList(s.db, bsiID)
	if err != nil {
		return nil, errInternal("Gagal mengambil daftar BSU")
	}

	kontribusiRows, err := s.repo.GetKontribusiPerBSU(s.db, bagiHasilID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data kontribusi BSU")
	}
	kontribusiPerBSU := make(map[string]float64)
	var totalNilaiBSU float64
	for _, row := range kontribusiRows {
		kontribusiPerBSU[row.BankID] = row.NilaiDipakai
		totalNilaiBSU += row.NilaiDipakai
	}

	transportSplitRows, err := s.repo.GetTransportSplitPerBSU(s.db, bagiHasilID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data transport split BSU")
	}
	transportSplitMap := make(map[string]repositories.TransportSplitRow)
	for _, row := range transportSplitRows {
		transportSplitMap[row.BankID] = row
	}

	var result *SubmitDistribusiResp

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		distribusiID := utils.GenerateID("DS")

		if err := s.repo.CreateDistribusiSisa(tx, &models.DistribusiSisa{
			DistribusiID: distribusiID,
			BagiHasilID:  bagiHasilID,
			TotalSisa:    totalSisa,
			SatuanTotal:  string(bagiHasil.SatuanBagiHasil),
			CreatedAt:    now,
			CreatedBy:    req.AdminID,
		}); err != nil {
			return errInternal("Gagal menyimpan data distribusi")
		}

		var ringkasanBSUList []RingkasanBSU
		var totalPorsiBSI float64
		var totalTransportBSI float64
		var transportDetailBSI models.TransportDetail

		for _, bsu := range daftarBSU {
			kontribusi, ada := kontribusiPerBSU[bsu.BankID]
			if !ada || kontribusi == 0 || totalNilaiBSU == 0 {
				continue
			}

			persentaseKontribusi := kontribusi / totalNilaiBSU
			bagianPokokBSU := totalSisa * porsiBSU * persentaseKontribusi
			bagianPokokBSI := totalSisa * porsiBSI * persentaseKontribusi
			bagianTransport := totalSisa * porsiTransport * persentaseKontribusi

			totalPorsiBSI += bagianPokokBSI

			// Transport split: proportional to qty mandiri vs non-mandiri from pengangkutan
			split := transportSplitMap[bsu.BankID]
			totalQty := split.QtyMandiri + split.QtyNonMandiri
			var persenMandiri float64
			if totalQty > 0 {
				persenMandiri = split.QtyMandiri / totalQty
			}
			transportasiBSU := bagianTransport * persenMandiri
			transportBSIdariIni := bagianTransport * (1 - persenMandiri)
			totalTransportBSI += transportBSIdariIni
			nominalBSU := bagianPokokBSU + transportasiBSU

			if transportBSIdariIni > 0 {
				transportDetailBSI = append(transportDetailBSI, models.TransportDetailItem{
					BsuID:     bsu.BankID,
					NamaBsu:   bsu.NamaBank,
					Transport: transportBSIdariIni,
				})
			}

			penerimaSisaID := utils.GenerateID("PDS")
			if err := s.repo.CreatePenerimaSisa(tx, &models.PenerimaDistribusiSisa{
				PenerimaSisaID:  penerimaSisaID,
				DistribusiID:    distribusiID,
				BankID:          bsu.BankID,
				JenisPenerimaan: models.JenisTerimaBagiHasilBSU,
				NominalDiterima: nominalBSU,
				Porsi:           bagianPokokBSU,
				Transportasi:    transportasiBSU,
				SatuanNominal:   string(bagiHasil.SatuanBagiHasil),
			}); err != nil {
				return errInternal("Gagal menyimpan penerima BSU: " + bsu.BankID)
			}

			bsuIDCopy := bsu.BankID
			if err := s.repo.UpdateSaldoRekening(tx, &bsuIDCopy, nil,
				rewardDistribusi.RewardID, *rewardDistribusi,
				nominalBSU, models.EntitasBankSampah,
				req.AdminID, now); err != nil {
				return errInternal("Gagal update saldo BSU " + bsu.BankID + ": " + err.Error())
			}

			if err := s.repo.SetPenerimaSisaID(tx, bagiHasilID, bsu.BankID, penerimaSisaID); err != nil {
				return errInternal("Gagal update penerima sisa BSU: " + bsu.BankID)
			}

			ringkasanBSUList = append(ringkasanBSUList, RingkasanBSU{
				BankID:         bsu.BankID,
				NamaBank:       bsu.NamaBank,
				Nominal:        nominalBSU,
				PenerimaSisaID: penerimaSisaID,
			})
		}

		totalNominalBSI := totalPorsiBSI + totalTransportBSI
		if err := s.repo.CreatePenerimaSisa(tx, &models.PenerimaDistribusiSisa{
			PenerimaSisaID:  utils.GenerateID("PDS"),
			DistribusiID:    distribusiID,
			BankID:          bsiID,
			JenisPenerimaan: models.JenisTerimaBagiHasilBSI,
			NominalDiterima: totalNominalBSI,
			Porsi:           totalPorsiBSI,
			Transportasi:    totalTransportBSI,
			SatuanNominal:   string(bagiHasil.SatuanBagiHasil),
			TransportDetail: transportDetailBSI,
		}); err != nil {
			return errInternal("Gagal menyimpan penerima BSI")
		}

		if err := s.repo.UpdateSaldoRekening(tx, &bsiID, nil,
			rewardDistribusi.RewardID, *rewardDistribusi,
			totalNominalBSI, models.EntitasBankSampah,
			req.AdminID, now); err != nil {
			return errInternal("Gagal update saldo BSI: " + err.Error())
		}

		if ringkasanBSUList == nil {
			ringkasanBSUList = []RingkasanBSU{}
		}

		result = &SubmitDistribusiResp{
			DistribusiID: distribusiID,
			BagiHasilID:  bagiHasilID,
			TotalSisa:    totalSisa,
			Satuan:       bagiHasil.SatuanBagiHasil,
			NominalBSI:   totalNominalBSI,
			PenerimaBSU:  ringkasanBSUList,
			NamaBSI:      bagiHasil.Bank.NamaBank,
		}
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}

func (s *DistribusiSisaService) GetDetailDistribusiSisa(distribusiID string) (*DetailDistribusiResp, error) {
	distribusi, err := s.repo.GetDistribusiSisa(distribusiID)
	if err != nil {
		return nil, errNotFound("Data distribusi tidak ditemukan")
	}

	namaAdmin := s.repo.GetAdminNama(distribusi.CreatedBy)

	penerimaBSI, err := s.repo.GetPenerimaBSI(distribusiID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data penerima BSI")
	}

	penerimaBSUList, err := s.repo.GetListPenerimaBSU(distribusiID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data penerima BSU")
	}

	bsuResp := make([]PenerimaBSUDetailResp, 0, len(penerimaBSUList))
	for _, p := range penerimaBSUList {
		bsuResp = append(bsuResp, PenerimaBSUDetailResp{
			PenerimaSisaID:  p.PenerimaSisaID,
			BankID:          p.BankID,
			NamaBank:        p.BankSampah.NamaBank,
			NominalDiterima: p.NominalDiterima,
			Porsi:           p.Porsi,
			Transportasi:    p.Transportasi,
			SatuanNominal:   p.SatuanNominal,
		})
	}

	return &DetailDistribusiResp{
		DistribusiID: distribusi.DistribusiID,
		BagiHasilID:  distribusi.BagiHasilID,
		TotalSisa:    distribusi.TotalSisa,
		Satuan:       distribusi.SatuanTotal,
		CreatedAt:    distribusi.CreatedAt,
		CreatedBy:    namaAdmin,
		PenerimaBSI: PenerimaBSIDetailResp{
			PenerimaSisaID:  penerimaBSI.PenerimaSisaID,
			BankID:          penerimaBSI.BankID,
			NamaBank:        penerimaBSI.BankSampah.NamaBank,
			NominalDiterima: penerimaBSI.NominalDiterima,
			Porsi:           penerimaBSI.Porsi,
			Transportasi:    penerimaBSI.Transportasi,
			SatuanNominal:   penerimaBSI.SatuanNominal,
			TransportDetail: penerimaBSI.TransportDetail,
		},
		PenerimaBSU: bsuResp,
	}, nil
}

func (s *DistribusiSisaService) ListBagiHasilBank(bankID string, startDate, endDate time.Time) (*ListBagiHasilBankResp, error) {
	bank, err := s.repo.GetBankByIDAndJenis(bankID, []models.JenisBank{models.BSU, models.BSI})
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	var jenisPenerimaan models.JenisTerimaEnum
	if bank.JenisBank == models.BSI {
		jenisPenerimaan = models.JenisTerimaBagiHasilBSI
	} else {
		jenisPenerimaan = models.JenisTerimaBagiHasilBSU
	}

	penerima, err := s.repo.GetPenerimaByBank(bankID, jenisPenerimaan, startDate, endDate)
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat distribusi")
	}

	riwayat := make([]ListItemResp, 0, len(penerima))
	for _, p := range penerima {
		riwayat = append(riwayat, ListItemResp{
			PenerimaSisaID:    p.PenerimaSisaID,
			DistribusiID:      p.DistribusiID,
			BagiHasilID:       p.DistribusiSisa.BagiHasilID,
			NominalDiterima:   p.NominalDiterima,
			Porsi:             p.Porsi,
			Transportasi:      p.Transportasi,
			SatuanNominal:     p.SatuanNominal,
			TanggalDistribusi: p.DistribusiSisa.CreatedAt,
		})
	}

	return &ListBagiHasilBankResp{
		BankID:    bankID,
		NamaBank:  bank.NamaBank,
		JenisBank: bank.JenisBank,
		Riwayat:   riwayat,
	}, nil
}

func (s *DistribusiSisaService) DetailBagiHasilBank(penerimaSisaID string) (*DetailBagiHasilBankResp, error) {
	penerima, err := s.repo.GetPenerimaSisaWithDetails(penerimaSisaID)
	if err != nil {
		return nil, errNotFound("Data penerimaan tidak ditemukan")
	}

	perhitungan, err := s.repo.GetPerhitunganSisa(penerimaSisaID)
	if err != nil {
		return nil, errInternal("Gagal mengambil detail perhitungan")
	}

	phList := make([]PerhitunganResp, 0, len(perhitungan))
	for _, ph := range perhitungan {
		namaSampah := ""
		satuan := ""
		if ph.TabunganSampah.KatalogSampah != nil {
			namaSampah = ph.TabunganSampah.KatalogSampah.Sarok.NamaSampah
			satuan = string(ph.TabunganSampah.KatalogSampah.Sarok.Satuan)
		}
		phList = append(phList, PerhitunganResp{
			TabunganID: ph.TabunganID,
			NamaSampah: namaSampah,
			QtyDipakai: ph.QtyDipakai,
			Satuan:     satuan,
		})
	}

	return &DetailBagiHasilBankResp{
		PenerimaSisaID:    penerima.PenerimaSisaID,
		DistribusiID:      penerima.DistribusiID,
		BagiHasilID:       penerima.DistribusiSisa.BagiHasilID,
		BankID:            penerima.BankID,
		NamaBank:          penerima.BankSampah.NamaBank,
		NominalDiterima:   penerima.NominalDiterima,
		Porsi:             penerima.Porsi,
		Transportasi:      penerima.Transportasi,
		SatuanNominal:     penerima.SatuanNominal,
		TanggalDistribusi: penerima.DistribusiSisa.CreatedAt,
		PerhitunganSisa:   phList,
	}, nil
}

func (s *DistribusiSisaService) GetAdminsByBankID(bankID string) ([]models.Admin, error) {
	return s.repo.GetAdminsByBankID(bankID)
}
