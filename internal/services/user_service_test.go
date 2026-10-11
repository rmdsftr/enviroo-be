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

func newUsrSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockUserRepo, *mocks.MockMailer, services.UserService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockUserRepo)
	mailerMock := new(mocks.MockMailer)
	return dbMock, repo, mailerMock, services.NewUserService(gdb, repo, mailerMock)
}

func usrUser() *models.User {
	return &models.User{
		UserID: "1371010101010001", Nama: "Budi Santoso",
		Email: "budi@mail.com", NoWhatsapp: "081234567890",
	}
}

func usrRow() *repositories.UserRow {
	return &repositories.UserRow{
		UserID: "1371010101010001", Nama: "Budi Santoso",
		Email: "budi@mail.com", NoWhatsapp: "081234567890",
	}
}

func usrNasabah(nasabahID string, status models.StatusAkun, namaBank string) models.Nasabah {
	return models.Nasabah{
		NasabahID: nasabahID, BankID: "BSU-001", UserID: "1371010101010001",
		NomorRekening: "REK-001", StatusNasabah: status,
		JoinedAt: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.Local),
		Bank:     models.BankSampah{BankID: "BSU-001", NamaBank: namaBank},
	}
}

func usrAdmin(adminID string, role models.RoleAdmin, status models.StatusAkun, namaBank string) models.Admin {
	bankID := "BSU-001"
	a := models.Admin{
		AdminID: adminID, BankID: &bankID, UserID: "1371010101010001",
		Role: role, StatusAdmin: status,
		JoinedAt: time.Date(2026, time.February, 3, 0, 0, 0, 0, time.Local),
		Bank:     models.BankSampah{BankID: "BSU-001", NamaBank: namaBank},
	}
	if role == models.SuperAdmin {
		a.BankID = nil
	}
	return a
}

func usrTambahInput() services.TambahUserInput {
	return services.TambahUserInput{
		UserID: "1371010101010001", Nama: "Budi Santoso",
		Email: "budi@mail.com", NoWhatsapp: "081234567890",
	}
}

func usrKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func TestGetProfilUser_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("GetProfilUser", "1371010101010001").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.GetProfilUser("1371010101010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, usrKode(t, err))
}

func TestGetProfilUser_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("GetProfilUser", "1371010101010001").Return(nil, errors.New("db error"))

	hasil, err := svc.GetProfilUser("1371010101010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestGetProfilUser_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	profil := &repositories.ProfilUserRow{
		UserID: "1371010101010001", Nama: "Budi Santoso", IsAdmin: true, AdminID: "23BSU-0010001",
	}
	repo.On("GetProfilUser", "1371010101010001").Return(profil, nil)

	hasil, err := svc.GetProfilUser("1371010101010001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "Budi Santoso", hasil.Nama)
		assert.True(t, hasil.IsAdmin)
	}
}

func TestGetDetailPetugas_IDKosong(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)

	hasil, err := svc.GetDetailPetugas("")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, usrKode(t, err))
	repo.AssertNotCalled(t, "GetDetailPetugas", mock.Anything)
}

func TestGetDetailPetugas_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("GetDetailPetugas", "33BSU-0010001").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.GetDetailPetugas("33BSU-0010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, usrKode(t, err))
}

func TestGetDetailPetugas_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("GetDetailPetugas", "33BSU-0010001").Return(nil, errors.New("db error"))

	hasil, err := svc.GetDetailPetugas("33BSU-0010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestGetDetailPetugas_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	detail := &repositories.DetailPetugasRow{
		PetugasID: "33BSU-0010001", Nama: "Budi Santoso", IsNasabah: true, NasabahID: "NSB-001",
	}
	repo.On("GetDetailPetugas", "33BSU-0010001").Return(detail, nil)

	hasil, err := svc.GetDetailPetugas("33BSU-0010001")

	assert.NoError(t, err)
	assert.Equal(t, "NSB-001", hasil.NasabahID)
}

func TestTambahUser_NIKSudahTerdaftar(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(1), nil)

	hasil, err := svc.TambahUser(usrTambahInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, usrKode(t, err))
	assert.Contains(t, err.Error(), "sudah terdaftar")
	repo.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
}

func TestTambahUser_EmailSudahDipakai(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(1), nil)

	hasil, err := svc.TambahUser(usrTambahInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, usrKode(t, err))
	assert.Contains(t, err.Error(), "Email")
}

func TestTambahUser_CekNIKError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), errors.New("db error"))

	hasil, err := svc.TambahUser(usrTambahInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestTambahUser_CekEmailError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(0), errors.New("db error"))

	hasil, err := svc.TambahUser(usrTambahInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestTambahUser_CreateError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(0), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.TambahUser(usrTambahInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestTambahUser_BacaUlangGagal(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(0), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindUserRow", "1371010101010001").Return(nil, errors.New("db error"))

	hasil, err := svc.TambahUser(usrTambahInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestTambahUser_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(0), nil)

	var dibuat *models.User
	repo.On("CreateUser", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { dibuat = args.Get(1).(*models.User) }).Return(nil)
	repo.On("FindUserRow", "1371010101010001").Return(usrRow(), nil)

	hasil, err := svc.TambahUser(usrTambahInput())

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	if assert.NotNil(t, dibuat) {
		assert.Equal(t, "1371010101010001", dibuat.UserID, "user_id = NIK")
		assert.Equal(t, "Budi Santoso", dibuat.Nama)
		assert.Empty(t, dibuat.Password, "password baru di-set saat aktivasi")
	}
}

func TestUpdateProfil_SemuaFieldKosong(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)

	hasil, err := svc.UpdateProfil("1371010101010001", services.UpdateProfilUserInput{})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, usrKode(t, err))
	assert.Contains(t, err.Error(), "Minimal satu field")
	repo.AssertNotCalled(t, "FindUser", mock.Anything)
}

func TestUpdateProfil_UserTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.UpdateProfil("1371010101010001", services.UpdateProfilUserInput{Nama: "Budi Baru"})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, usrKode(t, err))
}

func TestUpdateProfil_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(nil, errors.New("db error"))

	hasil, err := svc.UpdateProfil("1371010101010001", services.UpdateProfilUserInput{Nama: "Budi Baru"})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestUpdateProfil_HanyaFieldTerisiYangDiupdate(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)

	var fields map[string]interface{}
	repo.On("UpdateUser", "1371010101010001", mock.Anything).
		Run(func(args mock.Arguments) { fields = args.Get(1).(map[string]interface{}) }).Return(nil)
	repo.On("FindUserRow", "1371010101010001").Return(usrRow(), nil)

	_, err := svc.UpdateProfil("1371010101010001", services.UpdateProfilUserInput{Nama: "Budi Baru"})

	assert.NoError(t, err)
	assert.Equal(t, "Budi Baru", fields["nama"])
	assert.NotContains(t, fields, "no_whatsapp")
	assert.NotContains(t, fields, "photo_url")
}

func TestUpdateProfil_FotoBaruDipakai(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)

	var fields map[string]interface{}
	repo.On("UpdateUser", "1371010101010001", mock.Anything).
		Run(func(args mock.Arguments) { fields = args.Get(1).(map[string]interface{}) }).Return(nil)
	repo.On("FindUserRow", "1371010101010001").Return(usrRow(), nil)

	_, err := svc.UpdateProfil("1371010101010001", services.UpdateProfilUserInput{
		AdaFileFoto: true, FotoURL: "https://cdn/baru.png",
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://cdn/baru.png", fields["photo_url"])
}

func TestUpdateProfil_HapusFotoJadiKosong(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)

	var fields map[string]interface{}
	repo.On("UpdateUser", "1371010101010001", mock.Anything).
		Run(func(args mock.Arguments) { fields = args.Get(1).(map[string]interface{}) }).Return(nil)
	repo.On("FindUserRow", "1371010101010001").Return(usrRow(), nil)

	_, err := svc.UpdateProfil("1371010101010001", services.UpdateProfilUserInput{HapusFoto: true})

	assert.NoError(t, err)
	assert.Equal(t, "", fields["photo_url"])
}

func TestUpdateProfil_FileBaruMenangDariHapusFoto(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)

	var fields map[string]interface{}
	repo.On("UpdateUser", "1371010101010001", mock.Anything).
		Run(func(args mock.Arguments) { fields = args.Get(1).(map[string]interface{}) }).Return(nil)
	repo.On("FindUserRow", "1371010101010001").Return(usrRow(), nil)

	_, err := svc.UpdateProfil("1371010101010001", services.UpdateProfilUserInput{
		AdaFileFoto: true, FotoURL: "https://cdn/baru.png", HapusFoto: true,
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://cdn/baru.png", fields["photo_url"])
}

func TestUpdateProfil_UpdateError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)
	repo.On("UpdateUser", "1371010101010001", mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.UpdateProfil("1371010101010001", services.UpdateProfilUserInput{Nama: "Budi Baru"})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestUpdateProfil_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)

	var fields map[string]interface{}
	repo.On("UpdateUser", "1371010101010001", mock.Anything).
		Run(func(args mock.Arguments) { fields = args.Get(1).(map[string]interface{}) }).Return(nil)
	repo.On("FindUserRow", "1371010101010001").Return(usrRow(), nil)

	hasil, err := svc.UpdateProfil("1371010101010001", services.UpdateProfilUserInput{
		Nama: "Budi Baru", NoWhatsapp: "089999999999",
	})

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	assert.Equal(t, "Budi Baru", fields["nama"])
	assert.Equal(t, "089999999999", fields["no_whatsapp"])
}

func TestListNonAdmin_RepoError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("ListUserBukanAdmin").Return(nil, errors.New("db error"))

	hasil, err := svc.ListNonAdmin()

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestListNonAdmin_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("ListUserBukanAdmin").Return([]repositories.UserRow{*usrRow()}, nil)

	hasil, err := svc.ListNonAdmin()

	assert.NoError(t, err)
	assert.Len(t, hasil, 1)
}

func TestListNonNasabah_RepoError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("ListUserBukanNasabah").Return(nil, errors.New("db error"))

	hasil, err := svc.ListNonNasabah()

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestListNonNasabah_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("ListUserBukanNasabah").Return([]repositories.UserRow{}, nil)

	hasil, err := svc.ListNonNasabah()

	assert.NoError(t, err)
	assert.Len(t, hasil, 0)
}

func TestGetUserRingkas_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUserRingkas", "1371010101010001").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.GetUserRingkas("1371010101010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, usrKode(t, err))
}

func TestGetUserRingkas_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUserRingkas", "1371010101010001").Return(nil, errors.New("db error"))

	hasil, err := svc.GetUserRingkas("1371010101010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestGetUserRingkas_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUserRingkas", "1371010101010001").
		Return(&repositories.UserRingkasRow{UserID: "1371010101010001", Nama: "Budi Santoso"}, nil)

	hasil, err := svc.GetUserRingkas("1371010101010001")

	assert.NoError(t, err)
	assert.Equal(t, "Budi Santoso", hasil.Nama)
}

func TestUpdateFCMTokenUser_RepoError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("UpdateFCMToken", "1371010101010001", "fcm-abc").Return(errors.New("db error"))

	err := svc.UpdateFCMToken("1371010101010001", "fcm-abc")

	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestUpdateFCMTokenUser_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("UpdateFCMToken", "1371010101010001", "fcm-abc").Return(nil)

	err := svc.UpdateFCMToken("1371010101010001", "fcm-abc")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGetLogAkun_UserTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(nil, errors.New("not found"))

	hasil, err := svc.GetLogAkun("1371010101010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, usrKode(t, err))
}

func TestGetLogAkun_ErrorNasabah(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)
	repo.On("ListNasabahWithBank", "1371010101010001").Return(nil, errors.New("db error"))

	hasil, err := svc.GetLogAkun("1371010101010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
	assert.Contains(t, err.Error(), "nasabah")
}

func TestGetLogAkun_ErrorAdmin(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)
	repo.On("ListNasabahWithBank", "1371010101010001").Return([]models.Nasabah{}, nil)
	repo.On("ListAdminWithBank", "1371010101010001").Return(nil, errors.New("db error"))

	hasil, err := svc.GetLogAkun("1371010101010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
	assert.Contains(t, err.Error(), "admin")
}

func TestGetLogAkun_SliceKosongNonNil(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)
	repo.On("ListNasabahWithBank", "1371010101010001").Return([]models.Nasabah{}, nil)
	repo.On("ListAdminWithBank", "1371010101010001").Return([]models.Admin{}, nil)

	hasil, err := svc.GetLogAkun("1371010101010001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.NotNil(t, hasil.AkunNasabah, "slice kosong harus non-nil supaya JSON-nya []")
		assert.NotNil(t, hasil.AkunAdmin)
		assert.Equal(t, "Budi Santoso", hasil.Nama)
	}
}

func TestGetLogAkun_Mapping(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)
	repo.On("ListNasabahWithBank", "1371010101010001").
		Return([]models.Nasabah{usrNasabah("NSB-001", models.Nonaktif, "BSU Mawar")}, nil)
	repo.On("ListAdminWithBank", "1371010101010001").
		Return([]models.Admin{usrAdmin("33BSU-0010001", models.PetugasBSU, models.Aktif, "BSU Mawar")}, nil)

	hasil, err := svc.GetLogAkun("1371010101010001")

	assert.NoError(t, err)
	if assert.Len(t, hasil.AkunNasabah, 1) {
		assert.Equal(t, "NSB-001", hasil.AkunNasabah[0].NasabahID)
		assert.Equal(t, models.Nonaktif, hasil.AkunNasabah[0].Status)
		assert.Equal(t, "BSU Mawar", hasil.AkunNasabah[0].Afiliasi)
	}
	if assert.Len(t, hasil.AkunAdmin, 1) {
		assert.Equal(t, "Petugas BSU", hasil.AkunAdmin[0].Role, "role dipetakan ke label lengkap")
		assert.Equal(t, "BSU Mawar", hasil.AkunAdmin[0].Afiliasi)
	}
}

func TestGetLogAkun_AfiliasiSuperadmin(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)
	repo.On("ListNasabahWithBank", "1371010101010001").Return([]models.Nasabah{}, nil)
	repo.On("ListAdminWithBank", "1371010101010001").
		Return([]models.Admin{usrAdmin("100001", models.SuperAdmin, models.Aktif, "")}, nil)

	hasil, err := svc.GetLogAkun("1371010101010001")

	assert.NoError(t, err)
	if assert.Len(t, hasil.AkunAdmin, 1) {
		assert.Equal(t, "Super Admin", hasil.AkunAdmin[0].Role)
		assert.Equal(t, "Dinas Lingkungan Hidup Kota Padang", hasil.AkunAdmin[0].Afiliasi)
	}
}

func TestGetLogAkun_AdminTanpaBankID(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	rusak := usrAdmin("23BSU-0010001", models.AdminBSU, models.Aktif, "BSU Mawar")
	rusak.BankID = nil
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)
	repo.On("ListNasabahWithBank", "1371010101010001").Return([]models.Nasabah{}, nil)
	repo.On("ListAdminWithBank", "1371010101010001").Return([]models.Admin{rusak}, nil)

	hasil, err := svc.GetLogAkun("1371010101010001")

	assert.NoError(t, err)
	if assert.Len(t, hasil.AkunAdmin, 1) {
		assert.Equal(t, "", hasil.AkunAdmin[0].Afiliasi)
	}
}

func TestGetDetailUser_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUserWithAkun", "1371010101010001").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.GetDetailUser("1371010101010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, usrKode(t, err))
}

func TestGetDetailUser_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUserWithAkun", "1371010101010001").Return(nil, errors.New("db error"))

	hasil, err := svc.GetDetailUser("1371010101010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestGetDetailUser_Mapping(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	user := usrUser()
	user.PhotoURL = "https://cdn/foto.png"
	user.Nasabahs = []models.Nasabah{usrNasabah("NSB-001", models.Aktif, "BSU Mawar")}
	user.Admins = []models.Admin{usrAdmin("21BSI-0010001", models.AdminBSI, models.Aktif, "BSI Utama")}
	repo.On("FindUserWithAkun", "1371010101010001").Return(user, nil)

	hasil, err := svc.GetDetailUser("1371010101010001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "budi@mail.com", hasil.Email)
		assert.Equal(t, "https://cdn/foto.png", hasil.PhotoURL)
		if assert.Len(t, hasil.AkunNasabah, 1) {
			assert.Equal(t, "REK-001", hasil.AkunNasabah[0].NomorRekening,
				"detail user membawa nomor rekening, beda dengan log akun")
			assert.Equal(t, "BSU Mawar", hasil.AkunNasabah[0].Afiliasi)
		}
		if assert.Len(t, hasil.AkunAdmin, 1) {
			assert.Equal(t, "Admin BSI", hasil.AkunAdmin[0].Role)
		}
	}
}

func TestGetDetailUser_TanpaAkunSliceKosong(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUserWithAkun", "1371010101010001").Return(usrUser(), nil)

	hasil, err := svc.GetDetailUser("1371010101010001")

	assert.NoError(t, err)
	assert.NotNil(t, hasil.AkunNasabah)
	assert.NotNil(t, hasil.AkunAdmin)
	assert.Len(t, hasil.AkunNasabah, 0)
}

func TestHapusUser_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(nil, gorm.ErrRecordNotFound)

	err := svc.HapusUser("1371010101010001")

	assert.Equal(t, http.StatusNotFound, usrKode(t, err))
}

func TestHapusUser_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(nil, errors.New("db error"))

	err := svc.HapusUser("1371010101010001")

	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestHapusUser_PernahJadiNasabah(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)
	repo.On("CountNasabahByUser", "1371010101010001").Return(int64(1), nil)

	err := svc.HapusUser("1371010101010001")

	assert.Equal(t, http.StatusConflict, usrKode(t, err))
	assert.Contains(t, err.Error(), "nasabah")
	repo.AssertNotCalled(t, "DeleteUser", mock.Anything)
}

func TestHapusUser_PernahJadiAdmin(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)
	repo.On("CountNasabahByUser", "1371010101010001").Return(int64(0), nil)
	repo.On("CountAdminByUser", "1371010101010001").Return(int64(1), nil)

	err := svc.HapusUser("1371010101010001")

	assert.Equal(t, http.StatusConflict, usrKode(t, err))
	assert.Contains(t, err.Error(), "admin")
	repo.AssertNotCalled(t, "DeleteUser", mock.Anything)
}

func TestHapusUser_CekNasabahError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)
	repo.On("CountNasabahByUser", "1371010101010001").Return(int64(0), errors.New("db error"))

	err := svc.HapusUser("1371010101010001")

	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestHapusUser_CekAdminError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindUser", "1371010101010001").Return(usrUser(), nil)
	repo.On("CountNasabahByUser", "1371010101010001").Return(int64(0), nil)
	repo.On("CountAdminByUser", "1371010101010001").Return(int64(0), errors.New("db error"))

	err := svc.HapusUser("1371010101010001")

	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestHapusUser_DeleteError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	user := usrUser()
	repo.On("FindUser", "1371010101010001").Return(user, nil)
	repo.On("CountNasabahByUser", "1371010101010001").Return(int64(0), nil)
	repo.On("CountAdminByUser", "1371010101010001").Return(int64(0), nil)
	repo.On("DeleteUser", user).Return(errors.New("db error"))

	err := svc.HapusUser("1371010101010001")

	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestHapusUser_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	user := usrUser()
	repo.On("FindUser", "1371010101010001").Return(user, nil)
	repo.On("CountNasabahByUser", "1371010101010001").Return(int64(0), nil)
	repo.On("CountAdminByUser", "1371010101010001").Return(int64(0), nil)
	repo.On("DeleteUser", user).Return(nil)

	err := svc.HapusUser("1371010101010001")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestListSemuaUser_CountError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUser").Return(int64(0), errors.New("db error"))

	items, meta, err := svc.ListSemuaUser(1, 25)

	assert.Nil(t, items)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestListSemuaUser_ListError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUser").Return(int64(10), nil)
	repo.On("ListUserWithAkun", 25, 0).Return(nil, errors.New("db error"))

	items, meta, err := svc.ListSemuaUser(1, 25)

	assert.Nil(t, items)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestListSemuaUser_NormalisasiPageLimit(t *testing.T) {
	kasus := []struct {
		nama                        string
		page, limit                 int
		wantLimit, wantOffset, want int
	}{
		{"page nol jadi satu", 0, 25, 25, 0, 1},
		{"page negatif jadi satu", -5, 25, 25, 0, 1},
		{"limit nol jadi default", 1, 0, 25, 0, 1},
		{"limit di atas seratus dipotong", 1, 500, 100, 0, 1},
		{"halaman kedua", 2, 10, 10, 10, 2},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			_, repo, _, svc := newUsrSvc(t)
			repo.On("CountUser").Return(int64(0), nil)
			repo.On("ListUserWithAkun", k.wantLimit, k.wantOffset).Return([]models.User{}, nil)

			_, meta, err := svc.ListSemuaUser(k.page, k.limit)

			assert.NoError(t, err)
			if assert.NotNil(t, meta) {
				assert.Equal(t, k.want, meta.Page)
				assert.Equal(t, k.wantLimit, meta.Limit)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestListSemuaUser_TotalPages(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUser").Return(int64(31), nil)
	repo.On("ListUserWithAkun", 10, 0).Return([]models.User{}, nil)

	_, meta, err := svc.ListSemuaUser(1, 10)

	assert.NoError(t, err)
	assert.Equal(t, int64(31), meta.TotalItems)
	assert.Equal(t, 4, meta.TotalPages, "31 item / 10 per halaman dibulatkan ke atas")
}

func TestListSemuaUser_RoleBelumTerdaftar(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	user := *usrUser()
	user.Nasabahs = []models.Nasabah{usrNasabah("NSB-001", models.Nonaktif, "BSU Mawar")}
	user.Admins = []models.Admin{usrAdmin("33BSU-0010001", models.PetugasBSU, models.Pending, "BSU Mawar")}
	repo.On("CountUser").Return(int64(1), nil)
	repo.On("ListUserWithAkun", 25, 0).Return([]models.User{user}, nil)

	items, _, err := svc.ListSemuaUser(1, 25)

	assert.NoError(t, err)
	if assert.Len(t, items, 1) {
		assert.Equal(t, []string{"Belum Terdaftar"}, items[0].Roles)
	}
}

func TestListSemuaUser_RoleDiambilSatuPerJenis(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	user := *usrUser()
	user.Nasabahs = []models.Nasabah{
		usrNasabah("NSB-001", models.Aktif, "BSU Mawar"),
		usrNasabah("NSB-002", models.Aktif, "BSU Melati"),
	}
	user.Admins = []models.Admin{
		usrAdmin("33BSU-0010001", models.PetugasBSU, models.Nonaktif, "BSU Mawar"),
		usrAdmin("21BSI-0010001", models.AdminBSI, models.Aktif, "BSI Utama"),
	}
	repo.On("CountUser").Return(int64(1), nil)
	repo.On("ListUserWithAkun", 25, 0).Return([]models.User{user}, nil)

	items, _, err := svc.ListSemuaUser(1, 25)

	assert.NoError(t, err)
	if assert.Len(t, items, 1) {
		assert.Equal(t, []string{"Nasabah", "Admin BSI"}, items[0].Roles,
			"akun nonaktif dilewati, tiap jenis cuma diambil satu")
		assert.Equal(t, "budi@mail.com", items[0].Email)
	}
}

func TestListSemuaUser_SliceKosongNonNil(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUser").Return(int64(0), nil)
	repo.On("ListUserWithAkun", 25, 0).Return([]models.User{}, nil)

	items, meta, err := svc.ListSemuaUser(1, 25)

	assert.NoError(t, err)
	assert.NotNil(t, items)
	assert.Len(t, items, 0)
	assert.Equal(t, 0, meta.TotalPages)
}

func TestGetActiveAdmin_RepoError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindActiveAdmin", "21BSI-0010001").Return(nil, errors.New("db error"))

	hasil, err := svc.GetActiveAdmin("21BSI-0010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestGetActiveAdmin_BarisKosongJadiNotFound(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindActiveAdmin", "21BSI-0010001").Return(&repositories.ActiveAdminRow{}, nil)

	hasil, err := svc.GetActiveAdmin("21BSI-0010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, usrKode(t, err))
}

func TestGetActiveAdmin_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindActiveAdmin", "21BSI-0010001").
		Return(&repositories.ActiveAdminRow{BankID: "BSI-001", NamaBank: "BSI Utama", JenisBank: models.BSI}, nil)

	hasil, err := svc.GetActiveAdmin("21BSI-0010001")

	assert.NoError(t, err)
	assert.Equal(t, "BSI Utama", hasil.NamaBank)
}

func TestGetActivePetugas_RepoError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindActivePetugas", "33BSU-0010001").Return(nil, errors.New("db error"))

	hasil, err := svc.GetActivePetugas("33BSU-0010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestGetActivePetugas_BarisKosongJadiNotFound(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindActivePetugas", "33BSU-0010001").Return(&repositories.ActivePetugasRow{}, nil)

	hasil, err := svc.GetActivePetugas("33BSU-0010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, usrKode(t, err))
}

func TestGetActivePetugas_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("FindActivePetugas", "33BSU-0010001").
		Return(&repositories.ActivePetugasRow{AdminID: "33BSU-0010001", Nama: "Budi Santoso"}, nil)

	hasil, err := svc.GetActivePetugas("33BSU-0010001")

	assert.NoError(t, err)
	assert.Equal(t, "Budi Santoso", hasil.Nama)
}

func usrSuperadminInput() services.TambahSuperadminInput {
	return services.TambahSuperadminInput{
		UserID: "1371010101010001", Nama: "budi santoso",
		Email: "budi@mail.com", NoWhatsapp: "081234567890",
	}
}

func expectTxSuperadmin(m sqlmock.Sqlmock, commit bool) {
	m.ExpectBegin()
	m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"admin_id"}))
	m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"aktivasi_id"}))
	m.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	if commit {
		m.ExpectCommit()
	} else {
		m.ExpectRollback()
	}
}

func TestTambahSuperadmin_NIKSudahTerdaftar(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(1), nil)

	hasil, err := svc.TambahSuperadmin(usrSuperadminInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, usrKode(t, err))
}

func TestTambahSuperadmin_EmailSudahDipakai(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(1), nil)

	hasil, err := svc.TambahSuperadmin(usrSuperadminInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, usrKode(t, err))
}

func TestTambahSuperadmin_CreateUserError(t *testing.T) {
	dbMock, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(0), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.TambahSuperadmin(usrSuperadminInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
	assert.Contains(t, err.Error(), "membuat user")
}

func TestTambahSuperadmin_GenerateAdminIDGagal(t *testing.T) {
	dbMock, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(0), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))
	dbMock.ExpectRollback()

	hasil, err := svc.TambahSuperadmin(usrSuperadminInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
	assert.Contains(t, err.Error(), "generate admin ID")
	repo.AssertNotCalled(t, "CreateAdmin", mock.Anything, mock.Anything)
}

func TestTambahSuperadmin_CreateAdminError(t *testing.T) {
	dbMock, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(0), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateAdmin", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"admin_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.TambahSuperadmin(usrSuperadminInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
	assert.Contains(t, err.Error(), "membuat admin")
}

func TestTambahSuperadmin_AktivasiGagal(t *testing.T) {
	dbMock, repo, _, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(0), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateAdmin", mock.Anything, mock.Anything).Return(nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"admin_id"}))
	dbMock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))
	dbMock.ExpectRollback()

	hasil, err := svc.TambahSuperadmin(usrSuperadminInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
	assert.Contains(t, err.Error(), "aktivasi akun")
}

func TestTambahSuperadmin_EmailGagalMembatalkanTransaksi(t *testing.T) {
	dbMock, repo, mailerMock, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(0), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateAdmin", mock.Anything, mock.Anything).Return(nil)
	mailerMock.On("SendEmail", mock.Anything).Return(errors.New("smtp down"))
	expectTxSuperadmin(dbMock, false)

	hasil, err := svc.TambahSuperadmin(usrSuperadminInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
	assert.Contains(t, err.Error(), "mengirim email")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTambahSuperadmin_Sukses(t *testing.T) {
	dbMock, repo, mailerMock, svc := newUsrSvc(t)
	repo.On("CountUserByID", "1371010101010001").Return(int64(0), nil)
	repo.On("CountUserByEmail", "budi@mail.com").Return(int64(0), nil)

	var userDibuat *models.User
	repo.On("CreateUser", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { userDibuat = args.Get(1).(*models.User) }).Return(nil)

	var adminDibuat *models.Admin
	repo.On("CreateAdmin", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { adminDibuat = args.Get(1).(*models.Admin) }).Return(nil)
	mailerMock.On("SendEmail", mock.Anything).Return(nil)
	expectTxSuperadmin(dbMock, true)

	hasil, err := svc.TambahSuperadmin(usrSuperadminInput())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "Budi Santoso", hasil.Nama, "nama dinormalisasi ke Title Case")
		assert.Equal(t, models.SuperAdmin, hasil.Role)
		assert.NotEmpty(t, hasil.AdminID)
	}
	if assert.NotNil(t, adminDibuat) {
		assert.Nil(t, adminDibuat.BankID, "superadmin tidak terikat bank sampah")
		assert.Equal(t, models.Pending, adminDibuat.StatusAdmin, "pending sampai OTP diaktivasi")
		assert.Equal(t, models.SuperAdmin, adminDibuat.Role)
	}
	if assert.NotNil(t, userDibuat) {
		assert.Equal(t, "Budi Santoso", userDibuat.Nama)
		assert.Empty(t, userDibuat.Password)
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestListSuperadmin_RepoError(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("ListSuperadmin").Return(nil, errors.New("db error"))

	hasil, err := svc.ListSuperadmin()

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, usrKode(t, err))
}

func TestListSuperadmin_Sukses(t *testing.T) {
	_, repo, _, svc := newUsrSvc(t)
	repo.On("ListSuperadmin").Return([]repositories.SuperadminRow{
		{AdminID: "100001", Nama: "Budi Santoso", StatusAdmin: string(models.Aktif)},
	}, nil)

	hasil, err := svc.ListSuperadmin()

	assert.NoError(t, err)
	assert.Len(t, hasil, 1)
	assert.Equal(t, "100001", hasil[0].AdminID)
}
