// Chat_Service/storage/storage.go
package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

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
	fileID := uuid.NewString()

	// Читаем первые 512 байт для определения реального типа
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return nil, fmt.Errorf("failed to read file header: %w", err)
	}

	// Определяем реальный MIME по содержимому
	detectedMime := http.DetectContentType(buffer[:n])

	// Сбрасываем позицию обратно в начало
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	// Выбираем расширение по реальному MIME, не по заголовку
	ext := mimeToExt(detectedMime)

	// Если не распознали — берём из оригинального имени файла
	if ext == "" {
		ext = filepath.Ext(header.Filename)
	}

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

	return &SavedFile{
		ID:       fileID,
		FileName: header.Filename,
		MimeType: detectedMime,
		Size:     size,
		Path:     dstPath,
		URL:      s.baseURL + "/media/" + storeName,
	}, nil
}

// mimeToExt возвращает расширение по MIME-типу
func mimeToExt(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	default:
		return ""
	}
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
