package dto

// package dto → package untuk semua Data Transfer Object
// → diimport oleh handler untuk decode request dan oleh service sebagai parameter input
// → kalau dihapus: handler tidak punya tipe data untuk decode JSON body dari client

// CreateCatatanRequest → struct yang mendefinisikan bentuk data yang masuk dari client saat CREATE
// → hanya berisi field yang boleh dikirim client — id dan created_at tidak ada di sini
// → id tidak ada karena di-generate otomatis oleh MySQL AUTO_INCREMENT
// → created_at tidak ada karena di-generate otomatis oleh MySQL DEFAULT CURRENT_TIMESTAMP
// → arsip tidak ada karena defaultnya false, tidak perlu dikirim client saat create
type CreateCatatanRequest struct {
	Judul string `json:"judul" validate:"required,min=1,max=255"`
	// json:"judul"        → mapping key "judul" dari JSON body ke field Judul
	// validate:"required" → field wajib ada dan tidak boleh kosong string
	// validate:"min=1"    → minimal 1 karakter — kombinasi dengan required untuk cegah string spasi saja
	// validate:"max=255"  → maksimal 255 karakter — sinkron dengan VARCHAR(255) di database
	// → kalau json tag dihapus: Go pakai nama field "Judul" sebagai key, client harus kirim {"Judul": "..."} bukan {"judul": "..."}

	Isi string `json:"isi" validate:"required,min=1"`
	// json:"isi"          → mapping key "isi" dari JSON body ke field Isi
	// validate:"required" → field wajib ada dan tidak boleh kosong
	// validate:"min=1"    → minimal 1 karakter
	// → tidak ada max karena kolom isi di MySQL bertipe TEXT — tidak ada batas panjang praktis
}

// UpdateCatatanRequest → struct yang mendefinisikan bentuk data yang masuk dari client saat UPDATE
// → dipisah dari CreateCatatanRequest agar bisa berkembang sendiri
// → contoh: suatu saat Create butuh field tambahan yang tidak ada di Update, atau sebaliknya
// → kalau digabung jadi satu struct: terpaksa kompromi validate tag untuk dua kebutuhan berbeda
type UpdateCatatanRequest struct {
	Judul string `json:"judul" validate:"omitempty,min=1,max=255"`
	// validate:"omitempty" → kalau field kosong/tidak dikirim, skip semua validasi setelahnya
	// → kalau dikirim, baru validasi min=1 dan max=255
	// → ini partial update — client boleh kirim hanya judul saja tanpa isi, atau sebaliknya
	// → kalau pakai required seperti Create: client wajib kirim semua field setiap update

	Isi string `json:"isi" validate:"omitempty,min=1"`
	// validate:"omitempty" → sama, kalau tidak dikirim skip validasi
	// → kalau dikirim, minimal 1 karakter
}

// Perbedaan required vs omitempty:
// required  → dipakai di Create → field wajib ada dan tidak boleh kosong
// omitempty → dipakai di Update → field boleh kosong, kalau diisi baru divalidasi
//
// Kenapa ini penting:
// → Update adalah partial update — client boleh kirim sebagian field saja
// → Kalau pakai required di Update: client wajib kirim semua field setiap kali update
//   → tidak fleksibel, memaksa client kirim data yang tidak berubah
// → Dengan omitempty: client bisa kirim hanya field yang mau diubah

// Pattern file request DTO:
// 1. Struct terpisah untuk setiap operasi (Create, Update) — jangan digabung
// 2. Hanya field yang boleh dikirim client — id dan timestamps tidak ada di sini
// 3. JSON tag — pastikan nama key sinkron dengan yang dikirim client
// 4. Validate tag — required untuk Create, omitempty untuk Update
// 5. Batas max sinkron dengan constraint di database (VARCHAR(255) → max=255)
