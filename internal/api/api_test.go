package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"portfolio-api/internal/store"
)

const testPassword = "secret"
const testOrigin = "https://portfolio.example"

type testAPI struct {
	t       *testing.T
	handler http.Handler
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(context.Background(), filepath.Join(dir, "portfolio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	uploadDir := filepath.Join(dir, "uploads")
	if err := os.Mkdir(uploadDir, 0o755); err != nil {
		t.Fatal(err)
	}
	handler := New(Config{Store: st, Password: testPassword, UploadDir: uploadDir, Origins: []string{testOrigin}})
	return &testAPI{t: t, handler: handler}
}

// do sends a request; a non-nil body is sent as JSON unless it is already a *bytes.Buffer.
func (api *testAPI) do(method, path, token string, body any) *httptest.ResponseRecorder {
	api.t.Helper()
	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case *bytes.Buffer:
		reader = b
	default:
		data, err := json.Marshal(b)
		if err != nil {
			api.t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, path, reader)
	if reader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	api.handler.ServeHTTP(rec, req)
	return rec
}

func (api *testAPI) login() string {
	api.t.Helper()
	rec := api.do("POST", "/api/login", "", map[string]string{"password": testPassword})
	expectStatus(api.t, rec, http.StatusOK)
	var body struct{ Token string }
	decode(api.t, rec, &body)
	return body.Token
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, want, rec.Body)
	}
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decode %s: %v", rec.Body, err)
	}
}

func TestWritesNeedToken(t *testing.T) {
	api := newTestAPI(t)
	for _, req := range []struct{ method, path string }{
		{"PUT", "/api/profile"},
		{"POST", "/api/projects"},
		{"PUT", "/api/projects/1"},
		{"DELETE", "/api/projects/1"},
		{"POST", "/api/skills"},
		{"DELETE", "/api/experience/1"},
		{"POST", "/api/uploads"},
		{"DELETE", "/api/comments/1"},
		{"POST", "/api/logout"},
	} {
		for _, token := range []string{"", "not-a-real-token"} {
			if rec := api.do(req.method, req.path, token, map[string]string{}); rec.Code != http.StatusUnauthorized {
				t.Errorf("%s %s with token %q: status = %d, want 401", req.method, req.path, token, rec.Code)
			}
		}
	}
}

func TestLoginAndLogout(t *testing.T) {
	api := newTestAPI(t)

	expectStatus(t, api.do("POST", "/api/login", "", map[string]string{"password": "wrong"}), http.StatusUnauthorized)

	token := api.login()
	var session struct{ Admin bool }
	decode(t, api.do("GET", "/api/session", token, nil), &session)
	if !session.Admin {
		t.Fatal("session with a fresh token is not admin")
	}

	expectStatus(t, api.do("POST", "/api/logout", token, nil), http.StatusNoContent)
	decode(t, api.do("GET", "/api/session", token, nil), &session)
	if session.Admin {
		t.Fatal("token still works after logout")
	}
}

func TestProjectCRUD(t *testing.T) {
	api := newTestAPI(t)
	token := api.login()

	rec := api.do("POST", "/api/projects", token, store.Project{
		Name: "  new-project ", Type: "Personal", Language: "Go", Stack: []string{"Go", " ", "SQLite"},
		URL: "https://github.com/CrziDev/new-project", Image: "/uploads/abc.png",
	})
	expectStatus(t, rec, http.StatusCreated)
	var created store.Project
	decode(t, rec, &created)
	if created.ID == 0 || created.Name != "new-project" || !reflect.DeepEqual(created.Stack, []string{"Go", "SQLite"}) {
		t.Fatalf("created = %+v", created)
	}

	created.Description = "Updated description."
	path := "/api/projects/" + strconv.FormatInt(created.ID, 10)
	expectStatus(t, api.do("PUT", path, token, created), http.StatusOK)

	var p store.Portfolio
	decode(t, api.do("GET", "/api/portfolio", "", nil), &p)
	last := p.Projects[len(p.Projects)-1]
	if last.ID != created.ID || last.Description != "Updated description." {
		t.Fatalf("new project should be last and updated, got %+v", last)
	}

	expectStatus(t, api.do("DELETE", path, token, nil), http.StatusNoContent)
	expectStatus(t, api.do("DELETE", path, token, nil), http.StatusNotFound)
	expectStatus(t, api.do("PUT", path, token, created), http.StatusNotFound)
}

func TestStars(t *testing.T) {
	api := newTestAPI(t)
	alice, bob := "0d6c9a3e-6b4f-4c55-9a1e-2f7b8c1d0e11", "7f3e2b1a-9c8d-4e6f-a5b4-c3d2e1f0a9b8"

	var p store.Portfolio
	decode(t, api.do("GET", "/api/portfolio", "", nil), &p)
	path := "/api/projects/" + strconv.FormatInt(p.Projects[0].ID, 10) + "/star"

	star := func(visitor string, starred bool, want int) {
		t.Helper()
		rec := api.do("POST", path, "", map[string]any{"visitor": visitor, "starred": starred})
		expectStatus(t, rec, http.StatusOK)
		var body struct{ Stars int }
		decode(t, rec, &body)
		if body.Stars != want {
			t.Fatalf("stars = %d, want %d", body.Stars, want)
		}
	}
	star(alice, true, 1)
	star(alice, true, 1) // starring twice counts once
	star(bob, true, 2)
	star(alice, false, 1)

	decode(t, api.do("GET", "/api/portfolio", "", nil), &p)
	if p.Projects[0].Stars != 1 {
		t.Fatalf("portfolio stars = %d, want 1", p.Projects[0].Stars)
	}

	expectStatus(t, api.do("POST", path, "", map[string]any{"visitor": "x", "starred": true}), http.StatusBadRequest)
	expectStatus(t, api.do("POST", "/api/projects/9999/star", "", map[string]any{"visitor": alice, "starred": true}), http.StatusNotFound)

	// A starred project can still be deleted; its stars go with it.
	expectStatus(t, api.do("DELETE", strings.TrimSuffix(path, "/star"), api.login(), nil), http.StatusNoContent)
}

// A private window gets a new visitor ID every time, so stars are also capped per IP.
func TestStarLimitPerIP(t *testing.T) {
	api := newTestAPI(t)
	star := func(visitor, ip string, starred bool) int {
		body := fmt.Sprintf(`{"visitor":%q,"starred":%t}`, visitor, starred)
		req := httptest.NewRequest("POST", "/api/projects/1/star", strings.NewReader(body))
		req.Header.Set("X-Forwarded-For", ip)
		rec := httptest.NewRecorder()
		api.handler.ServeHTTP(rec, req)
		return rec.Code
	}
	visitor := func(i int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", i) }

	for i := range store.StarsPerIP {
		if code := star(visitor(i), "203.0.113.1", true); code != http.StatusOK {
			t.Fatalf("star %d: status = %d, want 200", i+1, code)
		}
	}
	if code := star(visitor(99), "203.0.113.1", true); code != http.StatusTooManyRequests {
		t.Errorf("one star too many: status = %d, want 429", code)
	}
	if code := star(visitor(0), "203.0.113.1", true); code != http.StatusOK {
		t.Errorf("starring again from a browser that already starred: status = %d, want 200", code)
	}
	if code := star(visitor(99), "203.0.113.2", true); code != http.StatusOK {
		t.Errorf("another network: status = %d, want 200", code)
	}

	// Unstarring frees up a place for the network.
	if code := star(visitor(0), "203.0.113.1", false); code != http.StatusOK {
		t.Fatalf("unstar: status = %d, want 200", code)
	}
	if code := star(visitor(98), "203.0.113.1", true); code != http.StatusOK {
		t.Errorf("after an unstar: status = %d, want 200", code)
	}
}

func TestComments(t *testing.T) {
	api := newTestAPI(t)

	var p store.Portfolio
	decode(t, api.do("GET", "/api/portfolio", "", nil), &p)
	path := "/api/projects/" + strconv.FormatInt(p.Projects[0].ID, 10) + "/comments"

	var comments []store.Comment
	decode(t, api.do("GET", path, "", nil), &comments)
	if comments == nil || len(comments) != 0 {
		t.Fatalf("comments on a new project = %v, want []", comments)
	}

	rec := api.do("POST", path, "", map[string]string{"name": "  Kim ", "body": " Nice work! "})
	expectStatus(t, rec, http.StatusCreated)
	var first store.Comment
	decode(t, rec, &first)
	if first.ID == 0 || first.Name != "Kim" || first.Body != "Nice work!" || first.CreatedAt.IsZero() {
		t.Fatalf("created = %+v", first)
	}
	expectStatus(t, api.do("POST", path, "", map[string]string{"body": "Second"}), http.StatusCreated)

	decode(t, api.do("GET", path, "", nil), &comments)
	if len(comments) != 2 || comments[0].ID != first.ID || comments[1].Name != "Anonymous" {
		t.Fatalf("comments = %+v, want Kim's then Anonymous's", comments)
	}

	for name, body := range map[string]map[string]string{
		"empty":     {"body": "  "},
		"long body": {"body": strings.Repeat("x", 1001)},
		"long name": {"name": strings.Repeat("x", 41), "body": "hi"},
	} {
		if rec := api.do("POST", path, "", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, rec.Code)
		}
	}
	expectStatus(t, api.do("GET", "/api/projects/9999/comments", "", nil), http.StatusNotFound)
	expectStatus(t, api.do("POST", "/api/projects/9999/comments", "", map[string]string{"body": "hi"}), http.StatusNotFound)

	token := api.login()
	commentPath := "/api/comments/" + strconv.FormatInt(first.ID, 10)
	expectStatus(t, api.do("DELETE", commentPath, token, nil), http.StatusNoContent)
	expectStatus(t, api.do("DELETE", commentPath, token, nil), http.StatusNotFound)
	decode(t, api.do("GET", path, "", nil), &comments)
	if len(comments) != 1 {
		t.Fatalf("after delete, %d comments, want 1", len(comments))
	}
}

func TestCommentRateLimit(t *testing.T) {
	api := newTestAPI(t)
	post := func(ip string) int {
		req := httptest.NewRequest("POST", "/api/projects/1/comments", strings.NewReader(`{"body":"hi"}`))
		req.Header.Set("X-Forwarded-For", ip)
		rec := httptest.NewRecorder()
		api.handler.ServeHTTP(rec, req)
		return rec.Code
	}
	for i := range 5 {
		if code := post("203.0.113.1"); code != http.StatusCreated {
			t.Fatalf("comment %d: status = %d, want 201", i+1, code)
		}
	}
	if code := post("203.0.113.1"); code != http.StatusTooManyRequests {
		t.Errorf("sixth comment: status = %d, want 429", code)
	}
	if code := post("203.0.113.2"); code != http.StatusCreated {
		t.Errorf("another visitor: status = %d, want 201", code)
	}
}

func TestValidation(t *testing.T) {
	api := newTestAPI(t)
	token := api.login()

	for name, body := range map[string]store.Project{
		"no name":        {Name: " "},
		"javascript url": {Name: "x", URL: "javascript:alert(1)"},
		"data image":     {Name: "x", Image: "data:image/png;base64,AAAA"},
		"relative image": {Name: "x", Image: "/uploads/../portfolio.db"},
		"gallery image":  {Name: "x", Gallery: []string{"/uploads/a.png", "javascript:alert(1)"}},
		"large gallery":  {Name: "x", Gallery: strings.Split(strings.Repeat("/uploads/a.png,", 21), ",")[:21]},
	} {
		if rec := api.do("POST", "/api/projects", token, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, rec.Code)
		}
	}

	if rec := api.do("POST", "/api/experience", token, store.Experience{Org: "Acme"}); rec.Code != http.StatusBadRequest {
		t.Errorf("experience without roles: status = %d, want 400", rec.Code)
	}
	if rec := api.do("POST", "/api/projects", token, map[string]string{"name": "x", "extra": "y"}); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown field: status = %d, want 400", rec.Code)
	}
}

func TestProfileUpdate(t *testing.T) {
	api := newTestAPI(t)
	token := api.login()

	var p store.Portfolio
	decode(t, api.do("GET", "/api/portfolio", "", nil), &p)
	p.Profile.Tagline = "A new tagline."
	p.Education.Year = "2025"
	expectStatus(t, api.do("PUT", "/api/profile", token, profileBody{p.Profile, p.Education}), http.StatusOK)

	var after store.Portfolio
	decode(t, api.do("GET", "/api/portfolio", "", nil), &after)
	if after.Profile.Tagline != "A new tagline." || after.Education.Year != "2025" {
		t.Fatalf("profile not saved: %+v %+v", after.Profile, after.Education)
	}
}

func multipartFile(t *testing.T, name string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)
	form.Close()
	return &buf, form.FormDataContentType()
}

func TestUpload(t *testing.T) {
	api := newTestAPI(t)
	token := api.login()

	upload := func(name string, content []byte) *httptest.ResponseRecorder {
		body, contentType := multipartFile(t, name, content)
		req := httptest.NewRequest("POST", "/api/uploads", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		api.handler.ServeHTTP(rec, req)
		return rec
	}

	// Named .png, but it's text: the bytes decide.
	expectStatus(t, upload("evil.png", []byte("<script>alert(1)</script>")), http.StatusBadRequest)

	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	rec := upload("photo.jpg", png)
	expectStatus(t, rec, http.StatusCreated)
	var body struct{ URL string }
	decode(t, rec, &body)
	if !strings.HasPrefix(body.URL, "/uploads/") || !strings.HasSuffix(body.URL, ".png") {
		t.Fatalf("url = %q, want /uploads/<name>.png", body.URL)
	}

	served := api.do("GET", body.URL, "", nil)
	expectStatus(t, served, http.StatusOK)
	if got := served.Header().Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", got)
	}
	if !bytes.Equal(served.Body.Bytes(), png) {
		t.Error("served file differs from the upload")
	}
	expectStatus(t, api.do("GET", "/uploads/..%2fportfolio.db", "", nil), http.StatusNotFound)
	expectStatus(t, api.do("GET", "/uploads/missing.png", "", nil), http.StatusNotFound)
}

func TestCORS(t *testing.T) {
	api := newTestAPI(t)

	preflight := func(origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("OPTIONS", "/api/projects", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
		rec := httptest.NewRecorder()
		api.handler.ServeHTTP(rec, req)
		return rec
	}

	rec := preflight(testOrigin)
	expectStatus(t, rec, http.StatusNoContent)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
		t.Errorf("allowed origin: Access-Control-Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
		t.Errorf("Access-Control-Allow-Headers = %q, want Authorization", got)
	}

	if got := preflight("https://evil.example").Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("other origin: Access-Control-Allow-Origin = %q, want none", got)
	}
}
