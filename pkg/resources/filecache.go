package resources

import (
	"fmt"
	"os"
)

type FileCache struct {
	path string
}

func NewFileCache(path string) *FileCache {
	return &FileCache{path: path}
}

func (f FileCache) Read() ([]byte, error) {
	data, err := readOrCreateCachefile(f.path)
	if err != nil {
		return nil, fmt.Errorf("could not read cache file: %w", err)
	}
	return data, nil
}

func (f FileCache) Write(data []byte) error {
	err := writeCacheFile(f.path, data)
	if err != nil {
		return fmt.Errorf("could not write cache file: %w", err)
	}
	return nil
}

func readOrCreateCachefile(file string) ([]byte, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		_, cErr := os.Create(file)
		if cErr != nil {
			return nil, cErr
		}
		return []byte{}, nil
	}

	return data, err
}

func writeCacheFile(file string, data []byte) error {
	err := os.WriteFile(file, data, 0644)
	if err != nil {
		return fmt.Errorf("could not write cache file: %w", err)
	}
	return nil
}
