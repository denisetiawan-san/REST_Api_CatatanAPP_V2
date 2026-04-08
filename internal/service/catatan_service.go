package service

// package service → sama dengan interface.go, satu package untuk semua service
// → implementasi konkret dari interface CatatanSvc yang didefinisikan di interface.go

import (
	"catatan_app/internal/apperror"
	// import apperror → butuh ErrInvalidID untuk validasi bisnis id <= 0

	"catatan_app/internal/dto"
	// import dto → butuh tipe dto.CreateCatatanRequest, dto.UpdateCatatanRequest, dto.PaginationQuery
	// → service menerima input dari handler dalam bentuk DTO

	"catatan_app/internal/modul"
	// import modul → butuh tipe modul.Catatan untuk bangun domain object

	"catatan_app/internal/repository"
	// import repository → butuh tipe repository.CatatanRepo (interface) sebagai dependency
	// → depend ke interface, bukan ke struct CatatanRepository yang konkret

	"context"
	// import context → butuh context.Context sebagai parameter pertama setiap method

	"errors"
	// import errors → butuh errors.New() untuk buat error validasi bisnis inline

	"strings"
	// import strings → butuh strings.TrimSpace() untuk bersihkan whitespace input
)

// CatatanService → struct konkret yang mengimplementasikan interface CatatanSvc
// → memegang repository.CatatanRepo sebagai dependency — interface, bukan struct konkret
// → kenapa interface: supaya di unit test bisa di-inject MockCatatanRepo tanpa database nyata
type CatatanService struct {
	repo repository.CatatanRepo
	// repo → interface CatatanRepo, bukan *CatatanRepository
	// → service tidak tahu dan tidak peduli implementasinya MySQL atau mock
}

// NewCatatanService → constructor untuk membuat instance CatatanService
// → menerima repository.CatatanRepo dari main.go dan inject ke struct
// → di production: main.go inject CatatanRepository (nyata)
// → di unit test: inject MockCatatanRepo (mock)
func NewCatatanService(repo repository.CatatanRepo) *CatatanService {
	return &CatatanService{repo: repo}
}

// compile-time check → pastikan CatatanService memenuhi semua method di interface CatatanSvc
// → kalau ada method yang belum diimplementasikan → kompilasi error, bukan runtime error
var _ CatatanSvc = (*CatatanService)(nil)

// Create → implementasi kontrak Create dari interface CatatanSvc
// → pintu masuk logic bisnis untuk buat catatan baru
// → menerima DTO dari handler, return domain object ke handler
func (s *CatatanService) Create(ctx context.Context, req dto.CreateCatatanRequest) (*modul.Catatan, error) {
	judul := strings.TrimSpace(req.Judul)
	isi := strings.TrimSpace(req.Isi)
	// TrimSpace → hapus spasi, tab, newline di awal dan akhir string
	// → kenapa dilakukan di service, bukan handler: karena ini keputusan bisnis
	//   "  judul  " dianggap sama dengan "judul" — itu aturan bisnis, bukan validasi format
	// → kenapa setelah validate tag di DTO: validate:"required" hanya cek string tidak kosong
	//   "   " (spasi saja) lolos required tapi tidak valid secara bisnis → perlu TrimSpace dulu

	if judul == "" {
		return nil, errors.New("judul harus diisi")
		// → validasi bisnis setelah TrimSpace — cegah judul yang hanya berisi spasi
		// → kenapa tidak pakai apperror.ErrBadRequest: karena ingin kasih pesan spesifik
		//   "judul harus diisi" lebih informatif dari "request tidak valid"
	}

	catatan := &modul.Catatan{
		Judul: judul,
		Isi:   isi,
		// → build domain object dari data yang sudah bersih
		// → ID dan CreatedAt tidak diisi — akan diisi database otomatis
		// → Arsip tidak diisi — default false sesuai nilai zero value bool di Go
	}

	return s.repo.Create(ctx, catatan)
	// → serahkan domain object ke repository, service tidak tahu cara INSERT
	// → return langsung hasil dari repository — tidak ada transformasi tambahan
}

// List → implementasi kontrak List dari interface CatatanSvc
// → pintu masuk logic bisnis untuk ambil semua catatan dengan filter dan pagination
func (s *CatatanService) List(ctx context.Context, arsip *bool, pagination dto.PaginationQuery) ([]modul.Catatan, int, error) {
	if pagination.Page < 1 {
		pagination.Page = 1
		// → sanitasi: page tidak boleh 0 atau negatif
		// → kalau handler kirim page=0 → paksa jadi 1
	}
	if pagination.Limit < 1 {
		pagination.Limit = 10
		// → sanitasi: limit tidak boleh 0 atau negatif → paksa ke default 10
	}
	if pagination.Limit > 100 {
		pagination.Limit = 100
		// → proteksi: limit tidak boleh lebih dari 100
		// → cegah client kirim limit=999999 yang bisa bikin server crash
		// → kenapa 100: cukup besar untuk UI tapi cukup kecil untuk proteksi server
	}
	// → kenapa sanitasi di service, bukan handler:
	//   handler hanya parse string ke int (format)
	//   service yang memutuskan nilai mana yang valid (bisnis)

	return s.repo.GetAll(ctx, arsip, pagination.Page, pagination.Limit)
	// → teruskan ke repository dengan nilai yang sudah disanitasi
	// → return tiga nilai: data, total, error — total untuk MetaData pagination di handler
}

// GetByID → implementasi kontrak GetByID dari interface CatatanSvc
// → pintu masuk logic bisnis untuk ambil satu catatan berdasarkan id
func (s *CatatanService) GetByID(ctx context.Context, id int) (*modul.Catatan, error) {
	if id <= 0 {
		return nil, apperror.ErrInvalidID
		// → validasi bisnis: id dari database selalu positif (AUTO_INCREMENT mulai dari 1)
		// → id 0 atau negatif tidak mungkin ada di database — tolak sebelum query
		// → handler akan mapping ErrInvalidID ke HTTP 400
	}

	return s.repo.GetByID(ctx, id)
	// → kalau id valid, serahkan ke repository
	// → repository return ErrNotFound kalau id tidak ada di database
}

// Update → implementasi kontrak Update dari interface CatatanSvc
// → pintu masuk logic bisnis untuk update catatan berdasarkan id
// → ini partial update — client boleh kirim hanya judul atau hanya isi
func (s *CatatanService) Update(ctx context.Context, id int, req dto.UpdateCatatanRequest) (*modul.Catatan, error) {
	if id <= 0 {
		return nil, apperror.ErrInvalidID
	}

	catatan, err := s.repo.GetByID(ctx, id)
	// → fetch data lama dulu sebelum update
	// → dua tujuan: validasi data exists + ambil nilai lama untuk partial update
	if err != nil {
		return nil, err
		// → kalau ErrNotFound: data tidak ada → return ke handler → HTTP 404
	}

	if req.Judul != "" {
		catatan.Judul = strings.TrimSpace(req.Judul)
		// → update judul hanya kalau client kirim judul baru
		// → kalau client tidak kirim judul → pertahankan nilai lama
	}
	if req.Isi != "" {
		catatan.Isi = strings.TrimSpace(req.Isi)
		// → update isi hanya kalau client kirim isi baru
		// → kalau client tidak kirim isi → pertahankan nilai lama
	}
	// → ini yang disebut partial update — merge nilai baru dengan nilai lama
	// → kalau langsung overwrite tanpa cek: field yang tidak dikirim jadi string kosong

	if catatan.Judul == "" {
		return nil, errors.New("judul harus ada")
		// → edge case: data lama judulnya kosong (tidak mungkin tapi defensive check)
	}

	return s.repo.Update(ctx, id, catatan)
	// → serahkan domain object yang sudah di-merge ke repository untuk di-UPDATE
}

// Arsip → implementasi kontrak Arsip dari interface CatatanSvc
// → shortcut untuk SetArsip dengan nilai true
func (s *CatatanService) Arsip(ctx context.Context, id int) (*modul.Catatan, error) {
	if id <= 0 {
		return nil, apperror.ErrInvalidID
		// → validasi bisnis: id tidak boleh 0 atau negatif
		// → handler akan mapping ErrInvalidID ke HTTP 400
	}

	return s.repo.SetArsip(ctx, id, true)
	// → true → arsipkan catatan (kolom arsip di database jadi true)
	// → tidak perlu fetch data lama dulu karena SetArsip di repository
	//   sudah cek RowsAffected — kalau 0 → return ErrNotFound → HTTP 404
}

// Unarsip → implementasi kontrak Unarsip dari interface CatatanSvc
// → shortcut untuk SetArsip dengan nilai false
func (s *CatatanService) Unarsip(ctx context.Context, id int) (*modul.Catatan, error) {
	if id <= 0 {
		return nil, apperror.ErrInvalidID
		// → validasi bisnis: id tidak boleh 0 atau negatif
		// → handler akan mapping ErrInvalidID ke HTTP 400
	}

	return s.repo.SetArsip(ctx, id, false)
	// → false → kembalikan catatan dari arsip (kolom arsip di database jadi false)
	// → sama dengan Arsip — repository yang handle cek apakah id ada atau tidak
}

// Delete → implementasi kontrak Delete dari interface CatatanSvc
// → return hanya error, tidak return data karena data sudah dihapus
func (s *CatatanService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return apperror.ErrInvalidID
		// → return error saja, bukan (nil, error)
		// → karena signature Delete di interface hanya return error, bukan (*modul.Catatan, error)
		// → handler akan mapping ErrInvalidID ke HTTP 400
	}

	return s.repo.Delete(ctx, id)
	// → serahkan id ke repository untuk dihapus dari database
	// → repository return ErrNotFound kalau id tidak ada → HTTP 404
	// → return nil kalau berhasil → handler return 204 No Content
}

// Kenapa Update fetch data lama dulu via GetByID sebelum update:
// → validasi exists: kalau id tidak ada → ErrNotFound → HTTP 404
// → partial update: ambil nilai lama, merge dengan nilai baru yang dikirim client
// → tanpa fetch dulu: tidak tahu nilai lama, tidak bisa partial update

// Kenapa Arsip dan Unarsip tidak fetch dulu:
// → karena SetArsip di repository sudah cek RowsAffected
// → kalau 0 rows affected → return ErrNotFound
// → tidak perlu dua query (GetByID + SetArsip) kalau satu query sudah cukup

// Pattern setiap function di service:
// 1. Validasi bisnis    → id <= 0 return ErrInvalidID, field kosong return error spesifik
// 2. Transformasi data  → TrimSpace input, merge nilai lama dan baru untuk partial update
// 3. Call repository    → serahkan domain object, bukan DTO
// 4. Return hasil       → domain object atau error ke handler, tanpa transformasi tambahan
