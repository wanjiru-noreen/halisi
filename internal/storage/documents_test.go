package storage

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"
)

func TestDocumentValidation(t *testing.T) {
	pdf := []byte("%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\n%%EOF\n")
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, mime string
		data       []byte
		valid      bool
	}{
		{"licence.pdf", "application/pdf", pdf, true}, {"photo.png", "image/png", imageData.Bytes(), true},
		{"evil.html", "text/html", []byte("<script>alert(1)</script>"), false},
		{"fake.pdf", "application/pdf", []byte("not a document"), false},
		{"wrong.exe", "application/pdf", pdf, false}, {"wrong.png", "image/png", pdf, false},
		{"truncated.pdf", "application/pdf", []byte("%PDF-1.4\n"), false},
		{"large.pdf", "application/pdf", bytes.Repeat([]byte("x"), MaxDocumentSize+1), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Validate(tt.name, tt.mime, tt.data)
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v, error=%v", tt.valid, err)
			}
		})
	}
}

func TestPrivateStoragePersists(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenDocuments(dir)
	if err != nil {
		t.Fatal(err)
	}
	data := "%PDF-1.4\n%%EOF\n"
	file, err := store.Put("../../licence.pdf", "application/pdf", strings.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if file.Name != "licence.pdf" || len(file.Key) != 64 || len(file.SHA256) != 64 {
		t.Fatal("unsafe metadata")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenDocuments(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	reader, err := store.Open(file.Key)
	if err != nil {
		t.Fatal("document lost after reopening persistent storage")
	}
	defer reader.Close()
	actual, err := io.ReadAll(reader)
	if err != nil || string(actual) != data {
		t.Fatal("persistent data changed")
	}
	if _, err := store.Open("../secret"); err == nil {
		t.Fatal("path traversal accepted")
	}
}
