// User_Service/services/avatar_service.go
package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"User_Service/config"
	"User_Service/models"

	"github.com/google/uuid"
)

// AvatarService управляет аватарами пользователей
type AvatarService struct {
	config *config.Config
}

// NewAvatarService создает новый сервис аватаров
func NewAvatarService(cfg *config.Config) *AvatarService {
	return &AvatarService{
		config: cfg,
	}
}

// FillAvatarURL дополняет URL аватара
func (s *AvatarService) FillAvatarURL(user *models.User) {
	if user.Picture == "" {
		user.Picture = s.getDefaultAvatarURL()
	} else if !strings.HasPrefix(user.Picture, "http") {
		user.Picture = s.getAvatarURL(user.Picture)
	}
}

// FillAvatarURLs дополняет URLs для списка пользователей
func (s *AvatarService) FillAvatarURLs(users []models.User) {
	for i := range users {
		s.FillAvatarURL(&users[i])
	}
}

// getDefaultAvatarURL возвращает URL дефолтного аватара
func (s *AvatarService) getDefaultAvatarURL() string {
	baseURL := s.config.Storage.BackendURL
	if s.config.Storage.CDNEnabled && s.config.Storage.CDNURL != "" {
		baseURL = s.config.Storage.CDNURL
	}
	return baseURL + s.config.Static.AvatarsPath + s.config.Static.DefaultAvatar
}

// getAvatarURL возвращает полный URL аватара
func (s *AvatarService) getAvatarURL(filename string) string {
	baseURL := s.config.Storage.BackendURL
	if s.config.Storage.CDNEnabled && s.config.Storage.CDNURL != "" {
		baseURL = s.config.Storage.CDNURL
	}
	return baseURL + s.config.Static.AvatarsPath + filename
}

// SaveAvatar сохраняет загруженный аватар
func (s *AvatarService) SaveAvatar(file multipart.File, header *multipart.FileHeader) (string, error) {
	// Проверка размера файла
	if header.Size > s.config.Static.MaxUploadSize {
		return "", fmt.Errorf("file too large: max size is %d bytes", s.config.Static.MaxUploadSize)
	}

	// Проверка типа файла
	contentType := header.Header.Get("Content-Type")
	if !isAllowedImageType(contentType) {
		return "", fmt.Errorf("invalid file type: %s", contentType)
	}

	// Генерация уникального имени файла
	ext := filepath.Ext(header.Filename)
	filename := uuid.New().String() + ext

	// Путь для сохранения
	avatarDir := filepath.Join(s.config.Static.Directory, "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create avatar directory: %w", err)
	}

	filePath := filepath.Join(avatarDir, filename)

	// Создание файла
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Копирование данных
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return filename, nil
}

// DeleteAvatar удаляет файл аватара
func (s *AvatarService) DeleteAvatar(filename string) error {
	if filename == "" || filename == s.config.Static.DefaultAvatar {
		return nil // Не удаляем пустые или дефолтные аватары
	}

	avatarDir := filepath.Join(s.config.Static.Directory, "avatars")
	filePath := filepath.Join(avatarDir, filename)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete avatar: %w", err)
	}

	return nil
}

// isAllowedImageType проверяет допустимость типа изображения
func isAllowedImageType(contentType string) bool {
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	return allowedTypes[contentType]
}
