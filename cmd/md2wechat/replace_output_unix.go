//go:build !windows

package main

import "os"

func replaceOutputFile(source, destination string) error {
	return os.Rename(source, destination)
}
