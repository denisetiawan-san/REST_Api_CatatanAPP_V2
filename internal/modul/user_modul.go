package modul

// package modul → sama dengan catatan_modul.go, satu package untuk semua domain model
// → diimport layer lain dengan: import "catatan_app/internal/modul"

import "time"

// import "time" → butuh tipe time.Time untuk field CreatedAt
// → kalau dihapus: kompilasi error karena time.Time tidak dikenali

// User → struct domain yang merepresentasikan tabel users di database
// → dipakai oleh user_repo.go untuk hasil query database
// → dipakai oleh auth_service.go sebagai objek yang diproses
// → tidak ada JSON tag karena ini murni domain object, bukan response API
// → tidak ada logic apapun karena tugasnya hanya menyimpan data
// → field Password ada di sini karena repository butuh simpan dan baca hash dari DB
// → tapi Password tidak pernah keluar ke client — tidak ada di response DTO
type User struct {
	ID        int       // → int karena kolom id di MySQL bertipe INT AUTO_INCREMENT
	Nama      string    // → string karena kolom nama di MySQL bertipe VARCHAR(100)
	Email     string    // → string karena kolom email di MySQL bertipe VARCHAR(100) UNIQUE
	Password  string    // → string karena kolom password di MySQL bertipe VARCHAR(255) — isinya bcrypt hash, bukan plain text
	CreatedAt time.Time // → time.Time karena kolom created_at di MySQL bertipe TIMESTAMP
	// → semua field harus sinkron dengan kolom tabel
	// → kalau tipe tidak cocok, rows.Scan() akan error saat runtime
}

// Pattern file domain model — sama persis dengan catatan_modul.go:
// 1. Definisi struct dengan nama resource
// 2. Field merepresentasikan kolom tabel — tipe data harus sinkron dengan MySQL
// 3. Tidak ada JSON tag — domain object tidak pernah langsung dikirim ke client
// 4. Tidak ada logic apapun — validasi di service, query di repository, HTTP di handler
//
// Perbedaan User vs Catatan:
// → User punya field Password — disimpan sebagai bcrypt hash di database
// → Password ada di domain model karena repository perlu baca nilainya untuk verifikasi login
// → Password tidak pernah masuk ke response DTO — client tidak boleh dan tidak perlu tahu
