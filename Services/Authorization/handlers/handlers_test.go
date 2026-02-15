// Authorization/handlers/handlers_test.go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"Authorization/auth"
	"Authorization/config"
	"Authorization/models"
)

// MockDatabase для тестирования
type MockDatabase struct {
	users map[string]models.User
}

func (m *MockDatabase) CreateUser(ctx context.Context, user models.User) error {
	m.users[user.Login] = user
	return nil
}

func (m *MockDatabase) GetUserByLogin(ctx context.Context, login string) (models.User, error) {
	user, ok := m.users[login]
	if !ok {
		return models.User{}, fmt.Errorf("user not found")
	}
	return user, nil
}

func (m *MockDatabase) UserExists(ctx context.Context, login string) (bool, error) {
	_, ok := m.users[login]
	return ok, nil
}

func (m *MockDatabase) DeleteUser(ctx context.Context, login string) error {
	delete(m.users, login)
	return nil
}

// MockUserServiceClient для тестирования
type MockUserServiceClient struct {
	shouldFail bool
}

func (m *MockUserServiceClient) CreateUserProfile(ctx context.Context, login string) error {
	if m.shouldFail {
		return fmt.Errorf("mock error")
	}
	return nil
}

func TestRegisterSuccess(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			SecretKey:     []byte("test-secret"),
			TokenDuration: 24 * time.Hour,
		},
	}

	mockDB := &MockDatabase{users: make(map[string]models.User)}

	handler := &AuthHandler{
		config:         cfg,
		db:             mockDB,
		jwtService:     auth.NewJWTService(cfg.JWT.SecretKey, cfg.JWT.TokenDuration),
		passwordHasher: auth.NewBcryptHasher(),
		userService:    &MockUserServiceClient{shouldFail: false},
	}

	reqBody := models.RegisterRequest{
		Login:    "testuser",
		Password: "testpass123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp models.AuthResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Token == "" {
		t.Error("Expected token in response")
	}
}

func TestRegisterDuplicateUser(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			SecretKey:     []byte("test-secret"),
			TokenDuration: 24 * time.Hour,
		},
	}

	mockDB := &MockDatabase{users: make(map[string]models.User)}
	mockDB.users["existinguser"] = models.User{Login: "existinguser"}

	handler := &AuthHandler{
		config:         cfg,
		db:             mockDB,
		jwtService:     auth.NewJWTService(cfg.JWT.SecretKey, cfg.JWT.TokenDuration),
		passwordHasher: auth.NewBcryptHasher(),
		userService:    &MockUserServiceClient{},
	}

	reqBody := models.RegisterRequest{
		Login:    "existinguser",
		Password: "testpass123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.Register(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

func TestLoginSuccess(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			SecretKey:     []byte("test-secret"),
			TokenDuration: 24 * time.Hour,
		},
	}

	hasher := auth.NewBcryptHasher()
	hashedPass, _ := hasher.Hash("testpass123")

	mockDB := &MockDatabase{users: make(map[string]models.User)}
	mockDB.users["testuser"] = models.User{
		Login:    "testuser",
		Password: hashedPass,
	}

	handler := &AuthHandler{
		config:         cfg,
		db:             mockDB,
		jwtService:     auth.NewJWTService(cfg.JWT.SecretKey, cfg.JWT.TokenDuration),
		passwordHasher: hasher,
	}

	reqBody := models.LoginRequest{
		Login:    "testuser",
		Password: "testpass123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.AuthResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Token == "" {
		t.Error("Expected token in response")
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			SecretKey:     []byte("test-secret"),
			TokenDuration: 24 * time.Hour,
		},
	}

	hasher := auth.NewBcryptHasher()
	hashedPass, _ := hasher.Hash("correctpass")

	mockDB := &MockDatabase{users: make(map[string]models.User)}
	mockDB.users["testuser"] = models.User{
		Login:    "testuser",
		Password: hashedPass,
	}

	handler := &AuthHandler{
		config:         cfg,
		db:             mockDB,
		jwtService:     auth.NewJWTService(cfg.JWT.SecretKey, cfg.JWT.TokenDuration),
		passwordHasher: hasher,
	}

	reqBody := models.LoginRequest{
		Login:    "testuser",
		Password: "wrongpass",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}
