package utils

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// OpenFileAcrossFilesystem tries to open the file in each provided filesystem
func OpenFileAcrossFilesystem(fileName string, fss ...fs.FS) ([]byte, error) {
	var file fs.File
	var err error

	for _, fsys := range fss {
		file, err = fsys.Open(fileName)
		if err == nil {
			// File is found, exit the loop
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("failed to open file: %v", err)
		}
	}
	if file == nil || err != nil {
		return nil, errors.New("file not found in any filesystem")
	}

	// Defer file.Close and read its content
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	return content, nil
}

// CopyFSDir copies the contents of an embedded FS in your binary
// to some directory on disk.
// f is the //go:embed attribute
// origin is the name of the directory in the embed call
// target is the target directory to write into
func CopyFSDir(f embed.FS, origin, target string) error {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		if err := os.MkdirAll(target, 0666); err != nil {
			err = fmt.Errorf("error creating directory: %v", err)
			return err
		}
	}

	files, err := f.ReadDir(origin)
	if err != nil {
		return fmt.Errorf("error reading directory: %v", err)
	}

	for _, file := range files {
		sourceFileName := filepath.Join(origin, file.Name())
		destFileName := filepath.Join(target, file.Name())

		if file.IsDir() {
			if err := CopyFSDir(f, sourceFileName, destFileName); err != nil {
				return fmt.Errorf("error copying subdirectory: %v", err)
			}
			continue
		}

		fileContent, err := f.ReadFile(sourceFileName)
		if err != nil {
			return fmt.Errorf("error reading file: %v", err)
		}

		if err := os.WriteFile(destFileName, fileContent, 0644); err != nil {
			return fmt.Errorf("error writing file: %w", err)
		}
	}

	return nil
}
