// Package sysfs provides an abstraction over file system operations to allow for easier testing.
package sysfs

import (
	"os"
	"path/filepath"
)

// FileSystem abstracts basic file operations used against sysfs/dev nodes.
// Tests can replace `FS` with a fake implementation.
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Glob(pattern string) ([]string, error)
	Stat(path string) (os.FileInfo, error)
}

type defaultFS struct{}

func (defaultFS) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
func (defaultFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}
func (defaultFS) Glob(pattern string) ([]string, error) { return filepath.Glob(pattern) }
func (defaultFS) Stat(path string) (os.FileInfo, error) { return os.Stat(path) }

// FS is the package-level FileSystem used by code accessing sysfs. Tests may replace it.
var FS FileSystem = defaultFS{}
