package api

import (
	"context"
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
