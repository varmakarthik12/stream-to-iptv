package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"stream-to-iptv/internal/models"
	"stream-to-iptv/internal/repository"
	"stream-to-iptv/internal/stream"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("stream-to-iptv-secret-jwt-key")

func init() {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		jwtSecret = []byte(s)
	}
}

type AuthHandler struct {
	repo *repository.Repository
}

func NewAuthHandler(repo *repository.Repository) *AuthHandler {
	return &AuthHandler{repo: repo}
}

type contextKey string

const userContextKey contextKey = "currentUser"

// SetupStatus returns if first-time setup is required
func (h *AuthHandler) SetupStatus(w http.ResponseWriter, r *http.Request) {
	count, err := h.repo.CountUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	configExists := false
	if _, err := os.Stat("config.json"); err == nil {
		configExists = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"setup_required":       count == 0,
		"has_config_to_import": configExists,
	})
}

// Setup creates the initial admin user and optionally imports config.json
func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	count, err := h.repo.CountUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if count > 0 {
		http.Error(w, "Setup already completed", http.StatusBadRequest)
		return
	}

	var req models.SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Username) == "" || len(req.Password) < 4 {
		http.Error(w, "Username and password (min 4 chars) required", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	user, err := h.repo.CreateUser(strings.TrimSpace(req.Username), string(hash))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
		return
	}

	// Optionally import existing config.json
	importedCount := 0
	if req.ImportExisting {
		importedCount = h.importConfigJSON()
	}

	// Generate token and set session cookie
	token, err := createToken(user.ID, user.Username)
	if err == nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400 * 7,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"user":           user,
		"imported_count": importedCount,
		"token":          token,
	})
}

func (h *AuthHandler) importConfigJSON() int {
	configPath := "config.json"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0
	}

	type OldStream struct {
		Channel   string   `json:"channel"`
		Media     string   `json:"media"`
		Logo      string   `json:"logo"`
		Groups    []string `json:"groups"`
		ProgramId string   `json:"program_id"`
		TVGId     string   `json:"tvg_id"`
	}

	var oldStreams []OldStream
	if err := json.Unmarshal(data, &oldStreams); err != nil {
		return 0
	}

	count := 0
	for _, osItem := range oldStreams {
		slug := stream.Slugify(osItem.Channel)
		st := &models.Stream{
			Name:            osItem.Channel,
			Slug:            slug,
			MediaURL:        osItem.Media,
			LogoURL:         osItem.Logo,
			ProgramID:       osItem.ProgramId,
			TVGId:           osItem.TVGId,
			Mode:            "ondemand",
			IdleTimeoutSec:  180,
			BufferSize:      getEnvWithDefault("BUFFER_SIZE", "1000000"),
			FifoSize:        os.Getenv("FIFO_SIZE"),
			UseGPU:          strings.EqualFold(os.Getenv("USE_GPU"), "true"),
			OverrunNonfatal: strings.EqualFold(os.Getenv("OVERRUN_NONFATAL"), "true"),
			Enabled:         true,
		}

		created, err := h.repo.CreateStream(st)
		if err == nil && len(osItem.Groups) > 0 {
			var catIDs []string
			for _, g := range osItem.Groups {
				g = strings.TrimSpace(g)
				if g == "" {
					continue
				}
				catSlug := stream.Slugify(g)
				cat, err := h.repo.CreateCategory(g, catSlug, 0)
				if err == nil && cat != nil {
					catIDs = append(catIDs, cat.ID)
				}
			}
			if len(catIDs) > 0 {
				_ = h.repo.SetStreamCategories(created.ID, catIDs)
			}
		}
		count++
	}
	logrus.Infof("Imported %d streams from config.json", count)
	return count
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user, err := h.repo.GetUserByUsername(strings.TrimSpace(req.Username))
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := createToken(user.ID, user.Username)
	if err != nil {
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
		"user": map[string]string{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		// 1. Check Cookie
		if cookie, err := r.Cookie("auth_token"); err == nil {
			tokenStr = cookie.Value
		}

		// 2. Check Authorization Header
		if tokenStr == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenStr == "" {
			http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized: invalid or expired token", http.StatusUnauthorized)
			return
		}

		user, err := h.repo.GetUserByID(claims.Subject)
		if err != nil || user == nil {
			http.Error(w, "Unauthorized: user not found", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func createToken(userID, username string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		Issuer:    "stream-to-iptv",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func getEnvWithDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

