package api

import (
	"context"
	"errors"
	"net/http"

	"portfolio-api/internal/store"
)

func (s *server) getPortfolio(w http.ResponseWriter, r *http.Request) {
	p, err := s.Store.Portfolio(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type profileBody struct {
	Profile   store.Profile   `json:"profile"`
	Education store.Education `json:"education"`
}

func (s *server) putProfile(w http.ResponseWriter, r *http.Request) {
	var body profileBody
	if !readValid(w, r, &body, validateProfile) {
		return
	}
	if err := s.Store.SaveProfile(r.Context(), body.Profile, body.Education); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// ---- Create, update and delete for the lists (projects, skills, experience) ----

func createHandler[T any](validate func(*T) error, insert func(context.Context, *T) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var item T
		if !readValid(w, r, &item, validate) {
			return
		}
		if err := insert(r.Context(), &item); err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	}
}

func updateHandler[T any](validate func(*T) error, update func(context.Context, int64, *T) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		var item T
		if !readValid(w, r, &item, validate) {
			return
		}
		if err := update(r.Context(), id, &item); err != nil {
			storeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	}
}

func deleteHandler(remove func(context.Context, int64) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		if err := remove(r.Context(), id); err != nil {
			storeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ---- Stars: any visitor can star a project, once per browser and a few times per IP ----

type starBody struct {
	Visitor string `json:"visitor"` // a random ID the visitor's browser keeps
	Starred bool   `json:"starred"`
}

func (s *server) starProject(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body starBody
	if !readValid(w, r, &body, validateStar) {
		return
	}
	stars, err := s.Store.SetStar(r.Context(), id, body.Visitor, clientIP(r), body.Starred)
	if errors.Is(err, store.ErrStarLimit) {
		writeError(w, http.StatusTooManyRequests, "this project has already been starred from your network")
		return
	}
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stars": stars, "starred": body.Starred})
}

// ---- Comments: any visitor can comment on a project; the owner can delete comments ----

type commentBody struct {
	Name string `json:"name"` // optional; "Anonymous" when blank
	Body string `json:"body"`
}

func (s *server) listComments(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	comments, err := s.Store.Comments(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, comments)
}

func (s *server) postComment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body commentBody
	if !readValid(w, r, &body, validateComment) {
		return
	}
	if !s.commentLimit.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many comments, try again in a few minutes")
		return
	}
	comment := store.Comment{Name: body.Name, Body: body.Body}
	if err := s.Store.InsertComment(r.Context(), id, &comment); err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}
