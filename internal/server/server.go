package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"tree-time-backend/internal/auth"
	"tree-time-backend/internal/config"
	"tree-time-backend/internal/repository"
)

type Server struct {
	cfg  config.Config
	repo *repository.Repository
}

func New(cfg config.Config, pool *pgxpool.Pool) *Server {
	return &Server{cfg: cfg, repo: repository.New(pool)}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)

	r.Post("/api/user/registration", s.registration)
	r.Post("/api/user/login", s.login)
	r.Post("/api/logout", s.logout)
	r.Get("/api/block-sessions", s.authenticated(s.listBlockSessions))
	r.Post("/api/block-sessions/start", s.authenticated(s.startBlockSession))
	r.Post("/api/block-sessions/finish", s.authenticated(s.finishBlockSession))
	return r
}

type registrationRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Login     string `json:"login"`
	Password  string `json:"password"`
}

func (s *Server) registration(w http.ResponseWriter, r *http.Request) {
	var req registrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "hash error", http.StatusInternalServerError)
		return
	}
	id, err := s.repo.CreateUser(r.Context(), repository.User{
		Email:        req.Email,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Login:        req.Login,
		PasswordHash: string(hash),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	user, err := s.repo.GetUserByLogin(r.Context(), req.Login)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	token, err := auth.GenerateToken(s.cfg.JWTSecret, user.ID, user.Email, user.FirstName, user.LastName, 24*time.Hour)
	if err != nil {
		http.Error(w, "token error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"login":      user.Login,
		},
	})
}

func (s *Server) logout(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type groupedSessionsResponse struct {
	ByYear  map[string][]repository.BlockSession `json:"by_year"`
	ByMonth map[string][]repository.BlockSession `json:"by_month"`
	ByDay   map[string][]repository.BlockSession `json:"by_day"`
}

func (s *Server) listBlockSessions(w http.ResponseWriter, r *http.Request, userID int64) {
	sessions, err := s.repo.ListBlockSessions(r.Context(), userID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	resp := groupedSessionsResponse{
		ByYear:  map[string][]repository.BlockSession{},
		ByMonth: map[string][]repository.BlockSession{},
		ByDay:   map[string][]repository.BlockSession{},
	}
	for _, sess := range sessions {
		y := sess.StartAt.Format("2006")
		m := sess.StartAt.Format("2006-01")
		d := sess.StartAt.Format("2006-01-02")
		resp.ByYear[y] = append(resp.ByYear[y], sess)
		resp.ByMonth[m] = append(resp.ByMonth[m], sess)
		resp.ByDay[d] = append(resp.ByDay[d], sess)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) startBlockSession(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := s.repo.StartBlockSession(r.Context(), userID); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}

type finishReq struct {
	BlockRange int64 `json:"block_range"`
}

func (s *Server) finishBlockSession(w http.ResponseWriter, r *http.Request, userID int64) {
	var req finishReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.BlockRange < 0 {
		req.BlockRange = 0
	}
	if err := s.repo.FinishLastBlockSession(r.Context(), userID, req.BlockRange); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) authenticated(next func(http.ResponseWriter, *http.Request, int64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" || token == r.Header.Get("Authorization") {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		claims, err := auth.ParseToken(s.cfg.JWTSecret, token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		next(w, r, claims.UserID)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
