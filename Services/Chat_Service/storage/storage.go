// Chat_Service/storage/storage.go
package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type FileStorage struct {
	directory string
	baseURL   string
}

func NewFileStorage(directory, baseURL string) (*FileStorage, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create media directory: %w", err)
	}
	return &FileStorage{directory: directory, baseURL: baseURL}, nil
}

type SavedFile struct {
	ID       string
	FileName string // оригинальное имя
	MimeType string
	Size     int64
	Path     string // путь на диске
	URL      string // публичный URL
}

// Save сохраняет файл на диск, возвращает метаданные
func (s *FileStorage) Save(file multipart.File, header *multipart.FileHeader) (*SavedFile, error) {
	// Генерируем уникальное имя, сохраняем расширение
	ext := filepath.Ext(header.Filename)
	fileID := uuid.NewString()
	storeName := fileID + ext

	dstPath := filepath.Join(s.directory, storeName)
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(dstPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	// Берём только основной тип без параметров (например boundary)
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	return &SavedFile{
		ID:       fileID,
		FileName: header.Filename,
		MimeType: mimeType,
		Size:     size,
		Path:     dstPath,
		URL:      s.baseURL + "/media/" + storeName,
	}, nil
}

// Delete удаляет файл по storeName (fileID + ext)
func (s *FileStorage) Delete(storeName string) error {
	path := filepath.Join(s.directory, storeName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// FilePath возвращает полный путь к файлу по storeName
func (s *FileStorage) FilePath(storeName string) string {
	return filepath.Join(s.directory, storeName)
}
