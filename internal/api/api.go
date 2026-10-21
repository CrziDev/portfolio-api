// Package api serves the portfolio's content as JSON, lets the signed-in
// owner edit it, and stores uploaded images.
package api

import (
	"net/http"
	"slices"

	"portfolio-api/internal/store"
)

type Config struct {
	Store     *store.Store
	Password  string   // for /api/login; editing is disabled when empty
	UploadDir string   // where uploaded images are kept
	Origins   []string // origins the pages are served from, allowed by CORS
}

type server struct {
	Config
}

// New returns the API's HTTP handler.
func New(c Config) http.Handler {
	s := &server{c}
	return withCORS(c.Origins, s.routes())
}

func (s *server) routes() http.Handler {
	st := s.Store
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/portfolio", s.getPortfolio)
	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("GET /api/session", s.session)
	mux.HandleFunc("POST /api/logout", s.admin(s.logout))

	mux.HandleFunc("PUT /api/profile", s.admin(s.putProfile))

	mux.HandleFunc("POST /api/projects", s.admin(createHandler(validateProject, st.InsertProject)))
	mux.HandleFunc("PUT /api/projects/{id}", s.admin(updateHandler(validateProject, st.UpdateProject)))
	mux.HandleFunc("DELETE /api/projects/{id}", s.admin(deleteHandler(st.DeleteProject)))
	mux.HandleFunc("POST /api/projects/{id}/star", s.starProject) // open to visitors

	mux.HandleFunc("POST /api/skills", s.admin(createHandler(validateSkill, st.InsertSkill)))
	mux.HandleFunc("PUT /api/skills/{id}", s.admin(updateHandler(validateSkill, st.UpdateSkill)))
	mux.HandleFunc("DELETE /api/skills/{id}", s.admin(deleteHandler(st.DeleteSkill)))

	mux.HandleFunc("POST /api/experience", s.admin(createHandler(validateExperience, st.InsertExperience)))
	mux.HandleFunc("PUT /api/experience/{id}", s.admin(updateHandler(validateExperience, st.UpdateExperience)))
	mux.HandleFunc("DELETE /api/experience/{id}", s.admin(deleteHandler(st.DeleteExperience)))

	mux.HandleFunc("POST /api/uploads", s.admin(s.upload))
	mux.HandleFunc("GET /uploads/{name}", s.serveUpload)

	return mux
}

// withCORS lets the pages, served from another origin, call the API.
// Requests from origins not in the list get no CORS headers, so browsers block them.
func withCORS(origins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Add("Vary", "Origin")
		if origin := r.Header.Get("Origin"); origin != "" && slices.Contains(origins, origin) {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			h.Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
