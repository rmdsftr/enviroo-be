package services_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func newBrgSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockBarangRepo, services.BarangService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBarangRepo)
	notif := new(mocks.MockNotifSvc)
	svc := services.NewBarangService(gdb, repo, notif)

	repo.On("FindAdminTargets", mock.Anything).Return(nil, errors.New("tidak distub")).Maybe()

	return dbMock, repo, svc
}

func brgStubNotifGoroutine(repo *mocks.MockBarangRepo) {
	repo.On("FindBank", mock.Anything).Return(nil, errors.New("tidak distub")).Maybe()
}

func brgBank(bankID string, jenis models.JenisBank) *models.BankSampah {
	return &models.BankSampah{BankID: bankID, NamaBank: "Bank " + bankID, JenisBank: jenis, IsActive: true}
}

func brgBSU(parentID string) *models.BankSampah {
	bank := brgBank("BSU-001", models.BSU)
	if parentID != "" {
		bank.ParentBankID = ptrStr(parentID)
	}
	return bank
}

func brgKatalog(produkID, bankID string, nilaiPoin float64) *models.KatalogBarang {
	bID := bankID
	return &models.KatalogBarang{
		ProdukID:     produkID,
		BankID:       &bID,
		BarangID:     7,
		NilaiPoin:    nilaiPoin,
		PhotoURL:     ptrStr("https://cdn/tumbler.png"),
		MasterBarang: models.Barang{BarangID: 7, NamaBarang: "Tumbler"},
	}
}

func brgStok(produkID, bankID string, stok float64) *models.StokBarang {
	return &models.StokBarang{ProdukID: produkID, BankID: bankID, Stok: stok}
}

func brgAkses(bankID string, role models.RoleAdmin) services.AksesBarangInput {
	return services.AksesBarangInput{ActorBankID: bankID, ActorRole: role}
}

func brgAksesBSU(bankID string) services.AksesBarangInput {
	return brgAkses(bankID, models.AdminBSU)
}

func brgDisba(status models.StatusDisbaEnum) *models.DistribusiBarangBsu {
	return &models.DistribusiBarangBsu{
		DisbaID:          "BSI-001260923001",
		BsiID:            "BSI-001",
		BsuID:            "BSU-001",
		AdminBsiID:       "ADM-BSI-01",
		StatusDistribusi: status,
	}
}

func brgDetail(produkID string, item float64) models.DetailDistribusiBarang {
	return models.DetailDistribusiBarang{
		DisbaID:      "BSI-001260923001",
		ProdukID:     produkID,
		Item:         item,
		Poin:         100,
		SubtotalPoin: 100 * item,
	}
}

func brgKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func brgTambahInput() services.TambahBarangInput {
	return services.TambahBarangInput{
		BankID:     "BSI-001",
		NamaBarang: "Tumbler",
		NilaiPoin:  100,
		StokAwal:   25,
		CreatedBy:  "ADM-BSI-01",
		FotoURL:    ptrStr("https://cdn/tumbler.png"),
	}
}

func TestListMasterBarang_RepoError(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("ListMasterBarang", "tumbler").Return(nil, errors.New("db error"))

	hasil, err := svc.ListMasterBarang("tumbler")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
}

func TestListMasterBarang_Sukses(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("ListMasterBarang", "").Return([]models.Barang{
		{BarangID: 7, NamaBarang: "Tumbler"},
		{BarangID: 8, NamaBarang: "Tote Bag"},
	}, nil)

	hasil, err := svc.ListMasterBarang("")

	assert.NoError(t, err)
	assert.Len(t, hasil, 2)
}

func TestTambahBarang_BarangIDTidakValid(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("AdaRewardBarangAktif", "BSI-001").Return(true, nil)
	repo.On("FindMasterBarang", 99).Return(nil, errors.New("not found"))

	in := brgTambahInput()
	in.BarangID = 99
	hasil, err := svc.TambahBarang(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, brgKode(t, err))
	assert.Contains(t, err.Error(), "barang_id")
}

func TestTambahBarang_BarangIDDipakaiApaAdanya(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("AdaRewardBarangAktif", "BSI-001").Return(true, nil)
	repo.On("FindMasterBarang", 7).Return(&models.Barang{BarangID: 7, NamaBarang: "Tumbler"}, nil)
	repo.On("CreateKatalogBarang", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	in := brgTambahInput()
	in.BarangID = 7
	hasil, err := svc.TambahBarang(in)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	repo.AssertNotCalled(t, "FindOrCreateMasterBarang", mock.Anything, mock.Anything)
}

func TestTambahBarang_TanpaBarangIDBuatMaster(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("AdaRewardBarangAktif", "BSI-001").Return(true, nil)
	repo.On("FindOrCreateMasterBarang", mock.Anything, "Tumbler").
		Return(&models.Barang{BarangID: 12, NamaBarang: "Tumbler"}, nil)
	repo.On("CreateKatalogBarang", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	hasil, err := svc.TambahBarang(brgTambahInput())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, 12, hasil.Katalog.BarangID)
		assert.Equal(t, "BSI-001012", hasil.Katalog.ProdukID, "produk_id = bank + barang_id 3 digit")
	}
}

func TestTambahBarang_MasterBarangError(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("AdaRewardBarangAktif", "BSI-001").Return(true, nil)
	repo.On("FindOrCreateMasterBarang", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	hasil, err := svc.TambahBarang(brgTambahInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
	assert.Contains(t, err.Error(), "master barang")
}

func TestTambahBarang_GenerateIDGagal(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("AdaRewardBarangAktif", "BSI-001").Return(true, nil)
	repo.On("FindMasterBarang", 1500).Return(&models.Barang{BarangID: 1500}, nil)

	in := brgTambahInput()
	in.BarangID = 1500
	hasil, err := svc.TambahBarang(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
	assert.Contains(t, err.Error(), "generate barang ID")
	repo.AssertNotCalled(t, "CreateKatalogBarang", mock.Anything, mock.Anything)
}

func TestTambahBarang_DuplikatJadiConflict(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("AdaRewardBarangAktif", "BSI-001").Return(true, nil)
	repo.On("FindMasterBarang", 7).Return(&models.Barang{BarangID: 7}, nil)
	repo.On("CreateKatalogBarang", mock.Anything, mock.Anything).Return(gorm.ErrDuplicatedKey)
	expectTxRollback(dbMock)

	in := brgTambahInput()
	in.BarangID = 7
	hasil, err := svc.TambahBarang(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, brgKode(t, err))
	assert.Contains(t, err.Error(), "sudah ada")
	repo.AssertNotCalled(t, "CreateStok", mock.Anything, mock.Anything)
}

func TestTambahBarang_CreateStokErrorRollback(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("AdaRewardBarangAktif", "BSI-001").Return(true, nil)
	repo.On("FindMasterBarang", 7).Return(&models.Barang{BarangID: 7}, nil)
	repo.On("CreateKatalogBarang", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	in := brgTambahInput()
	in.BarangID = 7
	hasil, err := svc.TambahBarang(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
	assert.Contains(t, err.Error(), "stok awal")
}

func TestTambahBarang_Sukses(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("AdaRewardBarangAktif", "BSI-001").Return(true, nil)
	repo.On("FindMasterBarang", 7).Return(&models.Barang{BarangID: 7, NamaBarang: "Tumbler"}, nil)
	repo.On("CreateKatalogBarang", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	in := brgTambahInput()
	in.BarangID = 7
	hasil, err := svc.TambahBarang(in)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BSI-001007", hasil.Katalog.ProdukID)
		assert.Equal(t, float64(100), hasil.Katalog.NilaiPoin)
		if assert.NotNil(t, hasil.Katalog.BankID) {
			assert.Equal(t, "BSI-001", *hasil.Katalog.BankID)
		}
		if assert.NotNil(t, hasil.Katalog.CreatedBy) && assert.NotNil(t, hasil.Katalog.UpdatedBy) {
			assert.Equal(t, "ADM-BSI-01", *hasil.Katalog.CreatedBy)
			assert.Equal(t, "ADM-BSI-01", *hasil.Katalog.UpdatedBy, "saat dibuat, updated_by = created_by")
		}
		assert.Equal(t, float64(25), hasil.Stok.Stok, "stok awal ikut yang dikirim")
		assert.Equal(t, "BSI-001007", hasil.Stok.ProdukID)
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestListBarangBank_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-XXX").Return(nil, errors.New("not found"))

	rows, meta, err := svc.ListBarangBank("BSI-XXX", 1)

	assert.Nil(t, rows)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
}

func TestListBarangBank_BSUPakaiKatalogInduk(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("ListBarangBank", "BSU-001", "BSI-001", 12, 0).
		Return([]repositories.BarangBankRow{{ProdukID: "BSI-001007", Stok: 4, TotalCount: 1}}, nil)

	rows, meta, err := svc.ListBarangBank("BSU-001", 1)

	assert.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.NotNil(t, meta)
	repo.AssertExpectations(t)
}

func TestListBarangBank_BSUTanpaIndukPakaiBankSendiri(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU(""), nil)
	repo.On("ListBarangBank", "BSU-001", "BSU-001", 12, 0).Return([]repositories.BarangBankRow{}, nil)

	_, _, err := svc.ListBarangBank("BSU-001", 1)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestListBarangBank_RepoError(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListBarangBank", "BSI-001", "BSI-001", 12, 0).Return(nil, errors.New("db error"))

	rows, meta, err := svc.ListBarangBank("BSI-001", 1)

	assert.Nil(t, rows)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
}

func TestListBarangBank_NilJadiSliceKosong(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListBarangBank", "BSI-001", "BSI-001", 0, 0).
		Return([]repositories.BarangBankRow(nil), nil)

	rows, meta, err := svc.ListBarangBank("BSI-001", 0)

	assert.NoError(t, err)
	assert.NotNil(t, rows, "nil harus jadi [] supaya JSON-nya bukan null")
	assert.Len(t, rows, 0)
	assert.Nil(t, meta, "page<=0 berarti tanpa pagination")
}

func TestListBarangBank_TotalDariTotalCount(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListBarangBank", "BSI-001", "BSI-001", 12, 12).
		Return([]repositories.BarangBankRow{{ProdukID: "BSI-001007", TotalCount: 30}}, nil)

	_, meta, err := svc.ListBarangBank("BSI-001", 2)

	assert.NoError(t, err)
	if assert.NotNil(t, meta) {
		assert.Equal(t, 2, meta.Page)
		assert.Equal(t, 12, meta.Limit)
		assert.Equal(t, 30, meta.Total)
		assert.Equal(t, 3, meta.TotalPages)
	}
}

func TestListBarangBank_HalamanKosongTotalNol(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListBarangBank", "BSI-001", "BSI-001", 12, 0).Return([]repositories.BarangBankRow{}, nil)

	_, meta, err := svc.ListBarangBank("BSI-001", 1)

	assert.NoError(t, err)
	if assert.NotNil(t, meta) {
		assert.Equal(t, 0, meta.Total)
		assert.Equal(t, 0, meta.TotalPages)
	}
}

func TestRiwayatDistribusi_BankIDKosong(t *testing.T) {
	_, _, svc := newBrgSvc(t)

	rows, err := svc.RiwayatDistribusiBarang("BSI-001007", "", brgAksesBSU("BSU-001"))

	assert.Nil(t, rows)
	assert.Equal(t, http.StatusBadRequest, brgKode(t, err))
}

func TestRiwayatDistribusi_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-XXX").Return(nil, errors.New("not found"))

	rows, err := svc.RiwayatDistribusiBarang("BSI-001007", "BSU-XXX", brgAksesBSU("BSU-XXX"))

	assert.Nil(t, rows)
	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
}

func TestRiwayatDistribusi_BankLainDitolak(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)

	rows, err := svc.RiwayatDistribusiBarang("BSI-001007", "BSU-001", brgAksesBSU("BSU-999"))

	assert.Nil(t, rows)
	assert.Equal(t, http.StatusForbidden, brgKode(t, err))
	repo.AssertNotCalled(t, "ListRiwayatDistribusiProduk", mock.Anything, mock.Anything, mock.Anything)
}

func TestRiwayatDistribusi_BSUSebagaiPenerima(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("ListRiwayatDistribusiProduk", "BSI-001007", "BSU-001", true).
		Return([]repositories.RiwayatDistribusiRow{{DisbaID: "D-001", Item: 5}}, nil)

	rows, err := svc.RiwayatDistribusiBarang("BSI-001007", "BSU-001", brgAksesBSU("BSU-001"))

	assert.NoError(t, err)
	assert.Len(t, rows, 1)
	repo.AssertExpectations(t)
}

func TestRiwayatDistribusi_BSISebagaiPengirim(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListRiwayatDistribusiProduk", "BSI-001007", "BSI-001", false).
		Return([]repositories.RiwayatDistribusiRow{}, nil)

	_, err := svc.RiwayatDistribusiBarang("BSI-001007", "BSI-001", brgAkses("BSI-001", models.AdminBSI))

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRiwayatDistribusi_RepoError(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListRiwayatDistribusiProduk", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("db error"))

	rows, err := svc.RiwayatDistribusiBarang("BSI-001007", "BSI-001", brgAkses("BSI-001", models.AdminBSI))

	assert.Nil(t, rows)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
}

func TestRiwayatDistribusi_NilJadiSliceKosong(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListRiwayatDistribusiProduk", mock.Anything, mock.Anything, mock.Anything).
		Return([]repositories.RiwayatDistribusiRow(nil), nil)

	rows, err := svc.RiwayatDistribusiBarang("BSI-001007", "BSI-001", brgAkses("BSI-001", models.AdminBSI))

	assert.NoError(t, err)
	assert.NotNil(t, rows)
	assert.Len(t, rows, 0)
}

func brgEditInput(bankID string) services.EditBarangInput {
	return services.EditBarangInput{
		UpdatedBy: "ADM-BSI-01",
		Akses:     brgAkses(bankID, models.AdminBSM),
	}
}

func TestEditBarang_TidakDitemukan(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(nil, errors.New("not found"))

	hasil, err := svc.EditBarang("BSI-001007", brgEditInput("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
}

func TestEditBarang_BankIDNil(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	katalog := brgKatalog("BSI-001007", "BSI-001", 100)
	katalog.BankID = nil
	repo.On("FindKatalogBarang", "BSI-001007").Return(katalog, nil)

	hasil, err := svc.EditBarang("BSI-001007", brgEditInput("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
	assert.Contains(t, err.Error(), "bank pemilik")
}

func TestEditBarang_BankLainDitolak(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)

	hasil, err := svc.EditBarang("BSI-001007", brgEditInput("BSM-009"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, brgKode(t, err))
	repo.AssertNotCalled(t, "UpdateKatalogBarang", mock.Anything, mock.Anything, mock.Anything)
}

func TestEditBarang_NilaiPoinTidakDikirimTidakDiupdate(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	katalog := brgKatalog("BSI-001007", "BSI-001", 100)
	repo.On("FindKatalogBarang", "BSI-001007").Return(katalog, nil)

	var fields map[string]any
	repo.On("UpdateKatalogBarang", mock.Anything, "BSI-001007", mock.Anything).
		Run(func(args mock.Arguments) { fields = args.Get(2).(map[string]any) }).Return(nil)
	expectTx(dbMock, 0)

	_, err := svc.EditBarang("BSI-001007", brgEditInput("BSI-001"))

	assert.NoError(t, err)
	assert.NotContains(t, fields, "nilai_poin")
	assert.Equal(t, "ADM-BSI-01", fields["updated_by"])
}

func TestEditBarang_NilaiPoinDikirimIkutDiupdate(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	katalog := brgKatalog("BSI-001007", "BSI-001", 100)
	repo.On("FindKatalogBarang", "BSI-001007").Return(katalog, nil)

	var fields map[string]any
	repo.On("UpdateKatalogBarang", mock.Anything, "BSI-001007", mock.Anything).
		Run(func(args mock.Arguments) { fields = args.Get(2).(map[string]any) }).Return(nil)
	expectTx(dbMock, 0)

	in := brgEditInput("BSI-001")
	in.NilaiPoin = 150
	in.SetNilaiPoin = true
	_, err := svc.EditBarang("BSI-001007", in)

	assert.NoError(t, err)
	assert.Equal(t, float64(150), fields["nilai_poin"])
}

func TestEditBarang_FotoNilPertahankanLama(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	katalog := brgKatalog("BSI-001007", "BSI-001", 100)
	repo.On("FindKatalogBarang", "BSI-001007").Return(katalog, nil)

	var fields map[string]any
	repo.On("UpdateKatalogBarang", mock.Anything, "BSI-001007", mock.Anything).
		Run(func(args mock.Arguments) { fields = args.Get(2).(map[string]any) }).Return(nil)
	expectTx(dbMock, 0)

	_, err := svc.EditBarang("BSI-001007", brgEditInput("BSI-001"))

	assert.NoError(t, err)
	foto, ok := fields["photo_url"].(*string)
	if assert.True(t, ok) && assert.NotNil(t, foto) {
		assert.Equal(t, "https://cdn/tumbler.png", *foto)
	}
}

func TestEditBarang_FotoBaruDipakai(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	katalog := brgKatalog("BSI-001007", "BSI-001", 100)
	repo.On("FindKatalogBarang", "BSI-001007").Return(katalog, nil)

	var fields map[string]any
	repo.On("UpdateKatalogBarang", mock.Anything, "BSI-001007", mock.Anything).
		Run(func(args mock.Arguments) { fields = args.Get(2).(map[string]any) }).Return(nil)
	expectTx(dbMock, 0)

	in := brgEditInput("BSI-001")
	in.FotoURL = ptrStr("https://cdn/baru.png")
	_, err := svc.EditBarang("BSI-001007", in)

	assert.NoError(t, err)
	foto := fields["photo_url"].(*string)
	assert.Equal(t, "https://cdn/baru.png", *foto)
}

func TestEditBarang_TambahStokDanStokBarenganDitolak(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)

	in := brgEditInput("BSI-001")
	in.TambahStok = 5
	in.Stok = 99
	in.SetStok = true
	hasil, err := svc.EditBarang("BSI-001007", in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, brgKode(t, err))
	assert.Contains(t, err.Error(), "tidak boleh diisi bersamaan")
	repo.AssertNotCalled(t, "UpdateKatalogBarang", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "TambahStok", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "SetStok", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestEditBarang_TambahStokNolDenganStokTetapKoreksi(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("UpdateKatalogBarang", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("SetStok", mock.Anything, "BSI-001007", "BSI-001", float64(40)).Return(nil)
	expectTx(dbMock, 0)

	in := brgEditInput("BSI-001")
	in.TambahStok = 0
	in.Stok = 40
	in.SetStok = true
	_, err := svc.EditBarang("BSI-001007", in)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "TambahStok", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestEditBarang_SetStok(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("UpdateKatalogBarang", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("SetStok", mock.Anything, "BSI-001007", "BSI-001", float64(40)).Return(nil)
	expectTx(dbMock, 0)

	in := brgEditInput("BSI-001")
	in.Stok = 40
	in.SetStok = true
	_, err := svc.EditBarang("BSI-001007", in)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestEditBarang_TanpaPerubahanStok(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("UpdateKatalogBarang", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	_, err := svc.EditBarang("BSI-001007", brgEditInput("BSI-001"))

	assert.NoError(t, err)
	repo.AssertNotCalled(t, "SetStok", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "TambahStok", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestEditBarang_UpdateKatalogError(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("UpdateKatalogBarang", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.EditBarang("BSI-001007", brgEditInput("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
	assert.Contains(t, err.Error(), "mengupdate katalog")
}

func TestEditBarang_TambahStokErrorRollback(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("UpdateKatalogBarang", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("TambahStok", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	in := brgEditInput("BSI-001")
	in.TambahStok = 5
	hasil, err := svc.EditBarang("BSI-001007", in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
	assert.Contains(t, err.Error(), "menambah stok")
}

func TestEditBarang_DibacaUlangSetelahUpdate(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	sebelum := brgKatalog("BSI-001007", "BSI-001", 100)
	sesudah := brgKatalog("BSI-001007", "BSI-001", 150)
	repo.On("FindKatalogBarang", "BSI-001007").Return(sebelum, nil).Once()
	repo.On("UpdateKatalogBarang", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("FindKatalogBarang", "BSI-001007").Return(sesudah, nil).Once()
	expectTx(dbMock, 0)

	in := brgEditInput("BSI-001")
	in.NilaiPoin = 150
	in.SetNilaiPoin = true
	hasil, err := svc.EditBarang("BSI-001007", in)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, float64(150), hasil.NilaiPoin, "nilai final diambil dari pembacaan ulang")
	}
}

func TestEditBarang_BacaUlangGagal(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil).Once()
	repo.On("UpdateKatalogBarang", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("FindKatalogBarang", "BSI-001007").Return(nil, errors.New("db error")).Once()
	expectTx(dbMock, 0)

	hasil, err := svc.EditBarang("BSI-001007", brgEditInput("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
	assert.Contains(t, err.Error(), "setelah update")
}

func brgHapusInput(bankID string) services.HapusBarangInput {
	return services.HapusBarangInput{ProdukID: "BSI-001007", Akses: brgAkses(bankID, models.AdminBSM)}
}

func TestHapusBarang_TidakDitemukan(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(nil, errors.New("not found"))

	err := svc.HapusBarang(brgHapusInput("BSI-001"))

	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
}

func TestHapusBarang_BankIDNil(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	katalog := brgKatalog("BSI-001007", "BSI-001", 100)
	katalog.BankID = nil
	repo.On("FindKatalogBarang", "BSI-001007").Return(katalog, nil)

	err := svc.HapusBarang(brgHapusInput("BSI-001"))

	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
}

func TestHapusBarang_BankLainDitolak(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)

	err := svc.HapusBarang(brgHapusInput("BSM-009"))

	assert.Equal(t, http.StatusForbidden, brgKode(t, err))
	repo.AssertNotCalled(t, "CountPemakaianBarang", mock.Anything)
}

func TestHapusBarang_CekPemakaianError(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("CountPemakaianBarang", "BSI-001007").Return(int64(0), errors.New("db error"))

	err := svc.HapusBarang(brgHapusInput("BSI-001"))

	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
}

func TestHapusBarang_MasihDipakai(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("CountPemakaianBarang", "BSI-001007").Return(int64(2), nil)

	err := svc.HapusBarang(brgHapusInput("BSI-001"))

	assert.Equal(t, http.StatusConflict, brgKode(t, err))
	assert.Contains(t, err.Error(), "sudah digunakan")
	repo.AssertNotCalled(t, "DeleteKatalogBarang", mock.Anything, mock.Anything)
}

func TestHapusBarang_StokDihapusSebelumKatalog(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("CountPemakaianBarang", "BSI-001007").Return(int64(0), nil)

	urutan := []string{}
	repo.On("DeleteStokByProduk", mock.Anything, "BSI-001007").
		Run(func(mock.Arguments) { urutan = append(urutan, "stok") }).Return(nil)
	repo.On("DeleteKatalogBarang", mock.Anything, "BSI-001007").
		Run(func(mock.Arguments) { urutan = append(urutan, "katalog") }).Return(nil)
	expectTx(dbMock, 0)

	err := svc.HapusBarang(brgHapusInput("BSI-001"))

	assert.NoError(t, err)
	assert.Equal(t, []string{"stok", "katalog"}, urutan)
}

func TestHapusBarang_DeleteStokErrorRollback(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("CountPemakaianBarang", "BSI-001007").Return(int64(0), nil)
	repo.On("DeleteStokByProduk", mock.Anything, "BSI-001007").Return(errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.HapusBarang(brgHapusInput("BSI-001"))

	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
	repo.AssertNotCalled(t, "DeleteKatalogBarang", mock.Anything, mock.Anything)
}

func TestHapusBarang_KatalogHilangDiTengahJalan(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindKatalogBarang", "BSI-001007").Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("CountPemakaianBarang", "BSI-001007").Return(int64(0), nil)
	repo.On("DeleteStokByProduk", mock.Anything, "BSI-001007").Return(nil)
	repo.On("DeleteKatalogBarang", mock.Anything, "BSI-001007").Return(gorm.ErrRecordNotFound)
	expectTxRollback(dbMock)

	err := svc.HapusBarang(brgHapusInput("BSI-001"))

	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
}

func brgBuatSesiInput() services.BuatSesiDistribusiInput {
	return services.BuatSesiDistribusiInput{
		BSIID:      "BSI-001",
		BSUID:      "BSU-001",
		AdminBSIID: "ADM-BSI-01",
		Items: []services.ItemDistribusiInput{
			{ProdukID: "BSI-001007", StokKirim: 5},
		},
		Akses: brgAkses("BSI-001", models.AdminBSI),
	}
}

func TestBuatSesiDistribusi_BukanBSIPengirim(t *testing.T) {
	_, _, svc := newBrgSvc(t)

	in := brgBuatSesiInput()
	in.Akses = brgAksesBSU("BSU-009")
	disbaID, err := svc.BuatSesiDistribusi(in)

	assert.Empty(t, disbaID)
	assert.Equal(t, http.StatusForbidden, brgKode(t, err))
}

func TestBuatSesiDistribusi_BSUTidakDitemukan(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(nil, errors.New("not found"))

	disbaID, err := svc.BuatSesiDistribusi(brgBuatSesiInput())

	assert.Empty(t, disbaID)
	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
}

func TestBuatSesiDistribusi_TujuanBukanBSU(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBank("BSU-001", models.BSI), nil)

	disbaID, err := svc.BuatSesiDistribusi(brgBuatSesiInput())

	assert.Empty(t, disbaID)
	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
}

func TestBuatSesiDistribusi_BSUBukanCabangBSITersebut(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-999"), nil)

	disbaID, err := svc.BuatSesiDistribusi(brgBuatSesiInput())

	assert.Empty(t, disbaID)
	assert.Equal(t, http.StatusBadRequest, brgKode(t, err))
	assert.Contains(t, err.Error(), "bukan cabang")
}

func TestBuatSesiDistribusi_BSUTanpaInduk(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU(""), nil)

	disbaID, err := svc.BuatSesiDistribusi(brgBuatSesiInput())

	assert.Empty(t, disbaID)
	assert.Equal(t, http.StatusBadRequest, brgKode(t, err))
}

func TestBuatSesiDistribusi_ProdukBukanMilikBSI(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("CreateDistribusi", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindKatalogBarangByBank", mock.Anything, "BSI-001007", "BSI-001").
		Return(nil, errors.New("not found"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"disba_id"}))
	dbMock.ExpectRollback()

	disbaID, err := svc.BuatSesiDistribusi(brgBuatSesiInput())

	assert.Empty(t, disbaID)
	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
	assert.Contains(t, err.Error(), "BSI-001007")
}

func TestBuatSesiDistribusi_CreateDistribusiError(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("CreateDistribusi", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"disba_id"}))
	dbMock.ExpectRollback()

	disbaID, err := svc.BuatSesiDistribusi(brgBuatSesiInput())

	assert.Empty(t, disbaID)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
}

func TestBuatSesiDistribusi_GenerateIDGagal(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnError(errors.New("lock timeout"))
	dbMock.ExpectRollback()

	disbaID, err := svc.BuatSesiDistribusi(brgBuatSesiInput())

	assert.Empty(t, disbaID)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
	assert.Contains(t, err.Error(), "disba ID")
}

func TestBuatSesiDistribusi_TotalDihitungDariKatalog(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("CreateDistribusi", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindKatalogBarangByBank", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("FindKatalogBarangByBank", mock.Anything, "BSI-001008", "BSI-001").
		Return(brgKatalog("BSI-001008", "BSI-001", 250), nil)

	var detail []models.DetailDistribusiBarang
	repo.On("CreateDetailDistribusi", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			detail = append(detail, *args.Get(1).(*models.DetailDistribusiBarang))
		}).Return(nil)

	var totals map[string]any
	repo.On("UpdateDistribusi", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { totals = args.Get(2).(map[string]any) }).Return(nil)
	expectTx(dbMock, 1)

	in := brgBuatSesiInput()
	in.Items = []services.ItemDistribusiInput{
		{ProdukID: "BSI-001007", StokKirim: 5},
		{ProdukID: "BSI-001008", StokKirim: 2},
	}
	disbaID, err := svc.BuatSesiDistribusi(in)

	assert.NoError(t, err)
	assert.NotEmpty(t, disbaID)
	assert.Equal(t, float64(7), totals["total_item"])
	assert.Equal(t, float64(1000), totals["total_poin"])
	if assert.Len(t, detail, 2) {
		assert.Equal(t, float64(100), detail[0].Poin)
		assert.Equal(t, float64(500), detail[0].SubtotalPoin)
		assert.Equal(t, float64(500), detail[1].SubtotalPoin)
	}
}

func TestBuatSesiDistribusi_Sukses(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)

	var disba *models.DistribusiBarangBsu
	repo.On("CreateDistribusi", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { disba = args.Get(1).(*models.DistribusiBarangBsu) }).Return(nil)
	repo.On("FindKatalogBarangByBank", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("CreateDetailDistribusi", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateDistribusi", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	disbaID, err := svc.BuatSesiDistribusi(brgBuatSesiInput())

	assert.NoError(t, err)
	assert.NotEmpty(t, disbaID)
	if assert.NotNil(t, disba) {
		assert.Equal(t, models.DisbaPending, disba.StatusDistribusi, "sesi baru selalu pending sampai BSU scan QR")
		assert.Equal(t, "BSI-001", disba.BsiID)
		assert.Equal(t, "BSU-001", disba.BsuID)
		assert.Equal(t, "ADM-BSI-01", disba.AdminBsiID)
		assert.Equal(t, disbaID, disba.DisbaID)
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func brgPreviewInput() services.PreviewDistribusiInput {
	return services.PreviewDistribusiInput{
		BSIID: "BSI-001",
		BSUID: "BSU-001",
		Items: []services.ItemDistribusiInput{{ProdukID: "BSI-001007", StokKirim: 5}},
		Akses: brgAkses("BSI-001", models.AdminBSI),
	}
}

func TestPreviewDistribusi_ItemsKosong(t *testing.T) {
	_, _, svc := newBrgSvc(t)

	in := brgPreviewInput()
	in.Items = nil
	previews, validasi, err := svc.PreviewDistribusi(in)

	assert.Nil(t, previews)
	assert.Nil(t, validasi)
	assert.Equal(t, http.StatusBadRequest, brgKode(t, err))
}

func TestPreviewDistribusi_BukanBSIPengirim(t *testing.T) {
	_, _, svc := newBrgSvc(t)

	in := brgPreviewInput()
	in.Akses = brgAksesBSU("BSU-009")
	_, _, err := svc.PreviewDistribusi(in)

	assert.Equal(t, http.StatusForbidden, brgKode(t, err))
}

func TestPreviewDistribusi_BSUBukanCabang(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-999"), nil)

	_, _, err := svc.PreviewDistribusi(brgPreviewInput())

	assert.Equal(t, http.StatusBadRequest, brgKode(t, err))
}

func TestPreviewDistribusi_ProdukTidakAdaJadiPesanValidasi(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("FindKatalogBarangWithMaster", "BSI-001007", "BSI-001").Return(nil, errors.New("not found"))

	previews, validasi, err := svc.PreviewDistribusi(brgPreviewInput())

	assert.NoError(t, err)
	assert.Len(t, previews, 0)
	if assert.Len(t, validasi, 1) {
		assert.Contains(t, validasi[0], "BSI-001007")
		assert.Contains(t, validasi[0], "tidak ditemukan di BSI")
	}
}

func TestPreviewDistribusi_StokBSITidakCukup(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("FindKatalogBarangWithMaster", "BSI-001007", "BSI-001").
		Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("FindStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 2), nil)

	previews, validasi, err := svc.PreviewDistribusi(brgPreviewInput())

	assert.NoError(t, err)
	assert.Len(t, previews, 0)
	if assert.Len(t, validasi, 1) {
		assert.Contains(t, validasi[0], "Tumbler", "pesan menyebut nama barangnya")
		assert.Contains(t, validasi[0], "tidak mencukupi")
	}
}

func TestPreviewDistribusi_StokBSITidakAda(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("FindKatalogBarangWithMaster", "BSI-001007", "BSI-001").
		Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("FindStok", mock.Anything, "BSI-001007", "BSI-001").Return(nil, gorm.ErrRecordNotFound)

	previews, validasi, err := svc.PreviewDistribusi(brgPreviewInput())

	assert.NoError(t, err)
	assert.Len(t, previews, 0)
	assert.Len(t, validasi, 1)
}

func TestPreviewDistribusi_BSUItemBaru(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("FindKatalogBarangWithMaster", "BSI-001007", "BSI-001").
		Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("FindStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 20), nil)
	repo.On("FindStok", mock.Anything, "BSI-001007", "BSU-001").Return(nil, gorm.ErrRecordNotFound)

	previews, validasi, err := svc.PreviewDistribusi(brgPreviewInput())

	assert.NoError(t, err)
	assert.Len(t, validasi, 0)
	if assert.Len(t, previews, 1) {
		p := previews[0]
		assert.True(t, p.BSUItemBaru)
		assert.Equal(t, float64(20), p.StokBSISebelum)
		assert.Equal(t, float64(15), p.StokBSISesudah)
		assert.Equal(t, float64(0), p.StokBSUSebelum)
		assert.Equal(t, float64(5), p.StokBSUSesudah)
		assert.Equal(t, "Tumbler", p.NamaBarang)
		assert.Equal(t, float64(100), p.NilaiPoin)
	}
}

func TestPreviewDistribusi_BSUSudahPunyaStok(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("FindKatalogBarangWithMaster", "BSI-001007", "BSI-001").
		Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("FindStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 20), nil)
	repo.On("FindStok", mock.Anything, "BSI-001007", "BSU-001").
		Return(brgStok("BSI-001007", "BSU-001", 3), nil)

	previews, _, err := svc.PreviewDistribusi(brgPreviewInput())

	assert.NoError(t, err)
	if assert.Len(t, previews, 1) {
		assert.False(t, previews[0].BSUItemBaru)
		assert.Equal(t, float64(3), previews[0].StokBSUSebelum)
		assert.Equal(t, float64(8), previews[0].StokBSUSesudah)
	}
}

func TestPreviewDistribusi_CekStokBSUError(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("FindKatalogBarangWithMaster", "BSI-001007", "BSI-001").
		Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("FindStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 20), nil)
	repo.On("FindStok", mock.Anything, "BSI-001007", "BSU-001").Return(nil, errors.New("db error"))

	previews, validasi, err := svc.PreviewDistribusi(brgPreviewInput())

	assert.NoError(t, err)
	assert.Len(t, previews, 0)
	if assert.Len(t, validasi, 1) {
		assert.Contains(t, validasi[0], "stok BSU")
	}
}

func TestPreviewDistribusi_CampuranValidDanInvalid(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("FindKatalogBarangWithMaster", "BSI-001007", "BSI-001").
		Return(brgKatalog("BSI-001007", "BSI-001", 100), nil)
	repo.On("FindStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 20), nil)
	repo.On("FindStok", mock.Anything, "BSI-001007", "BSU-001").
		Return(brgStok("BSI-001007", "BSU-001", 3), nil)
	repo.On("FindKatalogBarangWithMaster", "BSI-001999", "BSI-001").Return(nil, errors.New("not found"))

	in := brgPreviewInput()
	in.Items = []services.ItemDistribusiInput{
		{ProdukID: "BSI-001007", StokKirim: 5},
		{ProdukID: "BSI-001999", StokKirim: 1},
	}
	previews, validasi, err := svc.PreviewDistribusi(in)

	assert.NoError(t, err)
	assert.Len(t, previews, 1, "item valid tetap masuk preview")
	assert.Len(t, validasi, 1, "item invalid jadi pesan validasi")
}

func brgSelesaikanInput() services.SelesaikanDistribusiInput {
	return services.SelesaikanDistribusiInput{
		DisbaID:    "BSI-001260923001",
		BSUID:      "BSU-001",
		AdminBSUID: "ADM-BSU-01",
		Akses:      brgAksesBSU("BSU-001"),
	}
}

func TestSelesaikanDistribusi_SesiTidakDitemukan(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(nil, errors.New("not found"))

	err := svc.SelesaikanDistribusi(brgSelesaikanInput())

	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
}

func TestSelesaikanDistribusi_SesiSudahDiproses(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaSelesai), nil)

	err := svc.SelesaikanDistribusi(brgSelesaikanInput())

	assert.Equal(t, http.StatusBadRequest, brgKode(t, err))
	assert.Contains(t, err.Error(), "sudah diproses")
}

func TestSelesaikanDistribusi_BSUTidakSesuai(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaPending), nil)

	in := brgSelesaikanInput()
	in.BSUID = "BSU-999"
	err := svc.SelesaikanDistribusi(in)

	assert.Equal(t, http.StatusBadRequest, brgKode(t, err))
	assert.Contains(t, err.Error(), "BSU tidak sesuai")
}

func TestSelesaikanDistribusi_BukanBSUTujuan(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaPending), nil)

	in := brgSelesaikanInput()
	in.Akses = brgAksesBSU("BSU-009")
	err := svc.SelesaikanDistribusi(in)

	assert.Equal(t, http.StatusForbidden, brgKode(t, err))
}

func TestSelesaikanDistribusi_DetailError(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaPending), nil)
	repo.On("ListDetailDistribusi", "BSI-001260923001").Return(nil, errors.New("db error"))

	err := svc.SelesaikanDistribusi(brgSelesaikanInput())

	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
}

func TestSelesaikanDistribusi_TanpaItem(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaPending), nil)
	repo.On("ListDetailDistribusi", "BSI-001260923001").Return([]models.DetailDistribusiBarang{}, nil)

	err := svc.SelesaikanDistribusi(brgSelesaikanInput())

	assert.Equal(t, http.StatusBadRequest, brgKode(t, err))
	assert.Contains(t, err.Error(), "Tidak ada item")
}

func TestSelesaikanDistribusi_StokDikunciSaatEksekusi(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaPending), nil)
	repo.On("ListDetailDistribusi", "BSI-001260923001").
		Return([]models.DetailDistribusiBarang{brgDetail("BSI-001007", 5)}, nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 20), nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSU-001").
		Return(brgStok("BSI-001007", "BSU-001", 3), nil)
	repo.On("SetStok", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateStokDetail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateDistribusi", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	brgStubNotifGoroutine(repo)
	expectTx(dbMock, 0)

	err := svc.SelesaikanDistribusi(brgSelesaikanInput())

	assert.NoError(t, err)
	repo.AssertNotCalled(t, "FindStok", mock.Anything, mock.Anything, mock.Anything)
}

func TestSelesaikanDistribusi_StokBSITidakCukup(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaPending), nil)
	repo.On("ListDetailDistribusi", "BSI-001260923001").
		Return([]models.DetailDistribusiBarang{brgDetail("BSI-001007", 5)}, nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 2), nil)
	expectTxRollback(dbMock)

	err := svc.SelesaikanDistribusi(brgSelesaikanInput())

	assert.Equal(t, http.StatusConflict, brgKode(t, err))
	assert.Contains(t, err.Error(), "tidak mencukupi")
	repo.AssertNotCalled(t, "SetStok", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSelesaikanDistribusi_BuatStokBSUKalauItemBaru(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaPending), nil)
	repo.On("ListDetailDistribusi", "BSI-001260923001").
		Return([]models.DetailDistribusiBarang{brgDetail("BSI-001007", 5)}, nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 20), nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSU-001").Return(nil, gorm.ErrRecordNotFound)

	var stokBaru *models.StokBarang
	repo.On("CreateStok", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { stokBaru = args.Get(1).(*models.StokBarang) }).Return(nil)
	repo.On("SetStok", mock.Anything, "BSI-001007", "BSI-001", float64(15)).Return(nil)
	repo.On("SetStok", mock.Anything, "BSI-001007", "BSU-001", float64(5)).Return(nil)
	repo.On("UpdateStokDetail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateDistribusi", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	brgStubNotifGoroutine(repo)
	expectTx(dbMock, 0)

	err := svc.SelesaikanDistribusi(brgSelesaikanInput())

	assert.NoError(t, err)
	if assert.NotNil(t, stokBaru) {
		assert.Equal(t, "BSU-001", stokBaru.BankID)
		assert.Equal(t, float64(0), stokBaru.Stok, "dibuat 0 dulu, baru di-SetStok ke nilai akhir")
	}
	repo.AssertExpectations(t)
}

func TestSelesaikanDistribusi_CreateStokBSUError(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaPending), nil)
	repo.On("ListDetailDistribusi", "BSI-001260923001").
		Return([]models.DetailDistribusiBarang{brgDetail("BSI-001007", 5)}, nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 20), nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSU-001").Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.SelesaikanDistribusi(brgSelesaikanInput())

	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
	assert.Contains(t, err.Error(), "stok BSU")
}

func TestSelesaikanDistribusi_SetStokBSIError(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaPending), nil)
	repo.On("ListDetailDistribusi", "BSI-001260923001").
		Return([]models.DetailDistribusiBarang{brgDetail("BSI-001007", 5)}, nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 20), nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSU-001").
		Return(brgStok("BSI-001007", "BSU-001", 3), nil)
	repo.On("SetStok", mock.Anything, "BSI-001007", "BSI-001", mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.SelesaikanDistribusi(brgSelesaikanInput())

	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
	assert.Contains(t, err.Error(), "stok BSI")
}

func TestSelesaikanDistribusi_UpdateHeaderError(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaPending), nil)
	repo.On("ListDetailDistribusi", "BSI-001260923001").
		Return([]models.DetailDistribusiBarang{brgDetail("BSI-001007", 5)}, nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 20), nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSU-001").
		Return(brgStok("BSI-001007", "BSU-001", 3), nil)
	repo.On("SetStok", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateStokDetail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateDistribusi", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.SelesaikanDistribusi(brgSelesaikanInput())

	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
}

func TestSelesaikanDistribusi_Sukses(t *testing.T) {
	dbMock, repo, svc := newBrgSvc(t)
	repo.On("FindDistribusi", "BSI-001260923001").Return(brgDisba(models.DisbaPending), nil)
	repo.On("ListDetailDistribusi", "BSI-001260923001").
		Return([]models.DetailDistribusiBarang{brgDetail("BSI-001007", 5)}, nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSI-001").
		Return(brgStok("BSI-001007", "BSI-001", 20), nil)
	repo.On("LockStok", mock.Anything, "BSI-001007", "BSU-001").
		Return(brgStok("BSI-001007", "BSU-001", 3), nil)
	repo.On("SetStok", mock.Anything, "BSI-001007", "BSI-001", float64(15)).Return(nil)
	repo.On("SetStok", mock.Anything, "BSI-001007", "BSU-001", float64(8)).Return(nil)

	var snapshot map[string]any
	repo.On("UpdateStokDetail", mock.Anything, "BSI-001260923001", "BSI-001007", mock.Anything).
		Run(func(args mock.Arguments) { snapshot = args.Get(3).(map[string]any) }).Return(nil)

	var header map[string]any
	repo.On("UpdateDistribusi", mock.Anything, "BSI-001260923001", mock.Anything).
		Run(func(args mock.Arguments) { header = args.Get(2).(map[string]any) }).Return(nil)
	brgStubNotifGoroutine(repo)
	expectTx(dbMock, 0)

	err := svc.SelesaikanDistribusi(brgSelesaikanInput())

	assert.NoError(t, err)
	assert.Equal(t, float64(20), snapshot["stok_bsi_sebelum"])
	assert.Equal(t, float64(15), snapshot["stok_bsi_sesudah"])
	assert.Equal(t, float64(3), snapshot["stok_bsu_sebelum"])
	assert.Equal(t, float64(8), snapshot["stok_bsu_sesudah"])
	assert.Equal(t, models.DisbaSelesai, header["status_distribusi"])
	assert.Equal(t, "ADM-BSU-01", header["admin_bsu_id"])
	repo.AssertExpectations(t)
}

func TestListDistribusi_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-XXX").Return(nil, errors.New("not found"))

	rows, err := svc.ListDistribusi("BSI-XXX", "", "")

	assert.Nil(t, rows)
	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
}

func TestListDistribusi_BSMDitolak(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSM-001").Return(brgBank("BSM-001", models.BSM), nil)

	rows, err := svc.ListDistribusi("BSM-001", "", "")

	assert.Nil(t, rows)
	assert.Equal(t, http.StatusForbidden, brgKode(t, err))
}

func TestListDistribusi_BSIFilterKolomPengirim(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListHeaderDistribusi", "d.bsi_id", "BSI-001", (*time.Time)(nil), (*time.Time)(nil)).
		Return([]repositories.DistribusiHeaderRow{}, nil)

	_, err := svc.ListDistribusi("BSI-001", "", "")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestListDistribusi_BSUFilterKolomPenerima(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
	repo.On("ListHeaderDistribusi", "d.bsu_id", "BSU-001", mock.Anything, mock.Anything).
		Return([]repositories.DistribusiHeaderRow{}, nil)

	_, err := svc.ListDistribusi("BSU-001", "", "")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestListDistribusi_EndDateSampaiAkhirHari(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListHeaderDistribusi", "d.bsi_id", "BSI-001",
		mock.MatchedBy(func(p *time.Time) bool {
			return p != nil && p.Format("2006-01-02 15:04:05") == "2026-09-01 00:00:00"
		}),
		mock.MatchedBy(func(p *time.Time) bool {
			return p != nil && p.Format("2006-01-02 15:04:05") == "2026-09-30 23:59:59"
		})).
		Return([]repositories.DistribusiHeaderRow{}, nil)

	_, err := svc.ListDistribusi("BSI-001", "2026-09-01", "2026-09-30")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestListDistribusi_TanggalSalahFormatDiabaikan(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListHeaderDistribusi", "d.bsi_id", "BSI-001", (*time.Time)(nil), (*time.Time)(nil)).
		Return([]repositories.DistribusiHeaderRow{}, nil)

	_, err := svc.ListDistribusi("BSI-001", "01-09-2026", "bukan tanggal")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestListDistribusi_RepoError(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListHeaderDistribusi", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("db error"))

	rows, err := svc.ListDistribusi("BSI-001", "", "")

	assert.Nil(t, rows)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
}

func TestListDistribusi_NilJadiSliceKosong(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindBank", "BSI-001").Return(brgBank("BSI-001", models.BSI), nil)
	repo.On("ListHeaderDistribusi", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]repositories.DistribusiHeaderRow(nil), nil)

	rows, err := svc.ListDistribusi("BSI-001", "", "")

	assert.NoError(t, err)
	assert.NotNil(t, rows)
	assert.Len(t, rows, 0)
}

func TestDetailDistribusi_TidakDitemukan(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindHeaderDistribusi", "D-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.DetailDistribusi("D-XXX", brgAksesBSU("BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
}

func TestDetailDistribusi_HeaderKosong(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindHeaderDistribusi", "D-XXX").Return(&repositories.DistribusiHeaderRow{}, nil)

	hasil, err := svc.DetailDistribusi("D-XXX", brgAksesBSU("BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, brgKode(t, err))
}

func TestDetailDistribusi_ItemError(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindHeaderDistribusi", "BSI-001260923001").
		Return(&repositories.DistribusiHeaderRow{DisbaID: "BSI-001260923001"}, nil)
	repo.On("ListItemDistribusi", "BSI-001260923001").Return(nil, errors.New("db error"))

	hasil, err := svc.DetailDistribusi("BSI-001260923001", brgAksesBSU("BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, brgKode(t, err))
}

func TestDetailDistribusi_Sukses(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	header := &repositories.DistribusiHeaderRow{
		DisbaID: "BSI-001260923001", BsiID: "BSI-001", NamaBSI: "BSI Utama",
		BsuID: "BSU-001", NamaBSU: "BSU Mawar", TotalItem: 5, TotalPoin: 500,
		StatusDistribusi: string(models.DisbaSelesai),
	}
	items := []repositories.DistribusiItemRow{
		{ProdukID: "BSI-001007", NamaBarang: "Tumbler", Item: 5, SubtotalPoin: 500},
	}
	repo.On("FindHeaderDistribusi", "BSI-001260923001").Return(header, nil)
	repo.On("ListItemDistribusi", "BSI-001260923001").Return(items, nil)

	hasil, err := svc.DetailDistribusi("BSI-001260923001", brgAksesBSU("BSU-001"))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BSI-001260923001", hasil.Header.DisbaID)
		assert.Equal(t, "BSU Mawar", hasil.Header.NamaBSU)
		assert.Len(t, hasil.Items, 1)
	}
}

func TestDetailDistribusi_ItemNilJadiSliceKosong(t *testing.T) {
	_, repo, svc := newBrgSvc(t)
	repo.On("FindHeaderDistribusi", "BSI-001260923001").
		Return(&repositories.DistribusiHeaderRow{DisbaID: "BSI-001260923001"}, nil)
	repo.On("ListItemDistribusi", "BSI-001260923001").Return([]repositories.DistribusiItemRow(nil), nil)

	hasil, err := svc.DetailDistribusi("BSI-001260923001", brgAksesBSU("BSU-001"))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.NotNil(t, hasil.Items, "nil harus jadi [] supaya JSON-nya bukan null")
		assert.Len(t, hasil.Items, 0)
	}
}

func TestAksesBarang_SuperadminDanStaffBSILintasBank(t *testing.T) {
	for _, akses := range []services.AksesBarangInput{
		brgAkses("", models.SuperAdmin),
		brgAkses("BSI-009", models.AdminBSI),
		brgAkses("BSI-009", models.PetugasBSI),
	} {
		_, repo, svc := newBrgSvc(t)
		repo.On("FindBank", "BSU-001").Return(brgBSU("BSI-001"), nil)
		repo.On("ListRiwayatDistribusiProduk", "BSI-001007", "BSU-001", true).
			Return([]repositories.RiwayatDistribusiRow{}, nil)

		rows, err := svc.RiwayatDistribusiBarang("BSI-001007", "BSU-001", akses)

		assert.NoError(t, err)
		assert.NotNil(t, rows)
	}
}
