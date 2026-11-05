package services

import (
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"enviroo-be/internal/adapters/mailer"
	"enviroo-be/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	MaxUkuranAPK    int64 = 200 * 1024 * 1024
	contentTypeAPK        = "application/vnd.android.package-archive"
	folderAPK             = "apk"
	maxPanjangVersi       = 20
)

var polaVersionName = regexp.MustCompile(`^[0-9A-Za-z._-]+$`)

type AppReleaseRepoIface interface {
	FindLatest() (*models.AppRelease, error)
	FindByID(releaseID int) (*models.AppRelease, error)
	MaxVersionCode() (int, error)
	AdaWajibDiAntara(current, latest int) (bool, error)
	List() ([]models.AppRelease, error)
	Create(tx *gorm.DB, rilis *models.AppRelease) error
	Save(rilis *models.AppRelease) error
	Delete(rilis *models.AppRelease) error
}

type AppReleaseStorage interface {
	UploadObject(body io.ReadSeeker, size int64, key, contentType string) error
	DeleteObject(key string) error
	PublicURL(key string) string
}

type VersiTerbaruRow struct {
	VersionCode int       `json:"version_code"`
	VersionName string    `json:"version_name"`
	ApkURL      string    `json:"apk_url"`
	FileSize    int64     `json:"file_size"`
	Changelog   string    `json:"changelog"`
	Wajib       bool      `json:"wajib"`
	CreatedAt   time.Time `json:"created_at"`
}

type RilisRow struct {
	ReleaseID   int       `json:"release_id"`
	VersionCode int       `json:"version_code"`
	VersionName string    `json:"version_name"`
	ApkKey      string    `json:"apk_key"`
	ApkURL      string    `json:"apk_url"`
	FileSize    int64     `json:"file_size"`
	Changelog   string    `json:"changelog"`
	IsMandatory bool      `json:"is_mandatory"`
	CreatedBy   *string   `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TambahRilisInput struct {
	File        io.ReadSeeker
	FileName    string
	FileSize    int64
	VersionCode int
	VersionName string
	Changelog   string
	Wajib       bool
	CreatedBy   string
}

type UbahRilisInput struct {
	Changelog *string
	Wajib     *bool
}

// LinkInstallProvider dipakai service lain untuk menyisipkan link APK
// terbaru di email aktivasi. Nil berarti belum ada rilis.
type LinkInstallProvider interface {
	InfoInstallTerbaru() *mailer.InstallInfo
}

func infoInstall(p LinkInstallProvider) *mailer.InstallInfo {
	if p == nil {
		return nil
	}
	return p.InfoInstallTerbaru()
}

type AppReleaseService interface {
	LinkInstallProvider
	CekVersi(currentParam string) (*VersiTerbaruRow, error)
	URLTerbaru() (string, error)
	ListRilis() ([]RilisRow, error)
	TambahRilis(in TambahRilisInput) (*RilisRow, error)
	UbahRilis(releaseIDParam string, in UbahRilisInput) (*RilisRow, error)
	HapusRilis(releaseIDParam string) error
}

type appReleaseService struct {
	repo         AppReleaseRepoIface
	storage      AppReleaseStorage
	apiPublicURL string
}

// apiPublicURL: alamat publik backend untuk link install permanen
// (<apiPublicURL>/app-version/latest). Boleh kosong.
func NewAppReleaseService(repo AppReleaseRepoIface, storage AppReleaseStorage, apiPublicURL string) AppReleaseService {
	return &appReleaseService{
		repo:         repo,
		storage:      storage,
		apiPublicURL: strings.TrimRight(strings.TrimSpace(apiPublicURL), "/"),
	}
}

func (s *appReleaseService) CekVersi(currentParam string) (*VersiTerbaruRow, error) {
	latest, err := s.repo.FindLatest()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errInternal("Gagal mengambil versi aplikasi: " + err.Error())
	}

	wajib := latest.IsMandatory
	if current, convErr := strconv.Atoi(strings.TrimSpace(currentParam)); convErr == nil {
		if current >= latest.VersionCode {
			wajib = false
		} else {
			ada, err := s.repo.AdaWajibDiAntara(current, latest.VersionCode)
			if err != nil {
				return nil, errInternal("Gagal mengecek rilis wajib: " + err.Error())
			}
			wajib = ada
		}
	}

	return &VersiTerbaruRow{
		VersionCode: latest.VersionCode,
		VersionName: latest.VersionName,
		ApkURL:      s.storage.PublicURL(latest.ApkKey),
		FileSize:    latest.FileSize,
		Changelog:   latest.Changelog,
		Wajib:       wajib,
		CreatedAt:   latest.CreatedAt,
	}, nil
}

func (s *appReleaseService) URLTerbaru() (string, error) {
	latest, err := s.repo.FindLatest()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errNotFound("Belum ada rilis aplikasi")
		}
		return "", errInternal("Gagal mengambil versi aplikasi: " + err.Error())
	}
	return s.storage.PublicURL(latest.ApkKey), nil
}

// InfoInstallTerbaru tidak pernah mengembalikan error: gagal atau belum ada
// rilis cukup berarti email dikirim tanpa blok install.
//
// Link memakai endpoint permanen /app-version/latest (selalu ke versi terbaru,
// tidak berakhiran .apk sehingga lebih aman dari filter spam). Bila alamat
// publik backend tidak diset, jatuh ke URL R2 rilis terbaru.
func (s *appReleaseService) InfoInstallTerbaru() *mailer.InstallInfo {
	latest, err := s.repo.FindLatest()
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("app_release: gagal ambil rilis terbaru untuk email: %v", err)
		}
		return nil
	}

	url := s.storage.PublicURL(latest.ApkKey)
	if s.apiPublicURL != "" {
		url = s.apiPublicURL + "/app-version/latest"
	}
	return &mailer.InstallInfo{
		URL:         url,
		VersionName: latest.VersionName,
	}
}

func (s *appReleaseService) ListRilis() ([]RilisRow, error) {
	rows, err := s.repo.List()
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat rilis: " + err.Error())
	}

	hasil := make([]RilisRow, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, s.toRilisRow(r))
	}
	return hasil, nil
}

func (s *appReleaseService) TambahRilis(in TambahRilisInput) (*RilisRow, error) {
	in.VersionName = strings.TrimSpace(in.VersionName)
	if err := validasiTambahRilis(in); err != nil {
		return nil, err
	}

	maks, err := s.repo.MaxVersionCode()
	if err != nil {
		return nil, errInternal("Gagal mengecek versi terakhir: " + err.Error())
	}
	if in.VersionCode <= maks {
		return nil, errConflict(
			fmt.Sprintf("version_code harus lebih besar dari versi terakhir (%d)", maks),
			map[string]interface{}{"version_code_terakhir": maks},
		)
	}

	// Suffix acak supaya key tidak pernah menimpa objek rilis lain,
	// sehingga rollback (hapus objek) selalu aman.
	key := fmt.Sprintf("%s/enviroo-%s-%d-%s.apk",
		folderAPK, in.VersionName, in.VersionCode, uuid.New().String()[:8])

	if err := s.storage.UploadObject(in.File, in.FileSize, key, contentTypeAPK); err != nil {
		return nil, errInternal("Gagal upload APK: " + err.Error())
	}

	rilis := models.AppRelease{
		VersionCode: in.VersionCode,
		VersionName: in.VersionName,
		ApkKey:      key,
		FileSize:    in.FileSize,
		Changelog:   strings.TrimSpace(in.Changelog),
		IsMandatory: in.Wajib,
	}
	if in.CreatedBy != "" {
		createdBy := in.CreatedBy
		rilis.CreatedBy = &createdBy
	}

	if err := s.repo.Create(nil, &rilis); err != nil {
		if delErr := s.storage.DeleteObject(key); delErr != nil {
			log.Printf("app_release: gagal rollback objek %s: %v", key, delErr)
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errConflict("version_code sudah dipakai rilis lain", nil)
		}
		return nil, errInternal("Gagal menyimpan rilis: " + err.Error())
	}

	row := s.toRilisRow(rilis)
	return &row, nil
}

func (s *appReleaseService) UbahRilis(releaseIDParam string, in UbahRilisInput) (*RilisRow, error) {
	rilis, err := s.cariRilis(releaseIDParam)
	if err != nil {
		return nil, err
	}

	if in.Changelog == nil && in.Wajib == nil {
		return nil, errBadRequest("Tidak ada data yang diubah")
	}
	if in.Changelog != nil {
		rilis.Changelog = strings.TrimSpace(*in.Changelog)
	}
	if in.Wajib != nil {
		rilis.IsMandatory = *in.Wajib
	}

	if err := s.repo.Save(rilis); err != nil {
		return nil, errInternal("Gagal mengubah rilis: " + err.Error())
	}

	row := s.toRilisRow(*rilis)
	return &row, nil
}

func (s *appReleaseService) HapusRilis(releaseIDParam string) error {
	rilis, err := s.cariRilis(releaseIDParam)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(rilis); err != nil {
		return errInternal("Gagal menghapus rilis: " + err.Error())
	}

	if err := s.storage.DeleteObject(rilis.ApkKey); err != nil {
		log.Printf("app_release: rilis %d terhapus tapi objek %s gagal dihapus: %v",
			rilis.ReleaseID, rilis.ApkKey, err)
	}
	return nil
}

func (s *appReleaseService) cariRilis(releaseIDParam string) (*models.AppRelease, error) {
	releaseID, err := strconv.Atoi(strings.TrimSpace(releaseIDParam))
	if err != nil || releaseID <= 0 {
		return nil, errBadRequest("release_id tidak valid")
	}

	rilis, err := s.repo.FindByID(releaseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Rilis tidak ditemukan")
		}
		return nil, errInternal("Gagal mencari rilis: " + err.Error())
	}
	return rilis, nil
}

func (s *appReleaseService) toRilisRow(r models.AppRelease) RilisRow {
	return RilisRow{
		ReleaseID:   r.ReleaseID,
		VersionCode: r.VersionCode,
		VersionName: r.VersionName,
		ApkKey:      r.ApkKey,
		ApkURL:      s.storage.PublicURL(r.ApkKey),
		FileSize:    r.FileSize,
		Changelog:   r.Changelog,
		IsMandatory: r.IsMandatory,
		CreatedBy:   r.CreatedBy,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func validasiTambahRilis(in TambahRilisInput) error {
	if in.File == nil {
		return errBadRequest("File APK wajib diupload")
	}
	if !strings.EqualFold(filepath.Ext(in.FileName), ".apk") {
		return errBadRequest("File harus berekstensi .apk")
	}
	if in.FileSize <= 0 {
		return errBadRequest("File APK kosong")
	}
	if in.FileSize > MaxUkuranAPK {
		return errBadRequest(fmt.Sprintf("Ukuran APK maksimal %d MB", MaxUkuranAPK/(1024*1024)))
	}
	if in.VersionCode <= 0 {
		return errBadRequest("version_code harus angka lebih dari 0")
	}
	name := strings.TrimSpace(in.VersionName)
	if name == "" || len(name) > maxPanjangVersi || !polaVersionName.MatchString(name) {
		return errBadRequest("version_name wajib diisi (maks 20 karakter, hanya angka, huruf, titik, - dan _)")
	}
	return nil
}
