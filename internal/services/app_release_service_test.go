package services_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func newAppReleaseSvc() (*mocks.MockAppReleaseRepo, *mocks.MockAppReleaseStorage, services.AppReleaseService) {
	repo := new(mocks.MockAppReleaseRepo)
	storage := new(mocks.MockAppReleaseStorage)
	return repo, storage, services.NewAppReleaseService(repo, storage, "")
}

func rilisContoh(versionCode int, wajib bool) *models.AppRelease {
	return &models.AppRelease{
		ReleaseID:   versionCode,
		VersionCode: versionCode,
		VersionName: "1.0.2",
		ApkKey:      "apk/enviroo-1.0.2-3-abcd1234.apk",
		FileSize:    85_000_000,
		Changelog:   "Perbaikan bug",
		IsMandatory: wajib,
	}
}

func inputRilisValid() services.TambahRilisInput {
	return services.TambahRilisInput{
		File:        strings.NewReader("isi-apk"),
		FileName:    "app-release.apk",
		FileSize:    7,
		VersionCode: 3,
		VersionName: "1.0.2",
		Changelog:   "Perbaikan bug",
		CreatedBy:   "1234567890",
	}
}

func assertServiceErrorCode(t *testing.T, err error, code int) {
	t.Helper()
	var svcErr *services.ServiceError
	if assert.True(t, errors.As(err, &svcErr), "error harus ServiceError") {
		assert.Equal(t, code, svcErr.Code)
	}
}

func TestAppReleaseCekVersi_BelumAdaRilis(t *testing.T) {
	repo, _, svc := newAppReleaseSvc()
	repo.On("FindLatest").Return(nil, gorm.ErrRecordNotFound)

	row, err := svc.CekVersi("2")

	assert.NoError(t, err)
	assert.Nil(t, row)
}

func TestAppReleaseCekVersi_AdaRilisWajibDiAntara(t *testing.T) {
	repo, _, svc := newAppReleaseSvc()
	repo.On("FindLatest").Return(rilisContoh(5, false), nil)
	repo.On("AdaWajibDiAntara", 2, 5).Return(true, nil)

	row, err := svc.CekVersi("2")

	assert.NoError(t, err)
	if assert.NotNil(t, row) {
		assert.Equal(t, 5, row.VersionCode)
		assert.True(t, row.Wajib)
		assert.Equal(t, "https://media.enviroo.tech/apk/enviroo-1.0.2-3-abcd1234.apk", row.ApkURL)
	}
	repo.AssertExpectations(t)
}

func TestAppReleaseCekVersi_SudahVersiTerbaru(t *testing.T) {
	repo, _, svc := newAppReleaseSvc()
	repo.On("FindLatest").Return(rilisContoh(5, true), nil)

	row, err := svc.CekVersi("5")

	assert.NoError(t, err)
	if assert.NotNil(t, row) {
		assert.False(t, row.Wajib)
	}
	repo.AssertNotCalled(t, "AdaWajibDiAntara", mock.Anything, mock.Anything)
}

func TestAppReleaseCekVersi_CurrentKosongPakaiFlagTerbaru(t *testing.T) {
	repo, _, svc := newAppReleaseSvc()
	repo.On("FindLatest").Return(rilisContoh(5, true), nil)

	row, err := svc.CekVersi("")

	assert.NoError(t, err)
	if assert.NotNil(t, row) {
		assert.True(t, row.Wajib)
	}
}

func TestAppReleaseTambahRilis_VersionCodeTidakNaik(t *testing.T) {
	repo, storage, svc := newAppReleaseSvc()
	repo.On("MaxVersionCode").Return(3, nil)

	_, err := svc.TambahRilis(inputRilisValid())

	assertServiceErrorCode(t, err, http.StatusConflict)
	storage.AssertNotCalled(t, "UploadObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestAppReleaseTambahRilis_BukanAPK(t *testing.T) {
	repo, storage, svc := newAppReleaseSvc()
	in := inputRilisValid()
	in.FileName = "app-release.zip"

	_, err := svc.TambahRilis(in)

	assertServiceErrorCode(t, err, http.StatusBadRequest)
	repo.AssertNotCalled(t, "MaxVersionCode")
	storage.AssertNotCalled(t, "UploadObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestAppReleaseTambahRilis_VersionNameTidakValid(t *testing.T) {
	_, _, svc := newAppReleaseSvc()
	in := inputRilisValid()
	in.VersionName = "1.0.2+3"

	_, err := svc.TambahRilis(in)

	assertServiceErrorCode(t, err, http.StatusBadRequest)
}

func TestAppReleaseTambahRilis_Sukses(t *testing.T) {
	repo, storage, svc := newAppReleaseSvc()
	repo.On("MaxVersionCode").Return(2, nil)
	storage.On("UploadObject", mock.Anything, int64(7),
		mock.MatchedBy(func(key string) bool {
			return strings.HasPrefix(key, "apk/enviroo-1.0.2-3-") && strings.HasSuffix(key, ".apk")
		}),
		"application/vnd.android.package-archive").Return(nil)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(r *models.AppRelease) bool {
		return r.VersionCode == 3 && r.VersionName == "1.0.2" && r.CreatedBy != nil && *r.CreatedBy == "1234567890"
	})).Return(nil)

	row, err := svc.TambahRilis(inputRilisValid())

	assert.NoError(t, err)
	if assert.NotNil(t, row) {
		assert.Equal(t, 3, row.VersionCode)
		assert.True(t, strings.HasPrefix(row.ApkURL, "https://media.enviroo.tech/apk/"))
	}
	storage.AssertNotCalled(t, "DeleteObject", mock.Anything)
	repo.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func TestAppReleaseTambahRilis_InsertGagalObjekDihapus(t *testing.T) {
	repo, storage, svc := newAppReleaseSvc()
	repo.On("MaxVersionCode").Return(2, nil)
	storage.On("UploadObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db down"))
	storage.On("DeleteObject", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "apk/enviroo-1.0.2-3-")
	})).Return(nil)

	_, err := svc.TambahRilis(inputRilisValid())

	assertServiceErrorCode(t, err, http.StatusInternalServerError)
	storage.AssertExpectations(t)
}

func TestAppReleaseHapusRilis_TidakDitemukan(t *testing.T) {
	repo, storage, svc := newAppReleaseSvc()
	repo.On("FindByID", 99).Return(nil, gorm.ErrRecordNotFound)

	err := svc.HapusRilis("99")

	assertServiceErrorCode(t, err, http.StatusNotFound)
	storage.AssertNotCalled(t, "DeleteObject", mock.Anything)
}

func TestAppReleaseHapusRilis_Sukses(t *testing.T) {
	repo, storage, svc := newAppReleaseSvc()
	rilis := rilisContoh(3, false)
	repo.On("FindByID", 3).Return(rilis, nil)
	repo.On("Delete", rilis).Return(nil)
	storage.On("DeleteObject", rilis.ApkKey).Return(nil)

	err := svc.HapusRilis("3")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func TestAppReleaseUbahRilis_TanpaPerubahan(t *testing.T) {
	repo, _, svc := newAppReleaseSvc()
	repo.On("FindByID", 3).Return(rilisContoh(3, false), nil)

	_, err := svc.UbahRilis("3", services.UbahRilisInput{})

	assertServiceErrorCode(t, err, http.StatusBadRequest)
	repo.AssertNotCalled(t, "Save", mock.Anything)
}

func TestAppReleaseInfoInstallTerbaru_AdaRilis(t *testing.T) {
	repo, _, svc := newAppReleaseSvc()
	repo.On("FindLatest").Return(rilisContoh(3, false), nil)

	info := svc.InfoInstallTerbaru()

	if assert.NotNil(t, info) {
		assert.Equal(t, "https://media.enviroo.tech/apk/enviroo-1.0.2-3-abcd1234.apk", info.URL)
		assert.Equal(t, "1.0.2", info.VersionName)
	}
}

func TestAppReleaseInfoInstallTerbaru_BelumAdaRilis(t *testing.T) {
	repo, _, svc := newAppReleaseSvc()
	repo.On("FindLatest").Return(nil, gorm.ErrRecordNotFound)

	assert.Nil(t, svc.InfoInstallTerbaru())
}

func TestAppReleaseInfoInstallTerbaru_RepoErrorTetapNil(t *testing.T) {
	repo, _, svc := newAppReleaseSvc()
	repo.On("FindLatest").Return(nil, errors.New("db down"))

	assert.Nil(t, svc.InfoInstallTerbaru())
}

func TestAppReleaseInfoInstallTerbaru_PakaiLinkPermanenBilaAPIPublicURLDiset(t *testing.T) {
	repo := new(mocks.MockAppReleaseRepo)
	storage := new(mocks.MockAppReleaseStorage)
	svc := services.NewAppReleaseService(repo, storage, "https://api.enviroo.tech/")
	repo.On("FindLatest").Return(rilisContoh(3, false), nil)

	info := svc.InfoInstallTerbaru()

	if assert.NotNil(t, info) {
		assert.Equal(t, "https://api.enviroo.tech/app-version/latest", info.URL, "slash di akhir base URL dibuang")
		assert.Equal(t, "1.0.2", info.VersionName)
	}
}

func TestAppReleaseInfoInstallTerbaru_APIPublicURLDisetTapiBelumAdaRilis(t *testing.T) {
	repo := new(mocks.MockAppReleaseRepo)
	storage := new(mocks.MockAppReleaseStorage)
	svc := services.NewAppReleaseService(repo, storage, "https://api.enviroo.tech")
	repo.On("FindLatest").Return(nil, gorm.ErrRecordNotFound)

	assert.Nil(t, svc.InfoInstallTerbaru(), "tanpa rilis, link tidak dicantumkan agar tidak 404")
}
