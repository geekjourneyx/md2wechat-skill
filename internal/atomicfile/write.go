package atomicfile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var writeTempFn = func(file *os.File, data []byte) (int, error) {
	return file.Write(data)
}

var replaceFileFn = replaceFile

// Probe verifies the destination using Write's sibling-temp directory without
// creating or replacing the destination.
func Probe(destination string) error {
	if destination == "" {
		return nil
	}

	info, err := os.Stat(destination)
	switch {
	case err == nil && !info.Mode().IsRegular():
		return fmt.Errorf("destination is not a regular file: %s", destination)
	case err != nil && !os.IsNotExist(err):
		return err
	}

	temp, err := os.CreateTemp(
		filepath.Dir(destination),
		"."+filepath.Base(destination)+".probe-*",
	)
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return os.Remove(tempPath)
}

// Write atomically replaces destination with data using a sibling temporary file.
func Write(destination string, data []byte) (string, error) {
	temp, err := os.CreateTemp(filepath.Dir(destination), "."+filepath.Base(destination)+".tmp-*")
	if err != nil {
		return "", err
	}
	tempPath := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = temp.Close()
			_ = os.Remove(tempPath)
		}
	}()

	if err := writeAndClose(temp, data, true); err != nil {
		return "", err
	}
	if err := replaceFileFn(tempPath, destination); err != nil {
		return "", err
	}
	removeTemp = false
	return destination, nil
}

// WriteTemp writes data to a durable temporary file and returns its path.
func WriteTemp(pattern string, data []byte) (string, error) {
	temp, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	tempPath := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = temp.Close()
			_ = os.Remove(tempPath)
		}
	}()

	if err := writeAndClose(temp, data, false); err != nil {
		return "", err
	}
	removeTemp = false
	return tempPath, nil
}

func writeAndClose(file *os.File, data []byte, chmod bool) error {
	written, err := writeTempFn(file, data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	if chmod {
		if err := file.Chmod(0644); err != nil {
			return err
		}
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return file.Close()
}
