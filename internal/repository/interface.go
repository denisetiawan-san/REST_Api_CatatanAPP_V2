package repository

// package repository → package untuk semua interface dan implementasi repository
// → diimport oleh service untuk depend ke interface, bukan ke implementasi konkret
// → kalau dihapus: service tidak punya kontrak untuk akses database

import (
	"catatan_app/internal/modul"
	// import modul → butuh tipe modul.Catatan dan modul.User sebagai parameter dan return type
	// → kalau dihapus: kompilasi error karena modul.Catatan dan modul.User tidak dikenali

	"context"
	// import context → butuh tipe context.Context sebagai parameter pertama setiap method
	// → context dipakai untuk timeout, cancellation, dan passing nilai antar layer
	// → kalau dihapus: kompilasi error karena context.Context tidak dikenali
)

// CatatanRepo → interface yang mendefinisikan kontrak semua operasi database untuk resource catatan
// → service depend ke interface ini, bukan ke struct CatatanRepository yang konkret
// → keuntungan: service tidak peduli implementasinya MySQL, PostgreSQL, atau memory
// → keuntungan: di unit test service bisa di-mock tanpa butuh database nyata
// → kalau dihapus: service harus depend langsung ke struct konkret — susah di-test dan susah diganti
type CatatanRepo interface {

	// Create → kontrak operasi INSERT catatan baru ke database
	// ctx context.Context → wajib di posisi pertama untuk support timeout dan cancellation
	// note *modul.Catatan → data catatan yang akan disimpan, pointer karena bisa nil
	// return *modul.Catatan → return data lengkap setelah INSERT, termasuk id dan created_at dari DB
	// return error         → kalau INSERT gagal
	Create(ctx context.Context, note *modul.Catatan) (*modul.Catatan, error)

	// GetAll → kontrak operasi SELECT semua catatan dengan filter dan pagination
	// arsip *bool → pointer bool untuk 3 kemungkinan: nil=semua, true=arsip, false=tidak arsip
	//             → kalau pakai bool biasa: tidak bisa bedakan "tidak dikirim" vs "false"
	// page, limit int → parameter pagination untuk LIMIT dan OFFSET di query SQL
	// return []modul.Catatan → slice data catatan sesuai filter dan halaman
	// return int             → total semua data sesuai filter, untuk MetaData pagination
	// return error           → kalau SELECT gagal
	GetAll(ctx context.Context, arsip *bool, page, limit int) ([]modul.Catatan, int, error)

	// GetByID → kontrak operasi SELECT satu catatan berdasarkan id
	// return *modul.Catatan → pointer karena bisa nil kalau tidak ditemukan
	// return error          → ErrNotFound kalau id tidak ada, error lain kalau query gagal
	GetByID(ctx context.Context, id int) (*modul.Catatan, error)

	// Update → kontrak operasi UPDATE catatan berdasarkan id
	// catatan *modul.Catatan → data baru yang akan menggantikan data lama
	// return *modul.Catatan  → return data terbaru setelah UPDATE — client butuh lihat perubahan
	// return error           → ErrNotFound kalau id tidak ada, error lain kalau query gagal
	Update(ctx context.Context, id int, catatan *modul.Catatan) (*modul.Catatan, error)

	// Delete → kontrak operasi DELETE catatan berdasarkan id
	// return error → cukup return error, tidak perlu return data karena data sudah dihapus
	//              → ErrNotFound kalau id tidak ada, error lain kalau query gagal
	Delete(ctx context.Context, id int) error

	// SetArsip → kontrak operasi UPDATE kolom arsip berdasarkan id
	// arsip bool          → true untuk arsipkan, false untuk unarsip
	// return *modul.Catatan → return data terbaru setelah UPDATE — client butuh lihat status arsip baru
	// return error          → ErrNotFound kalau id tidak ada, error lain kalau query gagal
	// → dipisah dari Update karena arsip adalah operasi bisnis tersendiri (PATCH, bukan PUT)
	SetArsip(ctx context.Context, id int, arsip bool) (*modul.Catatan, error)
}

// UserRepo → interface yang mendefinisikan kontrak semua operasi database untuk resource user
// → auth_service depend ke interface ini, bukan ke struct UserRepository yang konkret
// → lebih simpel dari CatatanRepo karena user tidak butuh operasi selengkap catatan
// → tidak ada Update dan Delete karena project ini belum butuh fitur edit atau hapus akun
type UserRepo interface {

	// Create → kontrak operasi INSERT user baru ke database
	// user *modul.User   → data user yang akan disimpan, password sudah di-hash sebelum sampai sini
	// return *modul.User → return data lengkap setelah INSERT, termasuk id dan created_at dari DB
	// return error       → kalau INSERT gagal, termasuk duplicate email
	Create(ctx context.Context, user *modul.User) (*modul.User, error)

	// GetByEmail → kontrak operasi SELECT user berdasarkan email
	// → dipakai auth_service saat login untuk cari user sebelum verifikasi password
	// → return *modul.User lengkap termasuk Password (hash) untuk di-compare dengan bcrypt
	// → return ErrNotFound kalau email tidak terdaftar
	GetByEmail(ctx context.Context, email string) (*modul.User, error)

	// GetByID → kontrak operasi SELECT user berdasarkan id
	// → dipakai auth_service setelah register untuk return data user baru ke client
	// → return ErrNotFound kalau id tidak ada
	GetByID(ctx context.Context, id int) (*modul.User, error)
}

// Kenapa arsip *bool bukan bool biasa di GetAll:
// → bool biasa hanya punya 2 nilai: true atau false
// → *bool punya 3 nilai: nil, true, false
//   nil   = client tidak kirim ?arsip= → tampilkan semua catatan
//   true  = client kirim ?arsip=true  → tampilkan hanya yang diarsip
//   false = client kirim ?arsip=false → tampilkan hanya yang tidak diarsip
// → kalau pakai bool biasa: tidak bisa bedakan "tidak dikirim" dari "false"
//   keduanya akan return false — tidak bisa tampilkan semua catatan

// Pattern file interface repository:
// 1. Satu file untuk semua interface repository — CatatanRepo dan UserRepo dalam satu file
// 2. Setiap method selalu punya ctx context.Context di posisi pertama
// 3. Operasi yang return data → return (pointer/slice, error)
// 4. Operasi yang tidak return data (Delete) → return error saja
// 5. Nama method deskriptif → Create, GetAll, GetByID, Update, Delete, SetArsip
// 6. Interface ini yang di-depend service, bukan struct implementasinya
