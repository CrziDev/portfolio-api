package api

import (
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxUpload = 5 << 20

// Allowed image types, by sniffed content type. SVG is left out on purpose, since it can carry scripts.
var imageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// upload stores the image in the multipart "file" field under a random name
// and returns its path, which the pages resolve against the API's URL.
func (s *server) upload(w http.ResponseWriter, r *http.Request) {
	// Leave some room for the multipart headers around the file.
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+64<<10)
	file, header, err := r.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "images can be at most 5 MB")
			return
		}
		writeError(w, http.StatusBadRequest, `expected an image in the "file" field`)
		return
	}
	defer file.Close()
	if header.Size > maxUpload {
		writeError(w, http.StatusRequestEntityTooLarge, "images can be at most 5 MB")
		return
	}

	// Trust the bytes, not the file name or the client's Content-Type.
	head := make([]byte, 512)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		writeError(w, http.StatusBadRequest, "could not read the image")
		return
	}
	ext, ok := imageExtensions[http.DetectContentType(head[:n])]
	if !ok {
		writeError(w, http.StatusBadRequest, "only JPEG, PNG, GIF and WebP images are allowed")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		serverError(w, err)
		return
	}

	name := strings.ToLower(rand.Text()) + ext
	path := filepath.Join(s.UploadDir, name)
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		serverError(w, err)
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		os.Remove(path)
		serverError(w, err)
		return
	}
	if err := out.Close(); err != nil {
		os.Remove(path)
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"url": "/uploads/" + name})
}

// serveUpload serves one uploaded file. Uploads get random names that are never reused, and seed images are
// renamed rather than replaced when they change, so browsers may cache them for good.
func (s *server) serveUpload(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.UploadDir, name)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, path)
}
