package conectdb

// package conectdb → package khusus untuk koneksi database
// → diimport hanya oleh main.go — satu-satunya yang perlu buat koneksi DB

import (
	"context"
	// import context → butuh context.WithTimeout() untuk batas waktu ping database

	"database/sql"
	// import database/sql → butuh sql.Open() untuk buka koneksi dan *sql.DB sebagai return type

	"errors"
	// import errors → butuh errors.New() untuk buat error kalau DB_DSN tidak di-set

	"os"
	// import os → butuh os.Getenv() untuk baca DB_DSN dari environment variable

	"time"
	// import time → butuh time.Minute untuk konfigurasi connection pool

	_ "github.com/go-sql-driver/mysql"
	// _ (blank import) → import hanya untuk side effect — jalankan init() di package mysql
	// → init() mendaftarkan driver "mysql" ke database/sql
	// → tanpa ini: sql.Open("mysql", ...) return error "unknown driver mysql"
	// → underscore karena kita tidak pakai fungsi apapun dari package ini secara langsung
)

// New → buat koneksi MySQL dan return *sql.DB yang siap dipakai
// → dipanggil sekali di main.go saat startup
// → return (*sql.DB, error) — main.go yang handle kalau error
func New() (*sql.DB, error) {
	dsn := os.Getenv("DB_DSN")
	// → baca DB_DSN dari environment variable
	// → format DSN: user:password@tcp(host:port)/dbname?parseTime=true
	// → contoh: root:deni123@tcp(catatan_mysql:3306)/catatan_app?parseTime=true
	// → parseTime=true → MySQL TIMESTAMP otomatis di-parse ke time.Time Go

	if dsn == "" {
		return nil, errors.New("DB_DSN belum di set")
		// → kalau DB_DSN tidak ada di .env → tolak langsung
		// → tidak ada gunanya lanjut kalau tidak tahu mau connect ke mana
	}

	db, err := sql.Open("mysql", dsn)
	// sql.Open → buat instance *sql.DB dengan driver "mysql" dan DSN
	// → TIDAK langsung buka koneksi ke database — hanya validasi format DSN
	// → koneksi nyata baru dibuat saat query pertama atau saat Ping dipanggil
	if err != nil {
		return nil, err
		// → format DSN tidak valid → return error
	}

	db.SetMaxOpenConns(25)
	// → maksimal 25 koneksi aktif ke database secara bersamaan
	// → kalau semua 25 sedang dipakai: request berikutnya antri
	// → kalau terlalu besar: beban database terlalu tinggi

	db.SetMaxIdleConns(10)
	// → maksimal 10 koneksi idle yang disimpan di pool
	// → koneksi idle = koneksi yang sudah selesai dipakai tapi belum ditutup
	// → disimpan supaya request berikutnya tidak perlu buat koneksi baru dari nol

	db.SetConnMaxLifetime(5 * time.Minute)
	// → maksimal umur satu koneksi adalah 5 menit
	// → setelah 5 menit koneksi ditutup dan diganti baru
	// → mencegah koneksi yang sudah stale atau expired dipakai lagi

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// → buat context dengan timeout 5 detik untuk operasi ping
	// → kalau database tidak merespons dalam 5 detik → ping gagal → return error
	defer cancel()
	// → pastikan context di-cancel setelah function selesai untuk bebaskan resource

	if err := db.PingContext(ctx); err != nil {
		// db.PingContext → baru ini koneksi nyata ke database dibuat dan diverifikasi
		// → kalau database tidak bisa dijangkau, password salah, atau timeout → return error
		_ = db.Close()
		// → tutup db sebelum return error
		// → kalau tidak ditutup: resource koneksi yang sudah dibuat tidak dibebaskan
		// → _ = db.Close() → abaikan error dari Close() karena kita sudah dalam kondisi error
		return nil, err
	}

	return db, nil
	// → return *sql.DB yang sudah terverifikasi — siap dipakai oleh repository
	// → *sql.DB ini yang di-inject ke CatatanRepository dan UserRepository di main.go
}

// Kenapa sql.Open tidak langsung connect tapi PingContext yang connect:
// → sql.Open hanya validasi format DSN dan siapkan struct *sql.DB
// → koneksi nyata dibuat lazy — saat pertama kali dibutuhkan
// → PingContext memaksa koneksi dibuat sekarang untuk verifikasi database bisa dijangkau
// → kalau tidak Ping: server bisa start tapi langsung error saat request pertama masuk

// Kenapa connection pool perlu dikonfigurasi:
// → tanpa konfigurasi: Go buat koneksi baru setiap kali dibutuhkan — lambat dan boros
// → dengan pool: koneksi dibuat sekali, disimpan, dan dipakai ulang — efisien
// → MaxOpenConns proteksi database dari terlalu banyak koneksi sekaligus
// → MaxIdleConns proteksi agar tidak terlalu banyak koneksi idle yang memakan resource
// → ConnMaxLifetime proteksi dari koneksi stale yang sudah tidak valid
