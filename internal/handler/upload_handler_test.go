package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// PNG 1x1 valid.
const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

func newUploadTestRouter(t *testing.T, dir string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewUploadHandler("https://api.alikmakanmie.my.id", dir)
	r := gin.New()
	r.POST("/api/v1/upload", h.Upload)
	return r
}

func multipartRequest(t *testing.T, fieldName, fileName string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestUploadValidPNG(t *testing.T) {
	dir := t.TempDir()
	r := newUploadTestRouter(t, dir)
	png, _ := base64.StdEncoding.DecodeString(tinyPNGBase64)
	req := multipartRequest(t, "file", "foto.png", png)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, _ := body["data"].(map[string]any)
	url, _ := data["url"].(string)
	if url == "" || len(url) < len("https://api.alikmakanmie.my.id/uploads/x.png") {
		t.Fatalf("url tidak valid: %q", url)
	}
	name, _ := data["filename"].(string)
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatalf("file tidak tersimpan: %v", err)
	}
	if data["mime"] != "image/png" {
		t.Fatalf("mime = %v, want image/png", data["mime"])
	}
}

func TestUploadValidJPEGByContent(t *testing.T) {
	dir := t.TempDir()
	r := newUploadTestRouter(t, dir)
	// Header JPEG minimal (FF D8 FF E0 + JFIF) — ekstensi .txt disengaja:
	// validasi harus melihat ISI, bukan nama file.
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01}, bytes.Repeat([]byte{0}, 100)...)
	req := multipartRequest(t, "file", "foto.txt", jpeg)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
}

func TestUploadRejectsNonImage(t *testing.T) {
	dir := t.TempDir()
	r := newUploadTestRouter(t, dir)
	req := multipartRequest(t, "file", "catatan.txt", []byte("ini bukan gambar"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestUploadRejectsOversize(t *testing.T) {
	dir := t.TempDir()
	r := newUploadTestRouter(t, dir)
	big := bytes.Repeat([]byte{0}, (5<<20)+1)
	req := multipartRequest(t, "file", "besar.jpg", big)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413: %s", rec.Code, rec.Body.String())
	}
}

func TestUploadMissingFile(t *testing.T) {
	dir := t.TempDir()
	r := newUploadTestRouter(t, dir)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}
