package storage

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const MaxDocumentSize = 5 * 1024 * 1024

var ErrDocument = errors.New("upload a valid PDF, JPEG or PNG document, up to 5 MiB")
var keyPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type File struct {
	Key, Name, ContentType, SHA256 string
	Size                           int64
}
type Documents struct{ root *os.Root }

func OpenDocuments(directory string) (*Documents, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absolute, 0700); err != nil {
		return nil, err
	}
	if err := os.Chmod(absolute, 0700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, err
	}
	return &Documents{root: root}, nil
}
func (d *Documents) Close() error { return d.root.Close() }
func (d *Documents) Put(name, claimedType string, reader io.Reader) (File, error) {
	data, err := io.ReadAll(io.LimitReader(reader, MaxDocumentSize+1))
	if err != nil {
		return File{}, err
	}
	mime, err := Validate(name, claimedType, data)
	if err != nil {
		return File{}, err
	}
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	if len(name) > 120 || name == "." || name == "" {
		return File{}, ErrDocument
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return File{}, err
	}
	id := hex.EncodeToString(key)
	file, err := d.root.OpenFile(id, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return File{}, err
	}
	success := false
	defer func() {
		_ = file.Close()
		if !success {
			_ = d.root.Remove(id)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return File{}, err
	}
	if err := file.Sync(); err != nil {
		return File{}, err
	}
	if err := file.Close(); err != nil {
		return File{}, err
	}
	sum := sha256.Sum256(data)
	success = true
	return File{Key: id, Name: name, ContentType: mime, Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:])}, nil
}
func Validate(name, claimedType string, data []byte) (string, error) {
	if len(data) == 0 || len(data) > MaxDocumentSize {
		return "", ErrDocument
	}
	ext := strings.ToLower(path.Ext(strings.ReplaceAll(name, `\`, "/")))
	mime := http.DetectContentType(data)
	switch mime {
	case "application/pdf":
		if ext != ".pdf" || !bytes.HasPrefix(data, []byte("%PDF-")) || !bytes.Contains(data[max(0, len(data)-1024):], []byte("%%EOF")) {
			return "", ErrDocument
		}
	case "image/jpeg", "image/png":
		if (mime == "image/jpeg" && ext != ".jpg" && ext != ".jpeg") || (mime == "image/png" && ext != ".png") {
			return "", ErrDocument
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 10000000 {
			return "", ErrDocument
		}
		if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
			return "", ErrDocument
		}
	default:
		return "", ErrDocument
	}
	if claimedType != "" && claimedType != mime {
		return "", ErrDocument
	}
	return mime, nil
}
func (d *Documents) Open(key string) (*os.File, error) {
	if !keyPattern.MatchString(key) {
		return nil, os.ErrNotExist
	}
	return d.root.Open(key)
}
func (d *Documents) Remove(key string) error {
	if !keyPattern.MatchString(key) {
		return os.ErrNotExist
	}
	return d.root.Remove(key)
}
