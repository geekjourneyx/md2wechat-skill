package atomicfile

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestProbeRejectsUnusableDestinations(t *testing.T) {
	tests := []struct {
		name string
		path func(string) string
	}{
		{
			name: "missing parent",
			path: func(dir string) string {
				return filepath.Join(dir, "missing", "article.html")
			},
		},
		{
			name: "directory destination",
			path: func(dir string) string { return dir },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Probe(tt.path(t.TempDir())); err == nil {
				t.Fatal("Probe() error = nil")
			}
		})
	}
}

func TestProbePreservesExistingDestinationAndCleansTemp(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "article.html")
	sentinel := []byte("existing")
	if err := os.WriteFile(destination, sentinel, 0600); err != nil {
		t.Fatal(err)
	}

	if err := Probe(destination); err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, sentinel) {
		t.Fatalf("destination changed: %q", data)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
		t.Fatalf("temporary file leaked: %#v", entries)
	}
}

func TestWriteRejectsShortWriteAndCleansTemp(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "article.html")
	oldWriteTemp := writeTempFn
	writeTempFn = func(file *os.File, data []byte) (int, error) {
		return file.Write(data[:len(data)/2])
	}
	t.Cleanup(func() { writeTempFn = oldWriteTemp })

	_, err := Write(destination, []byte("complete article"))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("Write() error = %v, want io.ErrShortWrite", err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("destination exists after short write: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary file leaked after short write: %#v", entries)
	}
}

func TestWriteReplaceFailurePreservesDestinationAndCleansTemp(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "article.html")
	sentinel := []byte("existing article")
	if err := os.WriteFile(destination, sentinel, 0600); err != nil {
		t.Fatal(err)
	}
	oldReplace := replaceFileFn
	replaceFileFn = func(_, _ string) error { return errors.New("replace failed") }
	t.Cleanup(func() { replaceFileFn = oldReplace })

	_, err := Write(destination, []byte("new article"))
	if err == nil || err.Error() != "replace failed" {
		t.Fatalf("Write() error = %v, want replace failure", err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, sentinel) {
		t.Fatalf("destination changed after replace failure: %q", data)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
		t.Fatalf("temporary file leaked after replace failure: %#v", entries)
	}
}

func TestWriteReplacesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "article.html")
	if err := os.WriteFile(destination, []byte("old article"), 0600); err != nil {
		t.Fatal(err)
	}

	path, err := Write(destination, []byte("new article"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if path != destination {
		t.Fatalf("Write() path = %q, want %q", path, destination)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new article" {
		t.Fatalf("destination data = %q", data)
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0644 {
		t.Fatalf("destination mode = %o, want 644", info.Mode().Perm())
	}
}

func TestWriteTempCreatesTemporaryFile(t *testing.T) {
	data := []byte("preview article")
	path, err := WriteTemp("md2wechat-atomicfile-test-*.html", data)
	if err != nil {
		t.Fatalf("WriteTemp() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, data) {
		t.Fatalf("temporary file data = %q", written)
	}
}
