package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Upload gambar produk (admin): POST /api/v1/upload
// multipart/form-data, field "file", maks 5MB, MIME harus image/jpeg,
// image/png, atau image/webp (di-sniff dari isi file, bukan dari nama
// atau header Content-Type klien). File disimpan ke ./uploads/ dengan
// nama acak (uuid + ekstensi sesuai MIME) dan mengembalikan URL publik.
const maxUploadSize = 5 << 20 // 5MB

var allowedImageMimes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type UploadHandler struct {
	publicBaseURL string
	uploadDir     string
}

func NewUploadHandler(publicBaseURL, uploadDir string) *UploadHandler {
	return &UploadHandler{publicBaseURL: publicBaseURL, uploadDir: uploadDir}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	// Batasi body agar klien nakal tidak membanjiri parser multipart.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize+(1<<20))
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "file wajib diisi (field 'file', multipart/form-data)")
		return
	}
	defer file.Close()
	if header.Size > maxUploadSize {
		utils.Error(c, http.StatusRequestEntityTooLarge, "ukuran file maksimal 5MB")
		return
	}
	// Baca seluruh isi (maks 5MB) agar MIME di-sniff dari konten asli.
	data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "gagal membaca file")
		return
	}
	if len(data) > maxUploadSize {
		utils.Error(c, http.StatusRequestEntityTooLarge, "ukuran file maksimal 5MB")
		return
	}
	if len(data) == 0 {
		utils.Error(c, http.StatusBadRequest, "file kosong")
		return
	}
	sniffLen := len(data)
	if sniffLen > 512 {
		sniffLen = 512
	}
	mime := http.DetectContentType(data[:sniffLen])
	ext, ok := allowedImageMimes[mime]
	if !ok {
		utils.Error(c, http.StatusBadRequest, "format gambar harus JPEG, PNG, atau WebP")
		return
	}
	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		utils.Error(c, http.StatusInternalServerError, "gagal menyiapkan folder upload")
		return
	}
	name := uuid.NewString() + ext
	path := filepath.Join(h.uploadDir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		utils.Error(c, http.StatusInternalServerError, "gagal menyimpan file")
		return
	}
	utils.Success(c, http.StatusCreated, "file diunggah", gin.H{
		"url":      strings.TrimSuffix(h.publicBaseURL, "/") + "/uploads/" + name,
		"path":     "/uploads/" + name,
		"filename": name,
		"size":     len(data),
		"mime":     mime,
	})
}
