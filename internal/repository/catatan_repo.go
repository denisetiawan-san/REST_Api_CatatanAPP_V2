package repository

// package repository → sama dengan interface.go, satu package untuk semua repository
// → implementasi konkret dari interface CatatanRepo yang didefinisikan di interface.go

import (
	"catatan_app/internal/apperror"
	// import apperror → butuh ErrNotFound untuk mapping sql.ErrNoRows
	// → kalau dihapus: tidak bisa return error standar yang dimengerti handler

	"catatan_app/internal/modul"
	// import modul → butuh tipe modul.Catatan sebagai parameter dan return type
	// → kalau dihapus: kompilasi error karena modul.Catatan tidak dikenali

	"context"
	// import context → butuh context.Context sebagai parameter pertama setiap method
	// → kalau dihapus: kompilasi error karena context.Context tidak dikenali

	"database/sql"
	// import database/sql → butuh *sql.DB untuk koneksi dan sql.ErrNoRows untuk cek data tidak ada
	// → kalau dihapus: kompilasi error karena sql.DB dan sql.ErrNoRows tidak dikenali

	"errors"
	// import errors → butuh errors.Is() untuk membandingkan error
	// → kalau dihapus: tidak bisa cek apakah error adalah sql.ErrNoRows
)

// CatatanRepository → struct konkret yang mengimplementasikan interface CatatanRepo
// → memegang *sql.DB sebagai satu-satunya dependency
// → semua method di struct ini punya akses ke koneksi database via r.db
type CatatanRepository struct {
	db *sql.DB
	// db *sql.DB → koneksi database yang di-inject dari luar lewat constructor
	// → tidak dibuat di dalam struct karena DI — dependency tidak boleh dibuat sendiri
	// → *sql.DB sudah include connection pool — tidak perlu buat koneksi baru setiap query
}

// NewCatatanRepository → constructor untuk membuat instance CatatanRepository
// → menerima *sql.DB dari luar (dari main.go) dan inject ke struct
// → ini adalah Dependency Injection — koneksi DB tidak dibuat di sini, hanya diterima
// → kalau dihapus: main.go tidak bisa buat instance CatatanRepository dengan cara yang benar
func NewCatatanRepository(db *sql.DB) *CatatanRepository {
	return &CatatanRepository{db: db}
}

// compile-time check → pastikan CatatanRepository memenuhi semua method di interface CatatanRepo
// → kalau ada method di CatatanRepo yang belum diimplementasikan → kompilasi error
// → kalau dihapus: error baru ketahuan saat runtime, bukan saat kompilasi
// → (*CatatanRepository)(nil) → buat pointer nil ke CatatanRepository, tidak alokasi memory
// → var _ → underscore berarti variabel ini tidak dipakai, hanya untuk trigger compile-time check
var _ CatatanRepo = (*CatatanRepository)(nil)

// Create → implementasi kontrak Create dari interface CatatanRepo
// → pintu masuk operasi INSERT catatan baru ke database
// → menerima domain object dari service, bukan request DTO — repository tidak tahu tentang HTTP
func (r *CatatanRepository) Create(ctx context.Context, note *modul.Catatan) (*modul.Catatan, error) {
	query := `INSERT INTO catatan (judul, isi, arsip, created_at) VALUES (?, ?, false, NOW())`
	// ? sebagai placeholder → anti SQL injection — nilai tidak disisipkan langsung ke string query
	// false → arsip default false saat create, tidak perlu dikirim dari service
	// NOW() → created_at diisi database langsung, bukan dari Go — agar timezone konsisten

	result, err := r.db.ExecContext(ctx, query, note.Judul, note.Isi)
	// ExecContext → dipakai untuk INSERT, UPDATE, DELETE — operasi yang tidak return rows
	// ctx → kalau context cancel/timeout, query langsung dihentikan
	// note.Judul, note.Isi → nilai yang menggantikan ? di query secara berurutan
	if err != nil {
		return nil, err
		// → return nil, err — tidak wrap error karena ini error teknis database, bukan error bisnis
	}

	id, err := result.LastInsertId()
	// LastInsertId → ambil id yang di-generate MySQL setelah INSERT AUTO_INCREMENT
	// → return int64, di-cast ke int saat dipakai
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, int(id))
	// → fetch ulang data lengkap dari database berdasarkan id baru
	// → kenapa tidak langsung return note: karena created_at belum terisi
	//   note.CreatedAt masih zero value — nilai aslinya ada di database
	// → dengan fetch ulang: response selalu akurat sesuai data di database
}

// SetArsip → implementasi kontrak SetArsip dari interface CatatanRepo
// → pintu masuk operasi UPDATE kolom arsip berdasarkan id
// → dipakai untuk operasi arsip (true) dan unarsip (false)
func (r *CatatanRepository) SetArsip(ctx context.Context, id int, arsip bool) (*modul.Catatan, error) {
	result, err := r.db.ExecContext(
		ctx,
		"UPDATE catatan SET arsip = ? WHERE id = ?",
		arsip, id,
		// arsip → nilai bool yang menggantikan ? pertama
		// id    → nilai int yang menggantikan ? kedua
		// → urutan parameter harus sinkron dengan urutan ? di query
	)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	// RowsAffected → berapa baris yang terpengaruh oleh UPDATE
	// → kalau 0: berarti WHERE id = ? tidak menemukan data — id tidak ada di database
	// → kalau 1: berarti UPDATE berhasil
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, apperror.ErrNotFound
		// → mapping ke ErrNotFound kalau id tidak ada
		// → handler akan mapping ErrNotFound ke HTTP 404
	}

	return r.GetByID(ctx, id)
	// → fetch ulang data lengkap setelah update
	// → client butuh lihat status arsip terbaru di response
}

// GetAll → implementasi kontrak GetAll dari interface CatatanRepo
// → pintu masuk operasi SELECT semua catatan dengan filter arsip dan pagination
// → return tiga nilai: data, total, error — total untuk MetaData pagination
func (r *CatatanRepository) GetAll(ctx context.Context, arsip *bool, page, limit int) ([]modul.Catatan, int, error) {
	query := `SELECT id, judul, isi, arsip, created_at FROM catatan`
	// → query dasar tanpa WHERE — akan ditambah kondisi secara dinamis

	countQuery := `SELECT COUNT(*) FROM catatan`
	// → query terpisah untuk hitung total data
	// → kenapa dua query: query utama pakai LIMIT/OFFSET — tidak bisa dipakai untuk hitung total
	//   COUNT(*) butuh semua data tanpa LIMIT untuk hasil yang akurat

	args := []interface{}{}
	// → slice untuk tampung semua parameter query secara dinamis
	// → interface{} karena parameter bisa bermacam tipe (bool, int)
	// → dipakai untuk kedua query: countQuery dan query utama

	if arsip != nil {
		query += " WHERE arsip = ?"
		countQuery += " WHERE arsip = ?"
		args = append(args, *arsip)
		// *arsip → dereference pointer untuk ambil nilai bool-nya
		// → kalau arsip = &true  → WHERE arsip = true  → ambil yang diarsip
		// → kalau arsip = &false → WHERE arsip = false → ambil yang tidak diarsip
	} else {
		query += " WHERE arsip = false"
		countQuery += " WHERE arsip = false"
		// → kalau arsip nil (tidak dikirim client) → default tampilkan yang tidak diarsip
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
		// QueryRowContext → untuk SELECT yang return tepat satu baris
		// Scan(&total)    → isi variabel total dengan hasil COUNT(*)
		// args...         → spread slice args sebagai variabel argument
	}

	offset := (page - 1) * limit
	// → rumus OFFSET untuk pagination
	// → halaman 1: (1-1)*10 = 0  → mulai dari data ke-1
	// → halaman 2: (2-1)*10 = 10 → mulai dari data ke-11
	// → halaman 3: (3-1)*10 = 20 → mulai dari data ke-21

	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	// → LIMIT  → berapa data yang diambil per halaman
	// → OFFSET → mulai dari data ke berapa
	// → ditambah setelah countQuery dieksekusi karena countQuery tidak butuh LIMIT/OFFSET

	rows, err := r.db.QueryContext(ctx, query, args...)
	// QueryContext → untuk SELECT yang return banyak baris
	// → berbeda dengan QueryRowContext yang hanya return satu baris
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	// defer rows.Close() → pastikan rows ditutup setelah function selesai
	// → kalau tidak ditutup: koneksi dari pool tidak dikembalikan → connection leak
	// → defer memastikan Close() dipanggil meski ada error di tengah jalan

	var catatan []modul.Catatan
	for rows.Next() {
		// rows.Next() → maju ke baris berikutnya, return false kalau sudah habis
		var n modul.Catatan
		if err := rows.Scan(&n.ID, &n.Judul, &n.Isi, &n.Arsip, &n.CreatedAt); err != nil {
			return nil, 0, err
			// Scan → mapping nilai kolom ke field struct secara berurutan
			// → urutan &n.ID, &n.Judul, dst harus sinkron dengan urutan SELECT di query
			// → kalau tidak sinkron: data masuk ke field yang salah
		}
		catatan = append(catatan, n)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
		// rows.Err() → cek error yang terjadi selama iterasi rows.Next()
		// → berbeda dengan error dari QueryContext — ini error saat membaca baris
		// → kalau tidak dicek: error saat iterasi akan diabaikan
	}

	if catatan == nil {
		catatan = []modul.Catatan{}
		// → kalau tidak ada data, return slice kosong bukan nil
		// → nil di-encode JSON menjadi null: {"data": null}
		// → slice kosong di-encode JSON menjadi array kosong: {"data": []}
		// → client lebih mudah handle array kosong daripada null
	}

	return catatan, total, nil
}

// GetByID → implementasi kontrak GetByID dari interface CatatanRepo
// → pintu masuk operasi SELECT satu catatan berdasarkan id
// → dipakai langsung oleh service dan dipanggil internal oleh Create, Update, SetArsip
func (r *CatatanRepository) GetByID(ctx context.Context, id int) (*modul.Catatan, error) {
	query := `SELECT id, judul, isi, arsip, created_at FROM catatan WHERE id = ?`

	var n modul.Catatan
	err := r.db.QueryRowContext(ctx, query, id).Scan(&n.ID, &n.Judul, &n.Isi, &n.Arsip, &n.CreatedAt)
	// QueryRowContext → untuk SELECT yang return tepat satu baris
	// .Scan() → langsung chain dari QueryRowContext — tidak perlu variable rows terpisah
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.ErrNotFound
			// sql.ErrNoRows → error dari database/sql kalau SELECT tidak menemukan baris
			// → di-mapping ke apperror.ErrNotFound agar handler bisa return HTTP 404
			// → ini satu-satunya tempat sql.ErrNoRows dikonversi — layer di atas tidak tahu sql.ErrNoRows
		}
		return nil, err
		// → error lain berarti masalah koneksi atau query — return as-is ke service
	}

	return &n, nil
	// → return pointer ke struct — bukan nilai langsung
	// → pointer karena interface CatatanRepo mendefinisikan return type *modul.Catatan
}

// Update → implementasi kontrak Update dari interface CatatanRepo
// → pintu masuk operasi UPDATE judul dan isi catatan berdasarkan id
func (r *CatatanRepository) Update(ctx context.Context, id int, catatan *modul.Catatan) (*modul.Catatan, error) {
	result, err := r.db.ExecContext(
		ctx,
		"UPDATE catatan SET judul = ?, isi = ? WHERE id = ?",
		catatan.Judul, catatan.Isi, id,
		// → urutan parameter harus sinkron dengan urutan ? di query
		// → catatan.Judul → menggantikan ? pertama (SET judul)
		// → catatan.Isi   → menggantikan ? kedua (SET isi)
		// → id            → menggantikan ? ketiga (WHERE id)
		// → id di posisi terakhir karena di query id ada di WHERE, bukan SET
	)
	// ExecContext → dipakai untuk UPDATE karena tidak return rows, return sql.Result
	if err != nil {
		return nil, err
		// → query gagal — error teknis database, return as-is ke service
	}

	rowsAffected, err := result.RowsAffected()
	// RowsAffected → berapa baris yang terpengaruh oleh UPDATE
	// → kalau 0: WHERE id = ? tidak menemukan data — id tidak ada di database
	// → kalau 1: UPDATE berhasil
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, apperror.ErrNotFound
		// → 0 rows affected berarti id tidak ada di database
		// → handler akan mapping ErrNotFound ke HTTP 404
	}

	return r.GetByID(ctx, id)
	// → fetch ulang data lengkap setelah UPDATE
	// → kenapa tidak langsung return catatan yang diterima:
	//   catatan yang diterima adalah hasil merge dari service — belum tentu akurat dengan DB
	// → dengan fetch ulang: response selalu fresh dari database
}

// Delete → implementasi kontrak Delete dari interface CatatanRepo
// → pintu masuk operasi DELETE catatan berdasarkan id
// → return hanya error, tidak return data karena data sudah dihapus
func (r *CatatanRepository) Delete(ctx context.Context, id int) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM catatan WHERE id = ?", id)
	// ExecContext → dipakai untuk DELETE karena tidak return rows
	// → "DELETE FROM catatan WHERE id = ?" → hapus baris dengan id yang sesuai
	// → id → menggantikan ? di query
	if err != nil {
		return err
		// → query gagal — error teknis database, return as-is ke service
	}

	rowsAffected, err := result.RowsAffected()
	// RowsAffected → berapa baris yang terhapus
	// → kalau 0: WHERE id = ? tidak menemukan data — id tidak ada di database
	// → kalau 1: DELETE berhasil
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return apperror.ErrNotFound
		// → 0 rows affected berarti id tidak ada di database
		// → handler akan mapping ErrNotFound ke HTTP 404
	}

	return nil
	// → return nil berarti DELETE berhasil
	// → tidak ada data yang dikembalikan — data sudah dihapus
	// → handler return HTTP 204 No Content — tidak ada body response
}

// Perbedaan ExecContext vs QueryContext vs QueryRowContext:
// ExecContext    → INSERT, UPDATE, DELETE — tidak return rows, return sql.Result
//                  sql.Result punya LastInsertId() dan RowsAffected()
// QueryContext   → SELECT banyak baris — return *sql.Rows, perlu di-iterasi dan di-Close
// QueryRowContext → SELECT satu baris — return *sql.Row, langsung .Scan()

// Pattern setiap function di repository:
// 1. Tulis query SQL    → string dengan ? sebagai placeholder, bukan string concatenation
// 2. Eksekusi query     → ExecContext (INSERT/UPDATE/DELETE) atau QueryContext (SELECT banyak)
//                         atau QueryRowContext (SELECT satu)
// 3. Handle hasil       → Scan rows, ambil LastInsertId(), cek RowsAffected()
// 4. Handle error       → mapping sql.ErrNoRows → apperror.ErrNotFound
// 5. Return data        → pointer untuk satu data, slice untuk banyak data, nil error kalau berhasil
