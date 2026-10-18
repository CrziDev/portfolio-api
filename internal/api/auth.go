package api

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
	"time"
)

const sessionTTL = 30 * 24 * time.Hour

// The pages live on another site, where a cookie from this API would be a
// third-party cookie that browsers increasingly block. So sign-in hands out a
// token that the pages send back as "Authorization: Bearer <token>".

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if s.Password == "" {
		writeError(w, http.StatusServiceUnavailable, "editing is disabled: ADMIN_PASSWORD is not set")
		return
	}
	// Comparing hashes keeps the comparison constant-time regardless of the guess's length.
	got, want := sha256.Sum256([]byte(body.Password)), sha256.Sum256([]byte(s.Password))
	if subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
		time.Sleep(500 * time.Millisecond) // slows down guessing
		writeError(w, http.StatusUnauthorized, "wrong password")
		return
	}

	token := rand.Text()
	if err := s.Store.CreateSession(r.Context(), token, time.Now().Add(sessionTTL)); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (s *server) session(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"admin": s.isAdmin(r)})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.DeleteSession(r.Context(), bearerToken(r)); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// admin lets the request through only with a valid session token.
func (s *server) admin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isAdmin(r) {
			writeError(w, http.StatusUnauthorized, "sign in required")
			return
		}
		next(w, r)
	}
}

func (s *server) isAdmin(r *http.Request) bool {
	token := bearerToken(r)
	if token == "" {
		return false
	}
	ok, err := s.Store.ValidSession(r.Context(), token)
	if err != nil {
		log.Printf("check session: %v", err)
	}
	return ok
}

func bearerToken(r *http.Request) string {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(token)
}
