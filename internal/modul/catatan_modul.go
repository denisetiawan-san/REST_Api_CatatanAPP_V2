package modul

// → nama package ini adalah "modul"
// → semua domain model — struct yang merepresentasikan data inti aplikasi — dikumpulkan di sini
// → diimport oleh layer lain dengan path: "catatan_app/internal/modul"
// → package name harus sama di semua file dalam satu folder
// → kalau package ini dihapus: seluruh aplikasi tidak bisa dikompilasi karena semua layer pakai tipe ini

import "time"

// → import package time dari standard library Go
// → dibutuhkan karena field CreatedAt bertipe time.Time
// → time.Time adalah tipe Go yang merepresentasikan titik waktu tertentu (tanggal + jam + detik + nanosecond)
// → kalau import ini dihapus: kompilasi error karena time.Time tidak dikenali

// Catatan adalah struct yang merepresentasikan satu baris data di tabel catatan
// → struct = kumpulan field yang membentuk satu unit data, seperti blueprint sebuah objek
// → setiap field di struct ini harus sinkron dengan kolom di tabel catatan di MySQL
// → dipakai oleh semua layer sebagai "bahasa yang sama" untuk data catatan:
// →   repository memakai Catatan untuk hasil scan query database
// →   service memakai Catatan untuk proses business logic
// →   handler memakai Catatan sebagai input mapper ke response DTO
// → tidak ada JSON tag di sini karena struct ini tidak pernah langsung dikirim ke client
// → tidak ada logic apapun karena tugasnya hanya menyimpan data, bukan memproses
type Catatan struct {
	ID int
	// → field ID bertipe int (bilangan bulat)
	// → merepresentasikan kolom id di tabel catatan yang bertipe INT AUTO_INCREMENT
	// → nama field ID (huruf kapital semua) mengikuti konvensi Go untuk singkatan
	// → kalau tipe diganti ke string: rows.Scan() akan error saat runtime karena tipe tidak cocok

	Judul string
	// → field Judul bertipe string
	// → merepresentasikan kolom judul di tabel catatan yang bertipe VARCHAR(255)
	// → string di Go bisa menampung semua nilai VARCHAR dari MySQL

	Isi string
	// → field Isi bertipe string
	// → merepresentasikan kolom isi di tabel catatan yang bertipe TEXT
	// → TEXT di MySQL juga di-scan ke string di Go — tipe Go-nya sama meskipun tipe MySQL berbeda

	Arsip bool
	// → field Arsip bertipe bool (true atau false)
	// → merepresentasikan kolom arsip di tabel catatan yang bertipe BOOLEAN
	// → BOOLEAN di MySQL di-scan ke bool di Go secara otomatis

	CreatedAt time.Time
	// → field CreatedAt bertipe time.Time
	// → merepresentasikan kolom created_at di tabel catatan yang bertipe TIMESTAMP
	// → TIMESTAMP di MySQL bisa di-scan ke time.Time di Go berkat parseTime=true di DSN
	// → tanpa parseTime=true di DSN: scan akan error karena MySQL kirim string, bukan time.Time
	// → kalau field ini dihapus: data created_at tidak bisa dibaca dari database
}

// → semua field exported (huruf kapital) agar bisa diakses dari package lain
// → field unexported (huruf kecil) hanya bisa diakses dalam package yang sama
// → kalau ada field yang tidak sinkron dengan kolom tabel: rows.Scan() error saat runtime
