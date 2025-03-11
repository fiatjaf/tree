package ostree

import (
	"os"
)

// FS uses the system filesystem
type FS struct{}

// Stat a path
func (f *FS) Stat(path string) (os.FileInfo, error) {
	return os.Lstat(path)
}

// ReadDir reads a directory
func (f *FS) ReadDir(path string) ([]string, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	names, err := dir.Readdirnames(-1)
	dir.Close()
	if err != nil {
		return nil, err
	}
	return names, nil
}
