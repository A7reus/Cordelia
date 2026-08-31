package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/A7reus/Cordelia/internal/identity"
	"github.com/A7reus/Cordelia/internal/registry"
)

const MaxMessageSize = 64 * 1024
const MaxUploadSize = 100 << 20

type Message struct {
	From string `json:"from"`
	Text string `json:"text"`
}

func InfoHandler(id identity.Identity) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(id)
	}
}

func PeersHandler(reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reg.Snapshot())
	}
}

func MessageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxMessageSize)

		var msg Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				http.Error(w, "message too long", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if msg.Text == "" {
			http.Error(w, "empty text", http.StatusBadRequest)
			return
		}

		log.Printf("message from %s: %s", msg.From, msg.Text)
		w.WriteHeader(http.StatusNoContent)
	}
}

func UploadHandler(downloadDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize+10<<20)

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid multipart", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file part", http.StatusBadRequest)
			return
		}
		defer file.Close()

		filename := filepath.Base(header.Filename)
		if filename == "" || filename == "." {
			http.Error(w, "invalid filename", http.StatusBadRequest)
			return
		}

		if err := os.MkdirAll(downloadDir, 0o755); err != nil {
			http.Error(w, "cannot create download dir", http.StatusInternalServerError)
			return
		}

		destPath := uniquePath(downloadDir, filename)
		dest, err := os.Create(destPath)
		if err != nil {
			http.Error(w, "cannot create file", http.StatusInternalServerError)
			return
		}
		defer dest.Close()

		h := sha256.New()
		if _, err := io.Copy(io.MultiWriter(dest, h), file); err != nil {
			http.Error(w, "failed to save", http.StatusInternalServerError)
			return
		}

		sum := hex.EncodeToString(h.Sum(nil))
		w.Header().Set("X-Checksum-Sha256", sum)
		log.Printf("received file %s (%d bytes) sha256 %s -> %s", filename, header.Size, sum, destPath)
		w.WriteHeader(http.StatusNoContent)
	}
}

func ResolveDownloadDir(outDir string) string {
	if outDir != "" {
		return outDir
	}
	home, err := os.UserHomeDir()
	if err == nil {
		dl := filepath.Join(home, "Downloads", "cordelia")
		if _, err := os.Stat(filepath.Join(home, "Downloads")); err == nil {
			return dl
		}
	}
	return filepath.Join(".", "downloads")
}

func uniquePath(dir, name string) string {
	candidate := filepath.Join(dir, name)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		tried := fmt.Sprintf("%s (%d)%s", base, i, ext)
		candidate = filepath.Join(dir, tried)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
