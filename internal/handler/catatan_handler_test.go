package handler_test

// → nama package ini sengaja diberi suffix "_test"
// → artinya ini package terpisah dari package handler yang sedang dites
// → file ini tetap berada di folder internal/handler, tapi Go memperlakukannya sebagai package berbeda
// → konsekuensinya: hanya bisa akses identifier yang exported (huruf kapital) dari package handler
// → tujuannya: memaksa kita test dari sudut pandang pengguna package, bukan dari dalam
// → keuntungannya: kalau kita refactor internal handler, test tidak perlu diubah selama hasilnya sama

import (
	"bytes"
	// → import package bytes bawaan Go
	// → dibutuhkan untuk bytes.NewBuffer() dan bytes.NewBufferString()
	// → dipakai untuk buat request body dari JSON yang sudah di-encode ke []byte

	"catatan_app/internal/apperror"
	// → import package apperror yang berisi semua sentinel error
	// → dibutuhkan untuk setup mock return error tertentu
	// → contoh: mockSvc.On(...).Return(nil, apperror.ErrNotFound)

	"catatan_app/internal/dto"
	// → import package dto yang berisi struct request dan response
	// → dibutuhkan untuk buat request body dan decode response body saat assert

	"catatan_app/internal/handler"
	// → import package yang sedang dites
	// → dibutuhkan untuk panggil handler.NewCatatanHandler() dan buat instance handler

	"catatan_app/internal/modul"
	// → import package modul yang berisi domain struct Catatan
	// → dibutuhkan untuk buat data dummy yang dikembalikan mock service

	"context"
	// → import package context bawaan Go
	// → dibutuhkan untuk signature method mock service yang punya parameter context.Context

	"encoding/json"
	// → import package encoding/json bawaan Go
	// → dibutuhkan untuk json.Marshal() mengubah struct ke []byte JSON untuk request body
	// → dibutuhkan untuk json.Unmarshal() mengubah []byte response body ke struct untuk assert

	"net/http"
	// → import package net/http bawaan Go
	// → dibutuhkan untuk konstanta HTTP method: http.MethodPost, http.MethodGet, http.MethodDelete
	// → dibutuhkan untuk konstanta HTTP status: http.StatusCreated, http.StatusOK, dll

	"net/http/httptest"
	// → import package httptest bawaan Go — khusus untuk testing HTTP handler
	// → dibutuhkan untuk httptest.NewRequest() membuat HTTP request tiruan tanpa server nyata
	// → dibutuhkan untuk httptest.NewRecorder() merekam response handler tanpa mengirim ke network

	"testing"
	// → import package testing bawaan Go
	// → wajib ada di setiap file test Go
	// → menyediakan tipe *testing.T yang dipakai di setiap function test untuk report hasil

	"time"
	// → import package time bawaan Go
	// → dibutuhkan untuk mengisi field CreatedAt di data dummy dummyCatatan

	"github.com/stretchr/testify/assert"
	// → import package assert dari library testify
	// → menyediakan fungsi assertion yang lebih readable dari cara manual Go
	// → contoh: assert.Equal(t, http.StatusOK, w.Code) lebih jelas dari if w.Code != 200 { t.Fatal(...) }

	"github.com/stretchr/testify/mock"
	// → import package mock dari library testify
	// → menyediakan struct mock.Mock yang di-embed untuk buat mock object
)

// MockCatatanSvc adalah struct yang berperan sebagai tiruan service
// → struct ini mengimplementasikan interface CatatanSvc persis seperti implementasi nyata
// → bedanya: tidak ada business logic, tidak ada koneksi database — semua jawaban diatur dari luar saat test
// → tujuan utama: mengisolasi test handler dari dependency service dan database
// → handler tidak tahu apakah ini mock atau implementasi nyata — karena keduanya penuhi interface CatatanSvc
type MockCatatanSvc struct {
	mock.Mock
	// → embed struct mock.Mock dari testify ke dalam MockCatatanSvc
	// → embed artinya semua method dari mock.Mock otomatis tersedia di MockCatatanSvc
	// → method yang didapat: On() untuk setup ekspektasi, Called() untuk catat panggilan,
	// →   AssertExpectations() untuk verifikasi semua ekspektasi terpenuhi,
	// →   AssertNotCalled() untuk verifikasi method tidak pernah dipanggil
}

// Di bawah ini adalah implementasi setiap method dari interface CatatanSvc
// Setiap method mengikuti pola yang sama persis:
// Langkah 1 → panggil m.Called() dengan semua argumen yang diterima
// Langkah 2 → m.Called() mengembalikan args — isinya adalah nilai yang sudah di-setup via .On().Return()
// Langkah 3 → ambil nilai dari args sesuai posisi dan tipe, lalu return

func (m *MockCatatanSvc) Create(ctx context.Context, req dto.CreateCatatanRequest) (*modul.Catatan, error) {
	// → method Create milik MockCatatanSvc, receiver pointer ke MockCatatanSvc
	// → signature harus identik dengan method Create di interface CatatanSvc
	// → kalau signature berbeda: MockCatatanSvc tidak memenuhi interface → compile error

	args := m.Called(ctx, req)
	// → beritahu mock bahwa method Create dipanggil dengan argumen ctx dan req
	// → testify akan cari ekspektasi yang cocok — yang di-setup via mockSvc.On("Create", ...)
	// → args berisi semua nilai yang sudah di-setup via .Return(...) untuk ekspektasi ini

	if args.Get(0) == nil {
		// → cek nil dulu sebelum type assert
		// → kalau langsung type assert ke *modul.Catatan saat nilnya nil → panic
		// → terjadi saat test setup .Return(nil, someError) untuk simulasi kegagalan
		return nil, args.Error(1)
		// → return nil untuk *modul.Catatan dan error yang sudah di-setup
	}
	return args.Get(0).(*modul.Catatan), args.Error(1)
	// → args.Get(0).(*modul.Catatan) → ambil return value pertama lalu type assert ke *modul.Catatan
	// → args.Error(1) → ambil return value kedua sebagai error
}

func (m *MockCatatanSvc) List(ctx context.Context, arsip *bool, pagination dto.PaginationQuery) ([]modul.Catatan, int, error) {
	// → method List milik MockCatatanSvc
	// → menerima filter arsip (pointer bool) dan pagination query

	args := m.Called(ctx, arsip, pagination)
	// → catat panggilan dengan semua argumen: context, pointer bool arsip, pagination

	return args.Get(0).([]modul.Catatan), args.Int(1), args.Error(2)
	// → args.Get(0).([]modul.Catatan) → ambil return pertama, type assert ke slice Catatan
	// → tidak perlu cek nil karena slice kosong []modul.Catatan{} bukan nil
	// → args.Int(1) → ambil return kedua sebagai int — ini total count untuk pagination
	// → args.Error(2) → ambil return ketiga sebagai error
}

func (m *MockCatatanSvc) GetByID(ctx context.Context, id int) (*modul.Catatan, error) {
	// → method GetByID milik MockCatatanSvc
	// → menerima context dan id catatan yang dicari

	args := m.Called(ctx, id)
	// → catat panggilan dengan argumen context dan id

	if args.Get(0) == nil {
		// → cek nil dulu — terjadi saat test simulasi "data tidak ditemukan"
		return nil, args.Error(1)
	}
	return args.Get(0).(*modul.Catatan), args.Error(1)
}

func (m *MockCatatanSvc) Update(ctx context.Context, id int, req dto.UpdateCatatanRequest) (*modul.Catatan, error) {
	// → method Update milik MockCatatanSvc
	// → menerima context, id yang diupdate, dan DTO request berisi data baru

	args := m.Called(ctx, id, req)
	// → catat panggilan dengan semua tiga argumen

	if args.Get(0) == nil {
		// → cek nil dulu — terjadi saat test simulasi update gagal
		return nil, args.Error(1)
	}
	return args.Get(0).(*modul.Catatan), args.Error(1)
}

func (m *MockCatatanSvc) Delete(ctx context.Context, id int) error {
	// → method Delete milik MockCatatanSvc
	// → hanya return error, tidak ada data yang dikembalikan

	args := m.Called(ctx, id)
	// → catat panggilan dengan argumen context dan id

	return args.Error(0)
	// → args.Error(0) → ambil return value pertama sebagai error
	// → tidak perlu cek nil karena tidak ada pointer yang perlu di-type assert
}

func (m *MockCatatanSvc) Arsip(ctx context.Context, id int) (*modul.Catatan, error) {
	// → method Arsip milik MockCatatanSvc
	// → menerima context dan id catatan yang akan diarsipkan

	args := m.Called(ctx, id)
	// → catat panggilan dengan argumen context dan id

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*modul.Catatan), args.Error(1)
}

func (m *MockCatatanSvc) Unarsip(ctx context.Context, id int) (*modul.Catatan, error) {
	// → method Unarsip milik MockCatatanSvc
	// → menerima context dan id catatan yang akan dibuka dari arsip

	args := m.Called(ctx, id)
	// → catat panggilan dengan argumen context dan id

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*modul.Catatan), args.Error(1)
}

// dummyCatatan adalah data dummy yang dipakai sebagai return value mock di banyak test
var dummyCatatan = &modul.Catatan{
	// → dideklarasikan di level package agar bisa diakses semua function test di file ini
	// → tidak perlu deklarasi ulang di setiap test function yang butuh data ini
	// → kalau suatu test butuh data yang berbeda, buat variabel lokal di dalam test tersebut

	ID: 1,
	// → ID = 1, angka valid untuk simulasi catatan yang sudah tersimpan di database

	Judul: "belajar golang",
	// → judul dummy yang cukup deskriptif untuk dibaca saat test gagal

	Isi: "golang adalah bahasa pemrograman",
	// → isi dummy, nilai tidak terlalu penting selama tidak kosong

	Arsip: false,
	// → status arsip false = catatan aktif, bukan arsip

	CreatedAt: time.Now(),
	// → isi dengan waktu saat ini — nilainya tidak di-assert di test manapun
	// → hanya untuk memenuhi field struct agar tidak zero value
}

// ===== TEST CREATE =====

func TestHandler_Create_Sukses(t *testing.T) {
	// → test skenario: handler Create menerima request valid dan berhasil membuat catatan baru
	// → yang dites di sini adalah LAYER HANDLER, bukan service atau repo
	// → service sudah diganti dengan mock, jadi kita hanya test logika HTTP handler

	// ── ARRANGE ──────────────────────────────────────────────
	mockSvc := new(MockCatatanSvc)
	// → buat instance mock service baru
	// → dibuat baru di setiap test agar state mock tidak bocor antar test

	h := handler.NewCatatanHandler(mockSvc)
	// → buat instance handler dengan inject mock service
	// → handler tidak tahu ini mock — karena mockSvc memenuhi interface CatatanSvc

	body := dto.CreateCatatanRequest{
		Judul: "belajar golang",
		// → judul valid, tidak kosong
		Isi: "golang adalah bahasa pemrograman",
		// → isi valid, tidak kosong
	}
	// → buat struct request yang akan dikirim sebagai JSON body

	bodyJSON, _ := json.Marshal(body)
	// → json.Marshal() → ubah struct body menjadi []byte berformat JSON
	// → hasilnya: []byte(`{"judul":"belajar golang","isi":"golang adalah bahasa pemrograman"}`)
	// → _ mengabaikan error karena struct ini dijamin bisa di-marshal

	mockSvc.On("Create", mock.Anything, body).Return(dummyCatatan, nil)
	// → setup ekspektasi: kalau service.Create() dipanggil dengan context apapun dan body persis ini
	// → mock.Anything → cocokkan context dengan nilai apapun
	// → body → cocokkan argumen kedua dengan struct body persis sama (deep equal)
	// → .Return(dummyCatatan, nil) → kalau terpenuhi, return dummyCatatan dan nil error

	r := httptest.NewRequest(http.MethodPost, "/api/v1/catatan", bytes.NewBuffer(bodyJSON))
	// → httptest.NewRequest() → buat HTTP request tiruan tanpa perlu server nyata
	// → http.MethodPost → method HTTP yang dipakai = "POST"
	// → "/api/v1/catatan" → URL path yang dituju
	// → bytes.NewBuffer(bodyJSON) → request body berisi JSON yang sudah di-encode
	// → hasilnya: *http.Request yang siap dipakai seperti request dari client nyata

	r.Header.Set("Content-Type", "application/json")
	// → set header Content-Type ke "application/json"
	// → dibutuhkan agar handler tahu cara decode body request
	// → tanpa ini: handler mungkin tidak bisa decode body dengan benar

	w := httptest.NewRecorder()
	// → httptest.NewRecorder() → buat response recorder yang merekam semua yang ditulis handler
	// → w.Code → menyimpan HTTP status code yang ditulis handler
	// → w.Body → menyimpan response body yang ditulis handler
	// → w.Header() → menyimpan response headers yang ditulis handler
	// → ini pengganti http.ResponseWriter nyata — tidak kirim ke network, hanya rekam di memori

	// ── ACT ──────────────────────────────────────────────────
	h.Create(w, r)
	// → panggil method Create milik handler secara langsung
	// → handler akan decode body, panggil service (mock), lalu tulis response ke w
	// → tidak perlu router atau server — handler dipanggil langsung seperti function biasa

	// ── ASSERT ───────────────────────────────────────────────
	assert.Equal(t, http.StatusCreated, w.Code)
	// → pastikan handler menulis status code 201 Created
	// → http.StatusCreated = 201
	// → kalau handler tulis status lain: test fail

	var response dto.CatatanResponse
	// → deklarasi variabel untuk menampung hasil decode response body
	// → tipe dto.CatatanResponse karena itu yang handler kirim sebagai JSON

	json.Unmarshal(w.Body.Bytes(), &response)
	// → json.Unmarshal() → decode []byte dari response body ke struct response
	// → w.Body.Bytes() → ambil semua byte yang sudah direkam recorder
	// → &response → pointer ke struct yang akan diisi hasil decode

	assert.Equal(t, 1, response.ID)
	// → pastikan ID di response = 1 sesuai dummyCatatan
	// → membuktikan handler return data yang benar dari service

	assert.Equal(t, "belajar golang", response.Judul)
	// → pastikan judul di response sesuai yang dikirim

	mockSvc.AssertExpectations(t)
	// → verifikasi semua ekspektasi yang di-setup via .On() benar-benar dipanggil
	// → kalau handler tidak memanggil service.Create() → test fail
}

func TestHandler_Create_BadRequest(t *testing.T) {
	// → test skenario: handler Create menerima JSON rusak dan harus return 400

	// ── ARRANGE ──────────────────────────────────────────────
	mockSvc := new(MockCatatanSvc)
	h := handler.NewCatatanHandler(mockSvc)
	// → tidak ada mockSvc.On() — tidak expect service dipanggil sama sekali

	r := httptest.NewRequest(http.MethodPost, "/api/v1/catatan", bytes.NewBufferString("ini bukan json"))
	// → bytes.NewBufferString() → buat buffer dari string biasa (bukan JSON)
	// → "ini bukan json" → string yang tidak bisa di-decode sebagai JSON
	// → tujuan: simulasi client yang kirim body rusak atau bukan JSON

	r.Header.Set("Content-Type", "application/json")
	// → set Content-Type tetap application/json
	// → handler akan coba decode body sebagai JSON tapi gagal karena isinya bukan JSON valid

	w := httptest.NewRecorder()
	// → buat response recorder untuk merekam response handler

	// ── ACT ──────────────────────────────────────────────────
	h.Create(w, r)
	// → panggil handler — handler akan coba decode body, gagal, lalu tulis error response

	// ── ASSERT ───────────────────────────────────────────────
	assert.Equal(t, http.StatusBadRequest, w.Code)
	// → pastikan handler return 400 Bad Request saat JSON rusak
	// → http.StatusBadRequest = 400

	mockSvc.AssertNotCalled(t, "Create")
	// → verifikasi service.Create() tidak pernah dipanggil
	// → handler harus tolak request sebelum sampai ke service saat JSON tidak valid
}

// ===== TEST GET BY ID =====

func TestHandler_GetByID_Sukses(t *testing.T) {
	// → test skenario: handler GetByID menerima ID valid dan data ditemukan

	// ── ARRANGE ──────────────────────────────────────────────
	mockSvc := new(MockCatatanSvc)
	h := handler.NewCatatanHandler(mockSvc)

	mockSvc.On("GetByID", mock.Anything, 1).Return(dummyCatatan, nil)
	// → setup ekspektasi: service.GetByID() dipanggil dengan context apapun dan id = 1
	// → return dummyCatatan dan nil error

	r := httptest.NewRequest(http.MethodGet, "/api/v1/catatan/1", nil)
	// → buat GET request ke path /api/v1/catatan/1
	// → body = nil karena GET request tidak punya body

	r.SetPathValue("id", "1")
	// → set path value "id" = "1" secara manual
	// → di production ini dilakukan otomatis oleh router saat pattern "/catatan/{id}" cocok
	// → di test kita harus set manual karena tidak pakai router — handler dipanggil langsung
	// → handler membaca id via r.PathValue("id") — ini yang menyediakan nilainya

	w := httptest.NewRecorder()
	// → buat response recorder

	// ── ACT ──────────────────────────────────────────────────
	h.GetByID(w, r)
	// → panggil handler GetByID langsung

	// ── ASSERT ───────────────────────────────────────────────
	assert.Equal(t, http.StatusOK, w.Code)
	// → pastikan handler return 200 OK saat data ditemukan
	// → http.StatusOK = 200

	mockSvc.AssertExpectations(t)
	// → verifikasi service.GetByID() dipanggil dengan argumen yang benar
}

func TestHandler_GetByID_TidakDitemukan(t *testing.T) {
	// → test skenario: handler GetByID dengan ID yang tidak ada di database

	// ── ARRANGE ──────────────────────────────────────────────
	mockSvc := new(MockCatatanSvc)
	h := handler.NewCatatanHandler(mockSvc)

	mockSvc.On("GetByID", mock.Anything, 99).Return(nil, apperror.ErrNotFound)
	// → setup ekspektasi: GetByID dengan id = 99 return nil dan ErrNotFound
	// → mensimulasikan kondisi data tidak ada di database

	r := httptest.NewRequest(http.MethodGet, "/api/v1/catatan/99", nil)
	// → buat GET request dengan ID = 99 yang tidak ada di database (simulasi)

	r.SetPathValue("id", "99")
	// → set path value "id" = "99" secara manual

	w := httptest.NewRecorder()

	// ── ACT ──────────────────────────────────────────────────
	h.GetByID(w, r)

	// ── ASSERT ───────────────────────────────────────────────
	assert.Equal(t, http.StatusNotFound, w.Code)
	// → pastikan handler return 404 Not Found saat service return ErrNotFound
	// → http.StatusNotFound = 404
	// → ini yang membuktikan handler memetakan ErrNotFound ke HTTP 404 dengan benar

	mockSvc.AssertExpectations(t)
	// → verifikasi service.GetByID() dipanggil dengan ID = 99
}

func TestHandler_GetByID_IDBukanAngka(t *testing.T) {
	// → test skenario: handler GetByID menerima ID yang bukan angka

	// ── ARRANGE ──────────────────────────────────────────────
	mockSvc := new(MockCatatanSvc)
	h := handler.NewCatatanHandler(mockSvc)
	// → tidak ada mockSvc.On() — tidak expect service dipanggil

	r := httptest.NewRequest(http.MethodGet, "/api/v1/catatan/abc", nil)
	// → buat GET request dengan ID = "abc" yang bukan angka

	r.SetPathValue("id", "abc")
	// → set path value "id" = "abc" — string yang tidak bisa di-parse ke int

	w := httptest.NewRecorder()

	// ── ACT ──────────────────────────────────────────────────
	h.GetByID(w, r)
	// → handler akan coba parse "abc" ke int, gagal, lalu tulis error response

	// ── ASSERT ───────────────────────────────────────────────
	assert.Equal(t, http.StatusBadRequest, w.Code)
	// → pastikan handler return 400 Bad Request saat ID bukan angka
	// → membuktikan handler memvalidasi format ID sebelum panggil service

	mockSvc.AssertNotCalled(t, "GetByID")
	// → verifikasi service.GetByID() tidak pernah dipanggil
	// → handler harus tolak request sebelum sampai ke service saat ID tidak valid
}

// ===== TEST DELETE =====

func TestHandler_Delete_Sukses(t *testing.T) {
	// → test skenario: handler Delete menerima ID valid dan data berhasil dihapus

	// ── ARRANGE ──────────────────────────────────────────────
	mockSvc := new(MockCatatanSvc)
	h := handler.NewCatatanHandler(mockSvc)

	mockSvc.On("Delete", mock.Anything, 1).Return(nil)
	// → setup ekspektasi: service.Delete() dipanggil dengan context apapun dan id = 1
	// → .Return(nil) → simulasi delete berhasil tanpa error

	r := httptest.NewRequest(http.MethodDelete, "/api/v1/catatan/1", nil)
	// → buat DELETE request ke path /api/v1/catatan/1
	// → body = nil karena DELETE request tidak punya body

	r.SetPathValue("id", "1")
	// → set path value "id" = "1" secara manual

	w := httptest.NewRecorder()

	// ── ACT ──────────────────────────────────────────────────
	h.Delete(w, r)
	// → panggil handler Delete langsung

	// ── ASSERT ───────────────────────────────────────────────
	assert.Equal(t, http.StatusNoContent, w.Code)
	// → pastikan handler return 204 No Content saat delete berhasil
	// → http.StatusNoContent = 204
	// → 204 artinya sukses tapi tidak ada body response — standar REST untuk delete

	mockSvc.AssertExpectations(t)
	// → verifikasi service.Delete() dipanggil dengan argumen yang benar
}

func TestHandler_Delete_TidakDitemukan(t *testing.T) {
	// → test skenario: handler Delete dengan ID yang tidak ada di database

	// ── ARRANGE ──────────────────────────────────────────────
	mockSvc := new(MockCatatanSvc)
	h := handler.NewCatatanHandler(mockSvc)

	mockSvc.On("Delete", mock.Anything, 99).Return(apperror.ErrNotFound)
	// → setup ekspektasi: Delete dengan id = 99 return ErrNotFound
	// → mensimulasikan kondisi: ID valid tapi data tidak ada di database

	r := httptest.NewRequest(http.MethodDelete, "/api/v1/catatan/99", nil)
	// → buat DELETE request dengan ID = 99 yang tidak ada

	r.SetPathValue("id", "99")
	// → set path value secara manual

	w := httptest.NewRecorder()

	// ── ACT ──────────────────────────────────────────────────
	h.Delete(w, r)

	// ── ASSERT ───────────────────────────────────────────────
	assert.Equal(t, http.StatusNotFound, w.Code)
	// → pastikan handler return 404 Not Found saat service return ErrNotFound
	// → membuktikan handler memetakan ErrNotFound ke HTTP 404 dengan benar

	mockSvc.AssertExpectations(t)
	// → verifikasi service.Delete() dipanggil dengan ID = 99
}

// Cara menjalankan test file ini:
// go test ./internal/handler/... -v
// → go test = perintah Go untuk menjalankan semua test
// → ./internal/handler/... = path ke semua package di dalam folder internal/handler
// → -v = verbose mode, tampilkan nama setiap test dan hasilnya (--- PASS atau --- FAIL)
// → tanpa -v: hanya tampilkan summary akhir, tidak tahu test mana yang pass atau fail
