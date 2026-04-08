package repository

// package repository → sama dengan catatan_repo.go, satu package untuk semua repository
// → implementasi konkret dari interface UserRepo yang didefinisikan di interface.go

import (
	"catatan_app/internal/apperror"
	// import apperror → butuh ErrNotFound untuk mapping sql.ErrNoRows

	"catatan_app/internal/modul"
	// import modul → butuh tipe modul.User sebagai parameter dan return type

	"context"
	// import context → butuh context.Context sebagai parameter pertama setiap method

	"database/sql"
	// import database/sql → butuh *sql.DB untuk koneksi dan sql.ErrNoRows untuk cek data tidak ada

	"errors"
	// import errors → butuh errors.Is() untuk membandingkan error
)

// UserRepository → struct konkret yang mengimplementasikan interface UserRepo
// → memegang *sql.DB sebagai satu-satunya dependency
// → pola sama persis dengan CatatanRepository — konsistensi di semua repository
type UserRepository struct {
	db *sql.DB
	// → koneksi database di-inject dari luar lewat constructor
	// → tidak dibuat di dalam struct karena DI
}

// NewUserRepository → constructor untuk membuat instance UserRepository
// → menerima *sql.DB dari main.go dan inject ke struct
// → dipanggil di main.go: userRepo := repository.NewUserRepository(db)
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// compile-time check → pastikan UserRepository memenuhi semua method di interface UserRepo
// → kalau Create, GetByEmail, atau GetByID belum diimplementasikan → kompilasi error
// → pola sama dengan CatatanRepository
var _ UserRepo = (*UserRepository)(nil)

// Create → implementasi kontrak Create dari interface UserRepo
// → pintu masuk operasi INSERT user baru ke database
// → password yang masuk ke sini sudah berupa bcrypt hash — di-hash di auth_service sebelum sampai sini
// → repository tidak tahu dan tidak peduli bahwa password sudah di-hash — itu urusan service
func (r *UserRepository) Create(ctx context.Context, user *modul.User) (*modul.User, error) {
	query := `INSERT INTO users (nama, email, password) VALUES (?, ?, ?)`
	// → tidak ada created_at di query karena MySQL isi otomatis via DEFAULT CURRENT_TIMESTAMP
	// → tidak ada id karena AUTO_INCREMENT
	// → tiga ? untuk tiga nilai: nama, email, password

	result, err := r.db.ExecContext(ctx, query, user.Nama, user.Email, user.Password)
	// ExecContext → INSERT tidak return rows, pakai ExecContext bukan QueryContext
	// → urutan parameter sinkron dengan urutan ? di query
	if err != nil {
		return nil, err
		// → kalau email sudah ada di database, MySQL return error duplicate entry
		// → error ini diteruskan ke auth_service yang akan cek dan return ErrEmailSudahDipakai
	}

	id, err := result.LastInsertId()
	// → ambil id yang di-generate MySQL setelah INSERT
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, int(id))
	// → fetch ulang data lengkap dari database
	// → sama dengan CatatanRepository — created_at belum terisi di struct user
	//   nilainya baru ada di database setelah INSERT
}

// GetByEmail → implementasi kontrak GetByEmail dari interface UserRepo
// → dipakai auth_service saat login untuk cari user berdasarkan email
// → return data lengkap termasuk Password (hash) karena service butuh untuk verifikasi bcrypt
// → ini satu-satunya place Password dibaca dari database — hanya untuk keperluan verifikasi
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*modul.User, error) {
	query := `SELECT id, nama, email, password, created_at FROM users WHERE email = ?`
	// → SELECT password ikut diambil karena auth_service butuh untuk bcrypt.CompareHashAndPassword()
	// → setelah verifikasi, password hash tidak pernah dikirim ke client
	//   tidak ada di response DTO manapun

	var u modul.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Nama, &u.Email, &u.Password, &u.CreatedAt,
		// → urutan Scan harus sinkron dengan urutan SELECT di query
		// → id → nama → email → password → created_at
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.ErrNotFound
			// → email tidak ditemukan di database
			// → auth_service terima ErrNotFound dan return ErrEmailAtauPasswordSalah ke handler
			// → kenapa tidak langsung return ErrEmailAtauPasswordSalah di sini:
			//   karena repository tidak boleh tahu tentang bisnis logic auth
			//   itu urusan service untuk interpretasi ErrNotFound menjadi ErrEmailAtauPasswordSalah
		}
		return nil, err
	}

	return &u, nil
}

// GetByID → implementasi kontrak GetByID dari interface UserRepo
// → dipakai auth_service setelah register untuk return data user baru ke client
// → juga dipakai internal oleh Create setelah INSERT untuk fetch data lengkap
func (r *UserRepository) GetByID(ctx context.Context, id int) (*modul.User, error) {
	query := `SELECT id, nama, email, password, created_at FROM users WHERE id = ?`
	// → SELECT password ikut diambil meski tidak selalu dibutuhkan
	// → karena return type adalah *modul.User yang include field Password
	// → password tidak akan dikirim ke client — tidak ada di response DTO manapun

	var u modul.User
	// → deklarasi variabel u bertipe modul.User
	// → akan diisi oleh Scan dari hasil query

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		// QueryRowContext → untuk SELECT yang return tepat satu baris
		// .Scan() → langsung chain — mapping nilai kolom ke field struct
		&u.ID, &u.Nama, &u.Email, &u.Password, &u.CreatedAt,
		// → urutan Scan harus sinkron dengan urutan SELECT di query
		// → id → nama → email → password → created_at
		// → kalau tidak sinkron: data masuk ke field yang salah
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.ErrNotFound
			// sql.ErrNoRows → id tidak ditemukan di database
			// → di-mapping ke apperror.ErrNotFound agar handler bisa return HTTP 404
		}
		return nil, err
		// → error lain berarti masalah koneksi atau query — return as-is ke service
	}

	return &u, nil
	// → return pointer ke struct yang sudah terisi data dari database
	// → pointer karena interface UserRepo mendefinisikan return type *modul.User
}

// Perbandingan UserRepository vs CatatanRepository:
// → UserRepository lebih simpel — hanya 3 method (Create, GetByEmail, GetByID)
// → CatatanRepository lebih lengkap — 6 method (Create, GetAll, GetByID, Update, Delete, SetArsip)
// → karena user tidak butuh Update dan Delete di project ini
// → GetByEmail hanya ada di UserRepository — spesifik untuk kebutuhan auth (login)
// → GetByEmail return Password hash — satu-satunya method yang return Password untuk verifikasi

// Pattern setiap function di user_repo.go:
// 1. Tulis query SQL     → string dengan ? sebagai placeholder
// 2. Eksekusi query      → ExecContext (INSERT) atau QueryRowContext (SELECT)
// 3. Handle hasil        → Scan ke struct, ambil LastInsertId
// 4. Handle error        → mapping sql.ErrNoRows → apperror.ErrNotFound
// 5. Return data         → pointer *modul.User atau error
