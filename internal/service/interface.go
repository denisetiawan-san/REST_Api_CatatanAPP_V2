package service

// package service → package untuk semua interface dan implementasi service
// → diimport oleh handler untuk depend ke interface, bukan ke implementasi konkret
// → kalau dihapus: handler tidak punya kontrak untuk panggil business logic

import (
	"catatan_app/internal/dto"
	// import dto → butuh tipe dto.CreateCatatanRequest, dto.UpdateCatatanRequest,
	//              dto.PaginationQuery, dto.RegisterRequest, dto.LoginRequest
	// → service menerima input dari handler dalam bentuk DTO

	"catatan_app/internal/modul"
	// import modul → butuh tipe modul.Catatan dan modul.User sebagai return type
	// → service return domain object ke handler, bukan DTO

	"context"
	// import context → butuh context.Context sebagai parameter pertama setiap method
)

// CatatanSvc → interface yang mendefinisikan kontrak semua business logic untuk resource catatan
// → handler depend ke interface ini, bukan ke struct CatatanService yang konkret
// → keuntungan: di unit test handler bisa di-mock tanpa jalankan business logic nyata
// → kalau dihapus: handler harus depend langsung ke struct konkret — susah di-test
type CatatanSvc interface {

	// Create → kontrak operasi buat catatan baru
	// req dto.CreateCatatanRequest → input dari handler sudah dalam bentuk DTO, bukan JSON raw
	// return *modul.Catatan        → return domain object, handler yang konversi ke response DTO
	// return error                 → ErrBadRequest kalau validasi bisnis gagal
	Create(ctx context.Context, req dto.CreateCatatanRequest) (*modul.Catatan, error)

	// List → kontrak operasi ambil semua catatan dengan filter dan pagination
	// arsip *bool              → pointer untuk 3 kemungkinan: nil=semua, true=arsip, false=tidak arsip
	// pagination dto.PaginationQuery → berisi page dan limit yang sudah divalidasi handler
	// return []modul.Catatan   → slice domain object untuk semua catatan di halaman ini
	// return int               → total semua data di database, diteruskan dari repository ke handler
	//                            handler butuh total untuk isi MetaData.Total di response
	List(ctx context.Context, arsip *bool, pagination dto.PaginationQuery) ([]modul.Catatan, int, error)

	// GetByID → kontrak operasi ambil satu catatan berdasarkan id
	// return *modul.Catatan → nil kalau tidak ditemukan
	// return error          → ErrInvalidID kalau id <= 0, ErrNotFound kalau tidak ada di DB
	GetByID(ctx context.Context, id int) (*modul.Catatan, error)

	// Update → kontrak operasi update catatan berdasarkan id
	// req dto.UpdateCatatanRequest → hanya field yang dikirim client, pakai omitempty
	// return *modul.Catatan        → data terbaru setelah update
	// return error                 → ErrInvalidID, ErrNotFound, atau ErrBadRequest
	Update(ctx context.Context, id int, req dto.UpdateCatatanRequest) (*modul.Catatan, error)

	// Arsip → kontrak operasi arsipkan catatan berdasarkan id
	// → shortcut untuk SetArsip(ctx, id, true) di repository
	// return *modul.Catatan → data terbaru dengan arsip=true
	// return error          → ErrInvalidID atau ErrNotFound
	Arsip(ctx context.Context, id int) (*modul.Catatan, error)

	// Unarsip → kontrak operasi kembalikan catatan dari arsip berdasarkan id
	// → shortcut untuk SetArsip(ctx, id, false) di repository
	// return *modul.Catatan → data terbaru dengan arsip=false
	// return error          → ErrInvalidID atau ErrNotFound
	Unarsip(ctx context.Context, id int) (*modul.Catatan, error)

	// Delete → kontrak operasi hapus catatan berdasarkan id
	// → tidak return data karena data sudah dihapus
	// return error → ErrInvalidID atau ErrNotFound
	Delete(ctx context.Context, id int) error
}

// AuthSvc → interface yang mendefinisikan kontrak business logic untuk autentikasi
// → handler depend ke interface ini, bukan ke struct AuthService yang konkret
// → lebih simpel dari CatatanSvc — hanya dua operasi: register dan login
type AuthSvc interface {

	// Register → kontrak operasi daftar user baru
	// req dto.RegisterRequest → nama, email, password dari client
	// return *modul.User      → data user baru yang berhasil dibuat
	// return error            → ErrEmailSudahDipakai kalau email duplikat
	Register(ctx context.Context, req dto.RegisterRequest) (*modul.User, error)

	// Login → kontrak operasi masuk dengan email dan password
	// req dto.LoginRequest → email dan password dari client
	// return string        → JWT token string kalau login berhasil
	//                        berbeda dari Register yang return *modul.User
	//                        karena login tujuannya dapat token, bukan data user
	// return error         → ErrEmailAtauPasswordSalah kalau credentials salah
	Login(ctx context.Context, req dto.LoginRequest) (string, error)
}

// Perbedaan interface service vs interface repository:
// → interface repository → parameter input adalah domain object (*modul.Catatan)
// → interface service    → parameter input adalah DTO (dto.CreateCatatanRequest)
//   karena service adalah layer yang menerima input dari handler (DTO)
//   dan meneruskan ke repository (domain object)
//   service yang bertanggung jawab konversi dari DTO ke domain object

// Perbedaan CatatanSvc vs AuthSvc:
// → CatatanSvc  → return domain object (*modul.Catatan) — handler yang konversi ke response DTO
// → AuthSvc.Login → return string (JWT token) — tidak return domain object
//   karena tujuan login adalah dapat token, bukan data user

// Pattern file interface service — sama dengan interface repository:
// 1. Satu file untuk semua interface service — CatatanSvc dan AuthSvc dalam satu file
// 2. Setiap method selalu punya ctx context.Context di posisi pertama
// 3. Input dari handler berupa DTO — bukan domain object, bukan raw JSON
// 4. Output ke handler berupa domain object — handler yang konversi ke response DTO
// 5. Interface ini yang di-depend handler, bukan struct implementasinya
