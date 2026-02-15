// Authorization/handlers/handlers.go
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"Authorization/auth"
	"Authorization/config"
	"Authorization/db"
	"Authorization/models"

	"github.com/gorilla/mux"
)

// AuthHandler обрабатывает запросы аутентификации
type AuthHandler struct {
	config         *config.Config
	db             *db.Database
	jwtService     *auth.JWTService
	passwordHasher auth.PasswordHasher
	userService    *UserServiceClient
}

// NewAuthHandler создает новый обработчик аутентификации
func NewAuthHandler(cfg *config.Config, database *db.Database) *AuthHandler {
	return &AuthHandler{
		config:         cfg,
		db:             database,
		jwtService:     auth.NewJWTService(cfg.JWT.SecretKey, cfg.JWT.TokenDuration),
		passwordHasher: auth.NewBcryptHasher(),
		userService:    NewUserServiceClient(cfg.UserService),
	}
}

// RegisterRoutes регистрирует маршруты для аутентификации
func (h *AuthHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/auth/register", h.Register).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/login", h.Login).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/validate", h.ValidateToken).Methods("GET", "OPTIONS")
}

// Register обрабатывает регистрацию нового пользователя
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON format")
		return
	}

	// Валидация входных данных
	if err := validateRegisterRequest(req); err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Проверка существования пользователя
	exists, err := h.db.UserExists(ctx, req.Login)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to check user existence")
		return
	}

	if exists {
		respondWithError(w, http.StatusConflict, "user_exists", "User already exists")
		return
	}

	// Хеширование пароля
	hashedPassword, err := h.passwordHasher.Hash(req.Password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		respondWithError(w, http.StatusInternalServerError, "hash_error", "Failed to process password")
		return
	}

	// Создание пользователя в БД
	user := models.User{
		Login:    req.Login,
		Password: hashedPassword,
	}

	if err := h.db.CreateUser(ctx, user); err != nil {
		log.Printf("Error creating user in database: %v", err)
		respondWithError(w, http.StatusInternalServerError, "database_error", "Failed to create user")
		return
	}

	// Создание профиля пользователя в User Service
	if err := h.userService.CreateUserProfile(ctx, req.Login); err != nil {
		log.Printf("Error creating user profile: %v", err)

		// Откат создания пользователя в auth БД
		if rollbackErr := h.db.DeleteUser(ctx, req.Login); rollbackErr != nil {
			log.Printf("CRITICAL: Failed to rollback user creation: %v", rollbackErr)
		}

		respondWithError(w, http.StatusInternalServerError, "profile_creation_error", "Failed to create user profile")
		return
	}

	// Генерация JWT токена
	token, err := h.jwtService.GenerateToken(req.Login)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		respondWithError(w, http.StatusInternalServerError, "token_error", "Failed to generate token")
		return
	}

	// Успешный ответ
	respondWithJSON(w, http.StatusCreated, models.AuthResponse{
		Token:     token,
		ExpiresIn: int64(h.config.JWT.TokenDuration.Seconds()),
		TokenType: "Bearer",
	})
}

// Login обрабатывает вход пользователя
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON format")
		return
	}

	// Валидация входных данных
	if err := validateLoginRequest(req); err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Получение пользователя из БД
	user, err := h.db.GetUserByLogin(ctx, req.Login)
	if err != nil {
		// Не раскрываем, существует ли пользователь
		respondWithError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid login or password")
		return
	}

	// Проверка пароля
	if err := h.passwordHasher.Compare(user.Password, req.Password); err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid login or password")
		return
	}

	// Генерация JWT токена
	token, err := h.jwtService.GenerateToken(user.Login)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		respondWithError(w, http.StatusInternalServerError, "token_error", "Failed to generate token")
		return
	}

	// Успешный ответ
	respondWithJSON(w, http.StatusOK, models.AuthResponse{
		Token:     token,
		ExpiresIn: int64(h.config.JWT.TokenDuration.Seconds()),
		TokenType: "Bearer",
	})
}

// ValidateToken проверяет валидность JWT токена
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.Header.Get("Authorization")
	if tokenStr == "" {
		respondWithError(w, http.StatusUnauthorized, "missing_token", "Authorization header required")
		return
	}

	// Удаляем префикс "Bearer " если есть
	if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}

	claims, err := h.jwtService.ValidateToken(tokenStr)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid_token", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, models.SuccessResponse{
		Status: "ok",
		Data: map[string]interface{}{
			"login":      claims.Login,
			"expires_at": claims.ExpiresAt.Time,
		},
	})
}

// --- Вспомогательные функции ---

func validateRegisterRequest(req models.RegisterRequest) error {
	if len(req.Login) < 3 || len(req.Login) > 50 {
		return &ValidationError{Field: "login", Message: "must be between 3 and 50 characters"}
	}
	if len(req.Password) < 2 || len(req.Password) > 72 {
		return &ValidationError{Field: "password", Message: "must be between 2 and 72 characters"}
	}
	return nil
}

func validateLoginRequest(req models.LoginRequest) error {
	if req.Login == "" {
		return &ValidationError{Field: "login", Message: "is required"}
	}
	if req.Password == "" {
		return &ValidationError{Field: "password", Message: "is required"}
	}
	return nil
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + " " + e.Message
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, statusCode int, code, message string) {
	respondWithJSON(w, statusCode, models.ErrorResponse{
		Error:   code,
		Message: message,
	})
}
