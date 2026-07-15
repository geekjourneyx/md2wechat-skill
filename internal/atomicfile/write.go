package atomicfile

import (
	"io"
	"os"
	"path/filepath"
)

var writeTempFn = func(file *os.File, data []byte) (int, error) {
	return file.Write(data)
}

var replaceFileFn = replaceFile

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
