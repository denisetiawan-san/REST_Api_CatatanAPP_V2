package repository_test

// → nama package ini sengaja diberi suffix "_test"
// → artinya ini package terpisah dari package repository yang sedang dites
// → file ini tetap berada di folder internal/repository, tapi Go memperlakukannya sebagai package berbeda
// → konsekuensinya: hanya bisa akses identifier yang exported (huruf kapital) dari package repository
// → tujuannya: memaksa kita test dari sudut pandang pengguna package, bukan dari dalam
// → keuntungannya: kalau kita refactor internal repository, test tidak perlu diubah selama hasilnya sama

import (
	"catatan_app/internal/apperror"
	// → import package apperror yang berisi semua sentinel error
	// → dibutuhkan untuk assert bahwa repo return error yang tepat
	// → contoh: assert.Equal(t, apperror.ErrNotFound, err)

	"catatan_app/internal/modul"
	// → import package modul yang berisi domain struct Catatan
	// → dibutuhkan untuk buat input data yang dikirim ke repo
	// → contoh: &modul.Catatan{Judul: "...", Isi: "..."}

	"catatan_app/internal/repository"
	// → import package yang sedang dites
	// → dibutuhkan untuk panggil repository.NewCatatanRepository() dan buat instance repo

	"context"
	// → import package context bawaan Go
	// → dibutuhkan untuk context.Background() sebagai argumen pertama setiap method repo

	"testing"
	// → import package testing bawaan Go
	// → wajib ada di setiap file test Go
	// → menyediakan tipe *testing.T yang dipakai di setiap function test untuk report hasil

	"time"
	// → import package time bawaan Go
	// → dibutuhkan untuk mengisi kolom created_at di row dummy sqlmock

	"github.com/DATA-DOG/go-sqlmock"
	// → import library sqlmock dari DATA-DOG
	// → menyediakan database SQL tiruan yang bisa diprogram ekspektasinya
	// → tujuan: test repository tanpa butuh database MySQL nyata
	// → sqlmock mencegat semua query SQL dan membalas dengan jawaban yang sudah kita program
	// → tanpa sqlmock: test butuh database nyata → lambat, butuh setup DB, tidak bisa jalan di CI tanpa DB

	"github.com/stretchr/testify/assert"
	// → import package assert dari library testify
	// → menyediakan fungsi assertion yang lebih readable dari cara manual Go
)

// ===== TEST CREATE =====

func TestRepo_Create_Sukses(t *testing.T) {
	// → test skenario: repo Create berhasil menyimpan data baru ke database
	// → yang dites di sini adalah LAYER REPOSITORY, bukan service atau handler
	// → database nyata diganti dengan sqlmock, jadi kita test logika SQL dan mapping data

	// ── ARRANGE ──────────────────────────────────────────────

	db, mock, err := sqlmock.New()
	// → sqlmock.New() → buat koneksi database tiruan dan mock controller-nya
	// → db → *sql.DB tiruan yang akan di-inject ke repository — persis seperti koneksi MySQL nyata
	// → mock → controller untuk setup ekspektasi query dan verifikasi query yang dijalankan
	// → err → error kalau sqlmock gagal dibuat
	// → keduanya selalu dibuat berpasangan — db untuk repo, mock untuk setup ekspektasi

	assert.NoError(t, err)
	// → pastikan sqlmock berhasil dibuat tanpa error sebelum lanjut

	defer db.Close()
	// → tutup koneksi tiruan setelah test function selesai
	// → defer = eksekusi pernyataan ini saat function return, bukan sekarang
	// → penting untuk bebaskan resource meskipun test gagal di tengah jalan

	repo := repository.NewCatatanRepository(db)
	// → buat instance repository dengan inject db tiruan dari sqlmock
	// → repository tidak tahu ini db tiruan atau MySQL nyata
	// → repository hanya tahu bahwa db adalah *sql.DB — itu sudah cukup

	mock.ExpectExec(`INSERT INTO catatan`).
		// → ExpectExec() → program mock untuk expect ada query eksekusi (INSERT/UPDATE/DELETE)
		// → bukan ExpectQuery() karena INSERT tidak mengembalikan rows, hanya result
		// → `INSERT INTO catatan` → substring query yang diharapkan dijalankan
		// → sqlmock akan cocokkan query yang datang dengan substring ini menggunakan regex
		WithArgs("belajar golang", "golang adalah bahasa pemrograman").
		// → WithArgs() → program mock untuk expect query dijalankan dengan argumen ini persis
		// → urutan argumen harus sama dengan urutan ? di query SQL
		// → kalau argumen tidak cocok: test fail dengan pesan "call to ExecQuery was not expected"
		WillReturnResult(sqlmock.NewResult(1, 1))
		// → WillReturnResult() → program mock untuk kembalikan result ini saat query dijalankan
		// → sqlmock.NewResult(lastInsertId, rowsAffected)
		// → lastInsertId = 1 → ID yang baru saja di-insert, dipakai repo untuk fetch data setelahnya
		// → rowsAffected = 1 → jumlah baris yang terpengaruh, menandakan INSERT berhasil

	rows := sqlmock.NewRows([]string{"id", "judul", "isi", "arsip", "created_at"}).
		// → sqlmock.NewRows() → buat tiruan hasil query SELECT
		// → parameter: slice nama kolom yang akan dikembalikan
		// → nama kolom harus sama persis dengan yang di-scan di repo
		AddRow(1, "belajar golang", "golang adalah bahasa pemrograman", false, time.Now())
		// → AddRow() → tambah satu baris data ke hasil query tiruan
		// → urutan nilai harus sama persis dengan urutan kolom di NewRows()
		// → ini yang akan di-scan repo ke struct modul.Catatan

	mock.ExpectQuery(`SELECT id, judul, isi, arsip, created_at FROM catatan WHERE id`).
		// → ExpectQuery() → program mock untuk expect ada query SELECT (yang mengembalikan rows)
		// → repo.Create() menjalankan SELECT setelah INSERT untuk fetch data lengkap yang baru dibuat
		// → `SELECT id, judul...` → substring query yang diharapkan dijalankan
		WithArgs(1).
		// → expect query dijalankan dengan argumen id = 1 (lastInsertId dari INSERT tadi)
		WillReturnRows(rows)
		// → WillReturnRows() → kembalikan rows tiruan yang sudah dibuat di atas

	catatan := &modul.Catatan{
		Judul: "belajar golang",
		// → judul yang akan di-insert
		Isi: "golang adalah bahasa pemrograman",
		// → isi yang akan di-insert
	}
	// → buat input data yang akan dikirim ke repo.Create()
	// → ID tidak diisi karena ID di-generate database, bukan dari kita

	// ── ACT ──────────────────────────────────────────────────
	result, err := repo.Create(context.Background(), catatan)
	// → panggil repo.Create() dengan context kosong dan data catatan
	// → repo akan jalankan INSERT lalu SELECT — keduanya akan dicegat sqlmock

	// ── ASSERT ───────────────────────────────────────────────
	assert.NoError(t, err)
	// → pastikan tidak ada error saat insert

	assert.NotNil(t, result)
	// → pastikan result bukan nil — data berhasil dibuat dan dikembalikan

	assert.Equal(t, 1, result.ID)
	// → pastikan ID di result = 1 sesuai lastInsertId yang di-setup mock
	// → membuktikan repo membaca ID yang benar setelah INSERT

	assert.Equal(t, "belajar golang", result.Judul)
	// → pastikan judul di result sesuai data yang di-insert
	// → membuktikan repo memetakan kolom database ke struct dengan benar

	assert.NoError(t, mock.ExpectationsWereMet())
	// → verifikasi SEMUA ekspektasi yang di-setup via mock.Expect...() benar-benar dijalankan
	// → kalau repo tidak jalankan INSERT atau SELECT yang diharapkan → test fail
	// → ini assertion terpenting di test repo — membuktikan query SQL benar-benar dieksekusi
}

// ===== TEST GET BY ID =====

func TestRepo_GetByID_Sukses(t *testing.T) {
	// → test skenario: repo GetByID berhasil menemukan data dengan ID yang dicari

	// ── ARRANGE ──────────────────────────────────────────────
	db, mock, err := sqlmock.New()
	// → buat koneksi db tiruan dan mock controller baru
	// → dibuat baru di setiap test agar state mock tidak bocor antar test

	assert.NoError(t, err)
	// → pastikan sqlmock berhasil dibuat

	defer db.Close()
	// → tutup koneksi tiruan setelah test selesai

	repo := repository.NewCatatanRepository(db)
	// → buat instance repository dengan db tiruan

	rows := sqlmock.NewRows([]string{"id", "judul", "isi", "arsip", "created_at"}).
		// → buat tiruan hasil query SELECT dengan kolom yang sesuai
		AddRow(1, "belajar golang", "golang adalah bahasa pemrograman", false, time.Now())
		// → tambah satu baris — simulasi data yang ditemukan di database

	mock.ExpectQuery(`SELECT id, judul, isi, arsip, created_at FROM catatan WHERE id`).
		// → program mock untuk expect query SELECT dengan filter WHERE id
		WithArgs(1).
		// → expect query dijalankan dengan argumen id = 1
		WillReturnRows(rows)
		// → kalau query dijalankan dengan argumen yang cocok, kembalikan rows ini

	// ── ACT ──────────────────────────────────────────────────
	result, err := repo.GetByID(context.Background(), 1)
	// → panggil repo.GetByID() dengan id = 1

	// ── ASSERT ───────────────────────────────────────────────
	assert.NoError(t, err)
	// → pastikan tidak ada error — data ditemukan

	assert.NotNil(t, result)
	// → pastikan result bukan nil

	assert.Equal(t, 1, result.ID)
	// → pastikan ID di result sesuai yang dicari
	// → membuktikan repo memetakan kolom id ke field ID dengan benar

	assert.NoError(t, mock.ExpectationsWereMet())
	// → verifikasi query SELECT benar-benar dijalankan oleh repo
}

func TestRepo_GetByID_TidakDitemukan(t *testing.T) {
	// → test skenario: repo GetByID dengan ID yang tidak ada di database

	// ── ARRANGE ──────────────────────────────────────────────
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := repository.NewCatatanRepository(db)

	mock.ExpectQuery(`SELECT id, judul, isi, arsip, created_at FROM catatan WHERE id`).
		// → program mock untuk expect query SELECT dengan filter WHERE id
		WithArgs(99).
		// → expect query dijalankan dengan argumen id = 99
		WillReturnRows(sqlmock.NewRows([]string{}))
		// → sqlmock.NewRows([]string{}) → buat rows kosong tanpa kolom apapun
		// → rows kosong = tidak ada data yang ditemukan
		// → mensimulasikan kondisi: SQL query berhasil dijalankan tapi tidak ada hasil

	// ── ACT ──────────────────────────────────────────────────
	result, err := repo.GetByID(context.Background(), 99)
	// → panggil repo.GetByID() dengan id = 99 yang tidak ada

	// ── ASSERT ───────────────────────────────────────────────
	assert.Error(t, err)
	// → pastikan ada error karena data tidak ditemukan

	assert.Equal(t, apperror.ErrNotFound, err)
	// → pastikan error-nya adalah ErrNotFound, bukan error SQL generik
	// → ini yang membuktikan repo memetakan "rows kosong" ke sentinel error ErrNotFound
	// → handler menggunakan ErrNotFound ini untuk return HTTP 404 ke client

	assert.Nil(t, result)
	// → pastikan result nil karena data tidak ditemukan

	assert.NoError(t, mock.ExpectationsWereMet())
	// → verifikasi query SELECT tetap dijalankan meskipun hasilnya kosong
}

// ===== TEST DELETE =====

func TestRepo_Delete_Sukses(t *testing.T) {
	// → test skenario: repo Delete berhasil menghapus data yang ada

	// ── ARRANGE ──────────────────────────────────────────────
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := repository.NewCatatanRepository(db)

	mock.ExpectExec(`DELETE FROM catatan WHERE id`).
		// → ExpectExec() karena DELETE tidak mengembalikan rows, hanya result
		// → `DELETE FROM catatan WHERE id` → substring query yang diharapkan
		WithArgs(1).
		// → expect query dijalankan dengan argumen id = 1
		WillReturnResult(sqlmock.NewResult(0, 1))
		// → sqlmock.NewResult(lastInsertId, rowsAffected)
		// → lastInsertId = 0 → DELETE tidak menghasilkan ID baru, jadi 0
		// → rowsAffected = 1 → satu baris berhasil dihapus, menandakan data ditemukan dan dihapus

	// ── ACT ──────────────────────────────────────────────────
	err = repo.Delete(context.Background(), 1)
	// → panggil repo.Delete() dengan id = 1
	// → hanya return error, tidak ada data yang dikembalikan

	// ── ASSERT ───────────────────────────────────────────────
	assert.NoError(t, err)
	// → pastikan delete berhasil tanpa error

	assert.NoError(t, mock.ExpectationsWereMet())
	// → verifikasi query DELETE benar-benar dijalankan oleh repo
}

func TestRepo_Delete_TidakDitemukan(t *testing.T) {
	// → test skenario: repo Delete dengan ID yang tidak ada di database

	// ── ARRANGE ──────────────────────────────────────────────
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := repository.NewCatatanRepository(db)

	mock.ExpectExec(`DELETE FROM catatan WHERE id`).
		// → program mock untuk expect query DELETE
		WithArgs(99).
		// → expect query dijalankan dengan argumen id = 99
		WillReturnResult(sqlmock.NewResult(0, 0))
		// → rowsAffected = 0 → tidak ada baris yang terhapus
		// → ini yang membedakan dari Delete_Sukses yang rowsAffected = 1
		// → artinya: query DELETE berhasil dijalankan tapi tidak ada data dengan id = 99

	// ── ACT ──────────────────────────────────────────────────
	err = repo.Delete(context.Background(), 99)
	// → panggil repo.Delete() dengan id = 99 yang tidak ada

	// ── ASSERT ───────────────────────────────────────────────
	assert.Error(t, err)
	// → pastikan ada error karena data tidak ditemukan

	assert.Equal(t, apperror.ErrNotFound, err)
	// → pastikan error-nya adalah ErrNotFound
	// → ini yang membuktikan repo memetakan "rowsAffected = 0" ke sentinel error ErrNotFound
	// → rowsAffected = 0 artinya tidak ada baris yang terhapus = data tidak ada = Not Found

	assert.NoError(t, mock.ExpectationsWereMet())
	// → verifikasi query DELETE tetap dijalankan meskipun tidak ada data yang terhapus
}

// ===== TEST GET ALL =====

func TestRepo_GetAll_Sukses(t *testing.T) {
	// → test skenario: repo GetAll berhasil mengambil semua data dengan pagination

	// ── ARRANGE ──────────────────────────────────────────────
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := repository.NewCatatanRepository(db)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	// → buat tiruan hasil query COUNT
	// → sqlmock.NewRows([]string{"count"}) → satu kolom bernama "count"
	// → AddRow(2) → nilai count = 2, artinya total ada 2 catatan di database
	// → dipakai untuk mengisi field Total di PaginatedResponse

	mock.ExpectQuery(`SELECT COUNT`).
		// → program mock untuk expect query SELECT COUNT yang dijalankan repo sebelum SELECT data
		// → repo.GetAll() menjalankan dua query: COUNT dulu untuk total, baru SELECT untuk data
		WillReturnRows(countRows)
		// → kembalikan countRows saat query COUNT dijalankan
		// → tidak ada WithArgs() karena query COUNT tidak punya filter spesifik di test ini

	rows := sqlmock.NewRows([]string{"id", "judul", "isi", "arsip", "created_at"}).
		// → buat tiruan hasil query SELECT data
		// → kolom harus sama persis dengan yang di-scan di repo.GetAll()
		AddRow(1, "catatan 1", "isi 1", false, time.Now()).
		// → tambah baris pertama
		AddRow(2, "catatan 2", "isi 2", false, time.Now())
		// → tambah baris kedua
		// → dua baris sesuai dengan total count = 2

	mock.ExpectQuery(`SELECT id, judul, isi, arsip, created_at FROM catatan`).
		// → program mock untuk expect query SELECT data setelah COUNT
		WillReturnRows(rows)
		// → kembalikan rows dua baris saat query SELECT dijalankan
		// → tidak ada WithArgs() karena tidak ada filter arsip di test ini (arsip = nil)

	// ── ACT ──────────────────────────────────────────────────
	result, total, err := repo.GetAll(context.Background(), nil, 1, 10)
	// → panggil repo.GetAll() dengan:
	// → context.Background() → context kosong
	// → nil → filter arsip = nil, artinya ambil semua (aktif dan arsip)
	// → 1 → nomor halaman pertama
	// → 10 → maksimal 10 item per halaman

	// ── ASSERT ───────────────────────────────────────────────
	assert.NoError(t, err)
	// → pastikan tidak ada error saat ambil data

	assert.Equal(t, 2, len(result))
	// → pastikan jumlah data yang dikembalikan = 2
	// → len(result) → hitung panjang slice hasil query
	// → membuktikan repo mengembalikan semua baris yang ada

	assert.Equal(t, 2, total)
	// → pastikan total count = 2 sesuai yang dikembalikan query COUNT
	// → total dipakai handler untuk mengisi MetaData.Total di response pagination
	// → membuktikan repo membaca hasil COUNT dengan benar

	assert.NoError(t, mock.ExpectationsWereMet())
	// → verifikasi KEDUA query (COUNT dan SELECT) benar-benar dijalankan oleh repo
	// → kalau salah satu tidak dijalankan → test fail
	// → urutan ekspektasi juga diverifikasi — COUNT harus dijalankan sebelum SELECT
}

// Cara menjalankan test file ini:
// go test ./internal/repository/... -v
// → go test = perintah Go untuk menjalankan semua test
// → ./internal/repository/... = path ke semua package di dalam folder internal/repository
// → -v = verbose mode, tampilkan nama setiap test dan hasilnya (--- PASS atau --- FAIL)
// → tanpa -v: hanya tampilkan summary akhir, tidak tahu test mana yang pass atau fail

// Cara menjalankan SEMUA test di seluruh project sekaligus:
// go test ./... -v
// → ./... = jalankan semua package yang punya file _test.go di seluruh project
// → berguna untuk verifikasi semua layer (service, handler, repository) lulus test sekaligus
