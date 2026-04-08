package service_test

// → nama package ini sengaja diberi suffix "_test"
// → artinya ini package terpisah dari package service yang sedang dites
// → file ini tetap berada di folder internal/service, tapi Go memperlakukannya sebagai package berbeda
// → konsekuensinya: hanya bisa akses identifier yang exported (huruf kapital) dari package service
// → tujuannya: memaksa kita test dari sudut pandang pengguna package, bukan dari dalam
// → keuntungannya: kalau kita refactor internal service, test tidak perlu diubah selama hasilnya sama
// → analogi: bukan masuk dapur untuk cek cara masak, tapi duduk di meja tamu untuk cek apakah makanannya enak

import (
	"catatan_app/internal/apperror"
	// → import package apperror yang berisi semua sentinel error
	// → dibutuhkan untuk assert bahwa service return error yang tepat
	// → contoh: assert.Equal(t, apperror.ErrInvalidID, err)

	"catatan_app/internal/dto"
	// → import package dto yang berisi struct request
	// → dibutuhkan untuk buat input yang dikirim ke service saat test
	// → contoh: dto.CreateCatatanRequest{Judul: "...", Isi: "..."}

	"catatan_app/internal/modul"
	// → import package modul yang berisi domain struct Catatan
	// → dibutuhkan untuk buat data dummy yang dikembalikan mock repo
	// → contoh: &modul.Catatan{ID: 1, Judul: "..."}

	"catatan_app/internal/service"
	// → import package yang sedang dites
	// → dibutuhkan untuk panggil NewCatatanService() dan buat instance service
	// → ini satu-satunya package internal yang langsung kita test perilakunya

	"context"
	// → import package context bawaan Go
	// → dibutuhkan untuk context.Background() sebagai argumen pertama setiap method service
	// → context.Background() = context kosong tanpa deadline, cocok untuk dipakai di test

	"testing"
	// → import package testing bawaan Go
	// → wajib ada di setiap file test Go
	// → menyediakan tipe *testing.T yang dipakai di setiap function test untuk report hasil

	"time"
	// → import package time bawaan Go
	// → dibutuhkan untuk mengisi field CreatedAt di data dummy dummyCatatan
	// → time.Now() menghasilkan waktu saat ini bertipe time.Time

	"github.com/stretchr/testify/assert"
	// → import package assert dari library testify
	// → menyediakan fungsi assertion yang lebih readable dari cara manual Go
	// → cara manual Go: if result != expected { t.Fatalf("expected %v got %v", expected, result) }
	// → dengan testify: assert.Equal(t, expected, result) — jauh lebih singkat dan jelas

	"github.com/stretchr/testify/mock"
	// → import package mock dari library testify
	// → menyediakan struct mock.Mock yang bisa di-embed untuk buat mock object
	// → mock object = objek palsu yang mensimulasikan behavior dependency tanpa implementasi nyata
)

// MockCatatanRepo adalah struct yang berperan sebagai tiruan database
// → struct ini mengimplementasikan interface CatatanRepo persis seperti implementasi nyata
// → bedanya: tidak ada SQL, tidak ada koneksi database — semua jawaban diatur dari luar saat test
// → tujuan utama: mengisolasi test service dari dependency database
// → kalau tidak pakai mock dan pakai database nyata: test jadi lambat, butuh setup DB, tidak bisa jalan di CI tanpa DB
// → dengan mock: test bisa jalan di mana saja, kapan saja, dalam milidetik
type MockCatatanRepo struct {
	mock.Mock
	// → embed struct mock.Mock dari testify ke dalam MockCatatanRepo
	// → embed artinya semua method dari mock.Mock otomatis tersedia di MockCatatanRepo
	// → method yang didapat: On() untuk setup ekspektasi, Called() untuk catat panggilan,
	// →   AssertExpectations() untuk verifikasi semua ekspektasi terpenuhi,
	// →   AssertNotCalled() untuk verifikasi method tidak pernah dipanggil
}

// Di bawah ini adalah implementasi setiap method dari interface CatatanRepo
// Setiap method mengikuti pola yang sama persis:
// Langkah 1 → panggil m.Called() dengan semua argumen yang diterima
// Langkah 2 → m.Called() mengembalikan args — isinya adalah nilai yang sudah di-setup via .On().Return()
// Langkah 3 → ambil nilai dari args sesuai posisi dan tipe, lalu return
// Kenapa perlu ini: supaya MockCatatanRepo memenuhi interface CatatanRepo dan bisa di-inject ke service

func (m *MockCatatanRepo) Create(ctx context.Context, note *modul.Catatan) (*modul.Catatan, error) {
	// → method Create milik MockCatatanRepo, receiver pointer ke MockCatatanRepo
	// → signature harus identik dengan method Create di interface CatatanRepo
	// → kalau signature berbeda: MockCatatanRepo tidak memenuhi interface → compile error

	args := m.Called(ctx, note)
	// → beritahu mock bahwa method Create dipanggil dengan argumen ctx dan note
	// → testify akan cari ekspektasi yang cocok — yang di-setup via mockRepo.On("Create", ...)
	// → args berisi semua nilai yang sudah di-setup via .Return(...) untuk ekspektasi ini

	if args.Get(0) == nil {
		// → args.Get(0) → ambil return value pertama (index 0) sebagai interface{}
		// → cek nil dulu sebelum type assert — kalau langsung type assert ke *modul.Catatan saat nilnya nil → panic
		// → kapan ini terjadi: saat test setup .Return(nil, someError) untuk simulasi kegagalan
		return nil, args.Error(1)
		// → args.Error(1) → ambil return value kedua (index 1) sebagai error
		// → return nil untuk *modul.Catatan dan error yang sudah di-setup
	}
	return args.Get(0).(*modul.Catatan), args.Error(1)
	// → args.Get(0).(*modul.Catatan) → ambil return value pertama lalu type assert ke *modul.Catatan
	// → type assert aman di sini karena sudah lolos pengecekan nil di atas
	// → args.Error(1) → ambil return value kedua sebagai error
}

func (m *MockCatatanRepo) GetAll(ctx context.Context, arsip *bool, page, limit int) ([]modul.Catatan, int, error) {
	// → implementasi mock untuk method GetAll
	// → menerima filter arsip, nomor halaman, dan jumlah item per halaman

	args := m.Called(ctx, arsip, page, limit)
	// → catat panggilan dengan semua argumen: context, pointer bool arsip, page, limit

	return args.Get(0).([]modul.Catatan), args.Int(1), args.Error(2)
	// → args.Get(0).([]modul.Catatan) → ambil return pertama, type assert ke slice Catatan
	// → tidak perlu cek nil sebelum type assert karena slice kosong []modul.Catatan{} bukan nil
	// → args.Int(1) → ambil return kedua sebagai int — ini total count untuk pagination
	// → args.Error(2) → ambil return ketiga sebagai error
}

func (m *MockCatatanRepo) GetByID(ctx context.Context, id int) (*modul.Catatan, error) {
	// → implementasi mock untuk method GetByID
	// → menerima context dan id catatan yang dicari

	args := m.Called(ctx, id)
	// → catat panggilan dengan argumen context dan id

	if args.Get(0) == nil {
		// → cek nil dulu sebelum type assert — sama seperti di Create
		// → terjadi saat test simulasi "data tidak ditemukan" dengan .Return(nil, apperror.ErrNotFound)
		return nil, args.Error(1)
	}
	return args.Get(0).(*modul.Catatan), args.Error(1)
	// → kalau data ditemukan: return pointer Catatan dan nil error
}

func (m *MockCatatanRepo) Update(ctx context.Context, id int, catatan *modul.Catatan) (*modul.Catatan, error) {
	// → implementasi mock untuk method Update
	// → menerima context, id yang diupdate, dan data baru berupa pointer Catatan

	args := m.Called(ctx, id, catatan)
	// → catat panggilan dengan semua tiga argumen

	if args.Get(0) == nil {
		// → cek nil dulu — terjadi saat test simulasi update gagal karena id tidak ditemukan
		return nil, args.Error(1)
	}
	return args.Get(0).(*modul.Catatan), args.Error(1)
	// → kalau update berhasil: return pointer Catatan yang sudah diupdate dan nil error
}

func (m *MockCatatanRepo) Delete(ctx context.Context, id int) error {
	// → implementasi mock untuk method Delete
	// → hanya return error, tidak ada data yang dikembalikan

	args := m.Called(ctx, id)
	// → catat panggilan dengan argumen context dan id

	return args.Error(0)
	// → args.Error(0) → ambil return value pertama (index 0) sebagai error
	// → tidak perlu cek nil karena tidak ada pointer yang perlu di-type assert
	// → kalau delete sukses: test setup .Return(nil), kalau gagal: .Return(someError)
}

func (m *MockCatatanRepo) SetArsip(ctx context.Context, id int, arsip bool) (*modul.Catatan, error) {
	// → implementasi mock untuk method SetArsip
	// → menerima context, id catatan, dan nilai bool arsip (true = arsipkan, false = buka arsip)

	args := m.Called(ctx, id, arsip)
	// → catat panggilan dengan semua tiga argumen

	if args.Get(0) == nil {
		// → cek nil dulu — terjadi saat test simulasi SetArsip gagal karena id tidak ditemukan
		return nil, args.Error(1)
	}
	return args.Get(0).(*modul.Catatan), args.Error(1)
	// → kalau berhasil: return pointer Catatan dengan nilai Arsip yang sudah diupdate dan nil error
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
	// → sengaja false agar test yang expect Arsip=true bisa dibedakan dengan jelas

	CreatedAt: time.Now(),
	// → isi dengan waktu saat ini — nilainya tidak di-assert di test manapun
	// → hanya untuk memenuhi field struct agar tidak zero value
}

// ===== TEST CREATE =====

func TestCreate_Sukses(t *testing.T) {
	// → nama function harus diawali dengan kata "Test" — aturan Go, bukan konvensi
	// → go test hanya menjalankan function yang namanya diawali "Test"
	// → format penamaan yang dipakai: Test + NamaMethod + _ + Skenario
	// → contoh: TestCreate_Sukses, TestCreate_JudulKosong, TestGetByID_TidakDitemukan
	// → parameter *testing.T wajib ada — dipakai oleh semua fungsi assert untuk report hasil

	// ── ARRANGE ──────────────────────────────────────────────
	// Bagian ini: siapkan semua yang dibutuhkan sebelum test dijalankan

	mockRepo := new(MockCatatanRepo)
	// → buat instance baru MockCatatanRepo
	// → new(T) = alokasikan zero value struct T di heap dan return pointer-nya
	// → hasilnya: *MockCatatanRepo yang siap dipakai
	// → dibuat baru di setiap test agar state mock tidak bocor antar test

	svc := service.NewCatatanService(mockRepo)
	// → buat instance service dengan inject mock repo
	// → ini persis seperti cara main.go inject repo nyata ke service
	// → service tidak tahu apakah ini mock atau implementasi nyata
	// → service hanya tahu bahwa mockRepo memenuhi interface CatatanRepo — itu sudah cukup

	req := dto.CreateCatatanRequest{
		Judul: "belajar golang",
		// → judul valid, tidak kosong
		Isi: "golang adalah bahasa pemrograman",
		// → isi valid, tidak kosong
	}
	// → buat request yang akan dikirim ke service.Create()

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*modul.Catatan")).
		Return(dummyCatatan, nil)
	// → setup ekspektasi mock: "kalau method Create dipanggil dengan argumen ini, return ini"
	// → "Create" → nama method yang diexpect akan dipanggil
	// → mock.Anything → cocokkan argumen pertama (context) dengan nilai apapun
	// →   dipakai untuk context karena kita tidak peduli context spesifik apa yang dipakai
	// → mock.AnythingOfType("*modul.Catatan") → cocokkan argumen kedua dengan tipe *modul.Catatan
	// →   dipakai karena service membuat struct Catatan baru dari req, kita tidak tahu nilai pastinya
	// → .Return(dummyCatatan, nil) → kalau ekspektasi terpenuhi, kembalikan dummyCatatan dan nil error

	// ── ACT ──────────────────────────────────────────────────
	// Bagian ini: jalankan tepat satu function yang sedang dites

	result, err := svc.Create(context.Background(), req)
	// → panggil service.Create() dengan context kosong dan request yang sudah disiapkan
	// → context.Background() = context paling dasar, tidak ada deadline, tidak ada value
	// → cocok untuk test karena kita tidak butuh context khusus

	// ── ASSERT ───────────────────────────────────────────────
	// Bagian ini: verifikasi bahwa hasilnya sesuai ekspektasi

	assert.NoError(t, err)
	// → pastikan err bernilai nil
	// → kalau err tidak nil: test fail dengan pesan "Received unexpected error: ..."

	assert.NotNil(t, result)
	// → pastikan result bukan nil
	// → kalau result nil: test fail dengan pesan "Expected value not to be nil"

	assert.Equal(t, "belajar golang", result.Judul)
	// → pastikan judul di result sama dengan yang diharapkan
	// → format: assert.Equal(t, expected, actual)
	// → kalau tidak sama: test fail dengan pesan yang menunjukkan perbedaan nilainya

	mockRepo.AssertExpectations(t)
	// → verifikasi bahwa semua ekspektasi yang di-setup via .On() benar-benar dipanggil
	// → kalau service.Create() tidak memanggil repo.Create() sama sekali → test fail
	// → ini memastikan service benar-benar berinteraksi dengan repo, bukan skip repo
}

func TestCreate_JudulKosong(t *testing.T) {
	// → test skenario negatif: apa yang terjadi kalau judul tidak diisi

	// ── ARRANGE ──────────────────────────────────────────────
	mockRepo := new(MockCatatanRepo)
	// → buat instance mock baru, terpisah dari test sebelumnya

	svc := service.NewCatatanService(mockRepo)
	// → buat service dengan inject mock repo

	req := dto.CreateCatatanRequest{
		Judul: "",
		// → sengaja kosong untuk memicu validasi di service
		Isi: "isi catatan",
		// → isi diisi agar yang memicu error hanya judul, bukan isi
	}
	// → tidak ada mockRepo.On() di sini — karena kita tidak expect repo dipanggil sama sekali
	// → kalau service benar: dia reject request ini sebelum sampai ke repo

	// ── ACT ──────────────────────────────────────────────────
	result, err := svc.Create(context.Background(), req)
	// → kirim request dengan judul kosong ke service

	// ── ASSERT ───────────────────────────────────────────────
	assert.Error(t, err)
	// → pastikan ada error yang dikembalikan
	// → kalau err nil: test fail — berarti service tidak memvalidasi judul kosong

	assert.Nil(t, result)
	// → pastikan result nil karena operasi gagal
	// → kalau result tidak nil: test fail — service tidak boleh return data saat ada error

	mockRepo.AssertNotCalled(t, "Create")
	// → verifikasi bahwa repo.Create() tidak pernah dipanggil sama sekali
	// → ini adalah assertion terpenting di test ini
	// → membuktikan bahwa validasi judul kosong terjadi di service, sebelum sampai ke repo
	// → kalau repo.Create() terpanggil: berarti service bypass validasi → bug
}

// ===== TEST GET BY ID =====

func TestGetByID_Sukses(t *testing.T) {
	// → test skenario: GetByID dengan ID valid dan data ada di database

	// ── ARRANGE ──────────────────────────────────────────────
	mockRepo := new(MockCatatanRepo)
	svc := service.NewCatatanService(mockRepo)

	mockRepo.On("GetByID", mock.Anything, 1).
		Return(dummyCatatan, nil)
	// → setup ekspektasi: GetByID akan dipanggil dengan context apapun dan id = 1
	// → kalau terpenuhi: return dummyCatatan dan nil error
	// → angka 1 spesifik karena kita ingin pastikan service meneruskan ID yang benar ke repo

	// ── ACT ──────────────────────────────────────────────────
	result, err := svc.GetByID(context.Background(), 1)
	// → panggil service.GetByID() dengan ID = 1

	// ── ASSERT ───────────────────────────────────────────────
	assert.NoError(t, err)
	// → pastikan tidak ada error

	assert.NotNil(t, result)
	// → pastikan data ditemukan dan tidak nil

	assert.Equal(t, 1, result.ID)
	// → pastikan ID di result sama dengan yang diminta
	// → membuktikan service return data yang benar, bukan data acak

	mockRepo.AssertExpectations(t)
	// → verifikasi repo.GetByID() benar-benar dipanggil sekali dengan argumen yang benar
}

func TestGetByID_IDTidakValid(t *testing.T) {
	// → test skenario: GetByID dengan ID = 0 yang tidak valid

	// ── ARRANGE ──────────────────────────────────────────────
	mockRepo := new(MockCatatanRepo)
	svc := service.NewCatatanService(mockRepo)
	// → tidak ada mockRepo.On() — tidak expect repo dipanggil sama sekali

	// ── ACT ──────────────────────────────────────────────────
	result, err := svc.GetByID(context.Background(), 0)
	// → kirim ID = 0 — ID tidak valid karena ID database selalu dimulai dari 1

	// ── ASSERT ───────────────────────────────────────────────
	assert.Error(t, err)
	// → pastikan ada error karena ID tidak valid

	assert.Equal(t, apperror.ErrInvalidID, err)
	// → pastikan error yang dikembalikan adalah tepat ErrInvalidID, bukan error lain
	// → ini yang membuktikan service return sentinel error yang benar sesuai jenis kegagalan
	// → handler menggunakan error ini untuk return HTTP 400 ke client

	assert.Nil(t, result)
	// → pastikan result nil karena operasi gagal

	mockRepo.AssertNotCalled(t, "GetByID")
	// → verifikasi repo.GetByID() tidak pernah dipanggil
	// → service harus tolak ID tidak valid sebelum sampai ke repo
}

func TestGetByID_TidakDitemukan(t *testing.T) {
	// → test skenario: GetByID dengan ID valid tapi data tidak ada di database

	// ── ARRANGE ──────────────────────────────────────────────
	mockRepo := new(MockCatatanRepo)
	svc := service.NewCatatanService(mockRepo)

	mockRepo.On("GetByID", mock.Anything, 99).
		Return(nil, apperror.ErrNotFound)
	// → setup ekspektasi: GetByID dengan id = 99 dikembalikan nil dan ErrNotFound
	// → ini mensimulasikan kondisi: ID valid secara format, tapi tidak ada di database
	// → angka 99 dipilih sebagai ID yang "tidak mungkin ada" di data test

	// ── ACT ──────────────────────────────────────────────────
	result, err := svc.GetByID(context.Background(), 99)
	// → panggil service dengan ID yang tidak ada di database (simulasi)

	// ── ASSERT ───────────────────────────────────────────────
	assert.Error(t, err)
	// → pastikan ada error karena data tidak ditemukan

	assert.Equal(t, apperror.ErrNotFound, err)
	// → pastikan error-nya adalah ErrNotFound
	// → membuktikan service meneruskan ErrNotFound dari repo ke caller dengan benar
	// → handler menggunakan error ini untuk return HTTP 404 ke client

	assert.Nil(t, result)
	// → pastikan result nil karena data tidak ditemukan

	mockRepo.AssertExpectations(t)
	// → verifikasi repo.GetByID() dipanggil sekali dengan ID = 99
}

// ===== TEST DELETE =====

func TestDelete_Sukses(t *testing.T) {
	// → test skenario: Delete dengan ID valid dan data berhasil dihapus

	// ── ARRANGE ──────────────────────────────────────────────
	mockRepo := new(MockCatatanRepo)
	svc := service.NewCatatanService(mockRepo)

	mockRepo.On("Delete", mock.Anything, 1).Return(nil)
	// → setup ekspektasi: Delete dipanggil dengan context apapun dan id = 1
	// → .Return(nil) → simulasi delete berhasil, tidak ada error

	// ── ACT ──────────────────────────────────────────────────
	err := svc.Delete(context.Background(), 1)
	// → panggil service.Delete() — hanya return error, tidak ada data yang dikembalikan

	// ── ASSERT ───────────────────────────────────────────────
	assert.NoError(t, err)
	// → pastikan delete berhasil tanpa error

	mockRepo.AssertExpectations(t)
	// → verifikasi repo.Delete() benar-benar dipanggil
}

func TestDelete_IDTidakValid(t *testing.T) {
	// → test skenario: Delete dengan ID = 0 yang tidak valid

	// ── ARRANGE ──────────────────────────────────────────────
	mockRepo := new(MockCatatanRepo)
	svc := service.NewCatatanService(mockRepo)
	// → tidak ada mockRepo.On() — tidak expect repo dipanggil

	// ── ACT ──────────────────────────────────────────────────
	err := svc.Delete(context.Background(), 0)
	// → kirim ID = 0 yang tidak valid

	// ── ASSERT ───────────────────────────────────────────────
	assert.Error(t, err)
	// → pastikan ada error

	assert.Equal(t, apperror.ErrInvalidID, err)
	// → pastikan error-nya adalah ErrInvalidID
	// → pola yang sama dengan TestGetByID_IDTidakValid

	mockRepo.AssertNotCalled(t, "Delete")
	// → verifikasi repo.Delete() tidak pernah dipanggil
	// → service harus tolak ID tidak valid sebelum sampai ke repo
}

// ===== TEST ARSIP =====

func TestArsip_Sukses(t *testing.T) {
	// → test skenario: mengarsipkan catatan dengan ID valid

	// ── ARRANGE ──────────────────────────────────────────────
	mockRepo := new(MockCatatanRepo)
	svc := service.NewCatatanService(mockRepo)

	arsipCatatan := &modul.Catatan{
		ID: 1,
		// → ID sama dengan yang akan diarsipkan
		Arsip: true,
		// → Arsip = true — ini yang membedakan dari dummyCatatan yang Arsip = false
		// → sengaja buat variabel lokal di sini, bukan pakai dummyCatatan
		// → karena nilai yang di-assert (result.Arsip == true) berbeda dari dummyCatatan
	}

	mockRepo.On("SetArsip", mock.Anything, 1, true).
		Return(arsipCatatan, nil)
	// → setup ekspektasi: SetArsip dipanggil dengan context apapun, id = 1, dan arsip = true
	// → true spesifik karena method Arsip() di service harus meneruskan true ke repo
	// → .Return(arsipCatatan, nil) → return catatan yang sudah diarsipkan dan nil error

	// ── ACT ──────────────────────────────────────────────────
	result, err := svc.Arsip(context.Background(), 1)
	// → panggil service.Arsip() — method khusus untuk mengarsipkan catatan

	// ── ASSERT ───────────────────────────────────────────────
	assert.NoError(t, err)
	// → pastikan tidak ada error

	assert.True(t, result.Arsip)
	// → assert.True(t, x) adalah shorthand untuk assert.Equal(t, true, x)
	// → pastikan catatan yang dikembalikan sudah berstatus arsip (Arsip = true)
	// → ini assertion inti dari test ini — membuktikan Arsip benar-benar berubah jadi true

	mockRepo.AssertExpectations(t)
	// → verifikasi repo.SetArsip() dipanggil dengan argumen yang benar
}

// Cara menjalankan test file ini:
// go test ./internal/service/... -v
// → go test = perintah Go untuk menjalankan semua test
// → ./internal/service/... = path ke semua package di dalam folder internal/service
// → -v = verbose mode, tampilkan nama setiap test dan hasilnya (--- PASS atau --- FAIL)
// → tanpa -v: hanya tampilkan summary akhir, tidak tahu test mana yang pass atau fail

// ## Penjelasan Penting

// **Struktur setiap test selalu AAA:**
// ```
// Arrange → siapkan mock, input, ekspektasi
// Act     → jalankan function yang dites
// Assert  → pastikan hasilnya benar

// package service_test
// ```
// - Test ada **di luar** package, tapi file tetap di folder yang sama
// - Hanya bisa akses yang **exported** (huruf besar) saja
// - Cocok untuk test behavior dari sudut pandang pengguna package

// ---

// ## Kenapa Saya Pakai `service_test`?

// Karena ini **best practice industri** untuk unit test:

// > Test seharusnya tidak peduli **bagaimana** implementasinya — hanya peduli **apakah hasilnya benar.**

// Dengan `service_test`, kamu test service kamu persis seperti cara handler memakainya — hanya lewat interface publik. Kalau kamu refactor internal implementasi, test tidak perlu diubah selama behavior-nya sama.

// ---

// ## Analogi Simpel
// ```
// package service      → kamu masuk ke dapur restoran dan test cara masak
// package service_test → kamu duduk di meja tamu dan test apakah makanannya enak

//jalankan perintah go test ./internal/service/... -v
