package handler

// → package handler → semua handler dikumpulkan di package ini
// → diimport oleh router untuk daftarkan ke mux
// → satu-satunya layer yang tahu tentang HTTP — tidak ada logic bisnis di sini

import (
	"catatan_app/internal/apperror"
	// import apperror → butuh semua sentinel error untuk mapping di handleError

	"catatan_app/internal/dto"
	// import dto → butuh request DTO untuk decode body dan response DTO untuk encode response

	"catatan_app/internal/service"
	// import service → butuh service.CatatanSvc (interface) sebagai dependency
	// → depend ke interface, bukan ke struct CatatanService yang konkret

	"encoding/json"
	// import encoding/json → butuh json.NewDecoder untuk decode body dan json.NewEncoder untuk encode response

	"errors"
	// import errors → butuh errors.Is() untuk mapping jenis error di handleError

	"net/http"
	// import net/http → butuh http.ResponseWriter, *http.Request, http.Status*

	"strconv"
	// import strconv → butuh strconv.Atoi untuk konversi id dari string ke int
	//                   dan strconv.ParseBool untuk konversi arsip dari string ke bool
)

// CatatanHandler → struct yang memegang service sebagai dependency via interface
// → satu-satunya layer yang tahu tentang HTTP — decode request dan encode response
// → tidak ada business logic di sini — semua diproses di service
type CatatanHandler struct {
	service service.CatatanSvc
	// → interface CatatanSvc, bukan *CatatanService
	// → di unit test bisa di-inject MockCatatanSvc tanpa business logic nyata
}

// NewCatatanHandler → constructor untuk membuat instance CatatanHandler
// → menerima service.CatatanSvc dari main.go dan inject ke struct
func NewCatatanHandler(s service.CatatanSvc) *CatatanHandler {
	return &CatatanHandler{service: s}
}

// writeJSON → helper untuk tulis response JSON ke client
// → dipanggil di semua handler untuk response sukses maupun error
// → dipisah agar tidak duplikasi 3 baris yang sama di setiap handler
// → juga dipakai oleh auth_handler.go karena satu package
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	// → set header Content-Type dulu SEBELUM WriteHeader
	// → kalau set header setelah WriteHeader: header diabaikan, sudah terlambat

	w.WriteHeader(status)
	// → tulis HTTP status code ke response
	// → hanya boleh dipanggil sekali — kalau dipanggil dua kali, yang kedua diabaikan

	json.NewEncoder(w).Encode(data)
	// → encode data ke JSON dan tulis langsung ke response body
	// → NewEncoder(w) → tulis ke http.ResponseWriter sebagai stream
	// → lebih efisien dari json.Marshal karena tidak perlu alokasi []byte terpisah
}

// writeError → helper untuk tulis response error dalam format JSON standar
// → semua error response punya format yang sama: {"error": "pesan error"}
// → dipisah agar format error konsisten di semua handler
func writeError(w http.ResponseWriter, status int, pesan string) {
	writeJSON(w, status, map[string]string{
		"error": pesan,
		// → key "error" konsisten di semua error response
		// → client selalu tahu cara baca pesan error: response.error
	})
}

// handleError → helper untuk mapping sentinel error ke HTTP status yang tepat
// → dipanggil setiap kali service return error
// → satu tempat untuk semua mapping error → kalau mapping berubah, ubah di satu tempat saja
// → juga dipakai oleh auth_handler.go karena satu package
func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperror.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
		// → ErrNotFound → 404 Not Found
		// → err.Error() → ambil pesan dari sentinel error: "catatan tidak ditemukan"

	case errors.Is(err, apperror.ErrInvalidID):
		writeError(w, http.StatusBadRequest, err.Error())
		// → ErrInvalidID → 400 Bad Request

	case errors.Is(err, apperror.ErrBadRequest):
		writeError(w, http.StatusBadRequest, err.Error())
		// → ErrBadRequest → 400 Bad Request

	default:
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
		// → error tidak dikenal → 500 Internal Server Error
		// → sengaja tidak kirim err.Error() ke client untuk error tidak dikenal
		// → pesan error internal bisa expose informasi sensitif tentang sistem
		// → client cukup tahu "terjadi kesalahan", detail ada di server log
	}
}

// Create → handler untuk POST /api/v1/catatan
// → pintu masuk request CREATE dari client
func (h *CatatanHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateCatatanRequest
	// → deklarasi variabel req bertipe CreateCatatanRequest
	// → akan diisi dari JSON body request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// json.NewDecoder(r.Body) → buat decoder yang baca dari request body
		// .Decode(&req) → parse JSON dan isi ke struct req
		// → kalau JSON rusak atau field tidak cocok → return error
		writeError(w, http.StatusBadRequest, "request tidak valid")
		return
		// → return wajib setelah writeError — kalau tidak, code di bawah tetap jalan
	}

	catatan, err := h.service.Create(r.Context(), req)
	// r.Context() → ambil context dari HTTP request
	// → context berisi deadline, cancellation signal, dan nilai yang di-inject middleware
	// → kalau client disconnect, context cancel → query database di repository ikut cancel
	if err != nil {
		handleError(w, err)
		// → mapping error dari service ke HTTP status yang tepat
		return
	}

	writeJSON(w, http.StatusCreated, dto.ToCatatanResponse(catatan))
	// → 201 Created untuk operasi yang berhasil buat resource baru
	// → ToCatatanResponse → konversi domain object ke response DTO sebelum dikirim ke client
}

// List → handler untuk GET /api/v1/catatan
// → pintu masuk request LIST dari client dengan filter arsip dan pagination
func (h *CatatanHandler) List(w http.ResponseWriter, r *http.Request) {
	var arsip *bool
	// → deklarasi pointer bool, default nil
	// → nil berarti client tidak kirim filter arsip → tampilkan semua

	if q := r.URL.Query().Get("arsip"); q != "" {
		// r.URL.Query().Get("arsip") → ambil nilai query param ?arsip= dari URL
		// → kalau tidak ada → return string kosong ""
		// → kalau ada → masuk ke blok if
		val, err := strconv.ParseBool(q)
		// strconv.ParseBool → konversi string "true"/"false" ke bool
		// → return error kalau bukan "true" atau "false"
		if err != nil {
			writeError(w, http.StatusBadRequest, "nilai arsip tidak valid, gunakan true atau false")
			return
		}
		arsip = &val
		// → ambil alamat memory val untuk dapat *bool
		// → arsip != nil berarti client kirim filter arsip
	}

	page := 1
	// → default page 1 kalau client tidak kirim ?page=
	if q := r.URL.Query().Get("page"); q != "" {
		// → ambil nilai query param ?page= dari URL
		val, err := strconv.Atoi(q)
		// strconv.Atoi → konversi string ke int
		// → return error kalau bukan angka valid
		if err != nil || val < 1 {
			writeError(w, http.StatusBadRequest, "nilai page tidak valid")
			return
		}
		page = val
		// → ganti default 1 dengan nilai dari client
	}

	limit := 10
	// → default limit 10 kalau client tidak kirim ?limit=
	if q := r.URL.Query().Get("limit"); q != "" {
		// → ambil nilai query param ?limit= dari URL
		val, err := strconv.Atoi(q)
		if err != nil || val < 1 {
			writeError(w, http.StatusBadRequest, "nilai limit tidak valid")
			return
		}
		limit = val
		// → handler hanya cek format (< 1) — service yang batasi maksimal 100 (bisnis)
	}

	catatan, total, err := h.service.List(r.Context(), arsip, dto.PaginationQuery{
		Page:  page,
		Limit: limit,
		// → kirim nilai page dan limit yang sudah diparse ke service
	})
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.PaginatedResponse{
		Data: dto.ToCatatanResponses(catatan),
		// → konversi []modul.Catatan ke []CatatanResponse
		// → ToCatatanResponses memanggil ToCatatanResponse untuk setiap item
		Meta: dto.MetaData{
			Page:  page,
			Limit: limit,
			Total: total,
			// → total dari service/repository — jumlah semua data sesuai filter
			// → client pakai total untuk hitung jumlah halaman: ceil(total/limit)
		},
	})
}

// GetByID → handler untuk GET /api/v1/catatan/{id}
// → pintu masuk request GET SATU catatan dari client
func (h *CatatanHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	// r.PathValue("id") → ambil nilai {id} dari path URL — fitur Go 1.22+
	// → return string, perlu dikonversi ke int
	// → kalau URL adalah /api/v1/catatan/abc → strconv.Atoi("abc") return error
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		// → id bukan angka → 400 Bad Request
		return
	}

	catatan, err := h.service.GetByID(r.Context(), id)
	// → serahkan id ke service untuk dicari di database
	if err != nil {
		handleError(w, err)
		// → ErrInvalidID (id<=0) → 400, ErrNotFound → 404, lainnya → 500
		return
	}

	writeJSON(w, http.StatusOK, dto.ToCatatanResponse(catatan))
	// → 200 OK untuk operasi read yang berhasil
	// → ToCatatanResponse → konversi domain object ke response DTO
}

// Update → handler untuk PUT /api/v1/catatan/{id}
// → pintu masuk request UPDATE catatan dari client
func (h *CatatanHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	// r.PathValue("id") → ambil {id} dari path URL
	// strconv.Atoi → konversi string ke int
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		// → id bukan angka → 400 Bad Request
		return
	}

	var req dto.UpdateCatatanRequest
	// → deklarasi variabel req bertipe UpdateCatatanRequest
	// → akan diisi dari JSON body request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// json.NewDecoder(r.Body) → buat decoder yang baca dari request body
		// .Decode(&req) → parse JSON dan isi ke struct req
		// → kalau JSON rusak atau format salah → return error
		writeError(w, http.StatusBadRequest, "request tidak valid")
		return
	}

	catatan, err := h.service.Update(r.Context(), id, req)
	// → serahkan id dan req ke service untuk diproses
	// → service yang fetch data lama, merge dengan nilai baru, lalu update ke database
	if err != nil {
		handleError(w, err)
		// → ErrInvalidID → 400, ErrNotFound → 404, lainnya → 500
		return
	}

	writeJSON(w, http.StatusOK, dto.ToCatatanResponse(catatan))
	// → 200 OK untuk operasi update yang berhasil
	// → ToCatatanResponse → konversi domain object ke response DTO
	// → client terima data terbaru setelah update
}

// Arsip → handler untuk PATCH /api/v1/catatan/{id}/arsip
// → pintu masuk request ARSIP catatan dari client
func (h *CatatanHandler) Arsip(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	// r.PathValue("id") → ambil {id} dari path URL
	// strconv.Atoi → konversi string ke int
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		// → id bukan angka → 400 Bad Request
		return
	}

	catatan, err := h.service.Arsip(r.Context(), id)
	// → serahkan id ke service untuk diarsipkan
	// → service panggil repo.SetArsip(ctx, id, true)
	if err != nil {
		handleError(w, err)
		// → ErrInvalidID → 400, ErrNotFound → 404, lainnya → 500
		return
	}

	writeJSON(w, http.StatusOK, dto.ToCatatanResponse(catatan))
	// → 200 OK untuk operasi arsip yang berhasil
	// → client terima data terbaru dengan arsip=true
}

// Unarsip → handler untuk PATCH /api/v1/catatan/{id}/unarsip
// → pintu masuk request UNARSIP catatan dari client
func (h *CatatanHandler) Unarsip(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	// r.PathValue("id") → ambil {id} dari path URL
	// strconv.Atoi → konversi string ke int
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		// → id bukan angka → 400 Bad Request
		return
	}

	catatan, err := h.service.Unarsip(r.Context(), id)
	// → serahkan id ke service untuk dikembalikan dari arsip
	// → service panggil repo.SetArsip(ctx, id, false)
	if err != nil {
		handleError(w, err)
		// → ErrInvalidID → 400, ErrNotFound → 404, lainnya → 500
		return
	}

	writeJSON(w, http.StatusOK, dto.ToCatatanResponse(catatan))
	// → 200 OK untuk operasi unarsip yang berhasil
	// → client terima data terbaru dengan arsip=false
}

// Delete → handler untuk DELETE /api/v1/catatan/{id}
// → pintu masuk request DELETE catatan dari client
func (h *CatatanHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	// r.PathValue("id") → ambil {id} dari path URL
	// strconv.Atoi → konversi string ke int
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		// → id bukan angka → 400 Bad Request
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		// → service.Delete hanya return error, tidak return data
		// → kalau err != nil → ada masalah, mapping ke HTTP status
		handleError(w, err)
		// → ErrInvalidID → 400, ErrNotFound → 404, lainnya → 500
		return
	}

	w.WriteHeader(http.StatusNoContent)
	// → 204 No Content — berhasil dihapus, tidak ada body response
	// → kenapa tidak writeJSON: karena 204 tidak boleh punya body
	// → client tahu berhasil dari status code 204, tidak perlu pesan tambahan
}

// Perbedaan cara parse input di handler:
// PathValue("id")        → ambil {id} dari path URL: /catatan/{id}
// URL.Query().Get("key") → ambil ?key= dari query string: /catatan?arsip=true
// json.NewDecoder.Decode → ambil JSON dari request body: {"judul": "..."}

// Kenapa writeJSON, writeError, handleError diletakkan di catatan_handler.go:
// → karena dibuat pertama kali di file ini
// → auth_handler.go bisa langsung pakai karena satu package (package handler)
// → tidak perlu import — semua function dalam satu package bisa saling akses

// Pattern setiap function di handler:
// 1. Parse input    → PathValue (id dari URL), Query().Get (query param), Decode (JSON body)
// 2. Validasi input → cek format (strconv error, json decode error) — bukan logik bisnis
// 3. Call service   → serahkan ke service, handler tidak proses apapun sendiri
// 4. Handle error   → handleError() untuk mapping ke HTTP status yang tepat
// 5. Build response → ToCatatanResponse() atau ToCatatanResponses() untuk konversi ke DTO
// 6. Return JSON    → writeJSON() dengan status code yang tepat
