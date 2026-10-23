package api

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"portfolio-api/internal/store"
)

// Validators trim text, drop empty list entries, and check required fields and links.

func validateProfile(b *profileBody) error {
	p, e := &b.Profile, &b.Education
	trim(&p.Name, &p.Greeting, &p.Handle, &p.Initials, &p.Photo, &p.Tagline, &p.Status, &p.Location,
		&p.Phone, &p.Email, &p.GitHub, &p.LinkedIn, &e.Degree, &e.School, &e.Year)
	if p.Name == "" {
		return errors.New("name is required")
	}
	if !isImage(p.Photo) {
		return errors.New("photo must be an uploaded image or an http(s) URL")
	}
	return nil
}

func validateSkill(s *store.Skill) error {
	trim(&s.Label)
	s.Items = cleanList(s.Items)
	if s.Label == "" {
		return errors.New("label is required")
	}
	return nil
}

const maxGallery = 20

func validateProject(p *store.Project) error {
	trim(&p.Name, &p.Type, &p.Language, &p.Description, &p.Details, &p.Image, &p.URL)
	p.Stack = cleanList(p.Stack)
	p.Gallery = cleanList(p.Gallery)
	if p.Name == "" {
		return errors.New("name is required")
	}
	if !isLink(p.URL) {
		return errors.New("url must be an http(s) URL")
	}
	if !isImage(p.Image) {
		return errors.New("image must be an uploaded image or an http(s) URL")
	}
	if len(p.Gallery) > maxGallery {
		return fmt.Errorf("the gallery can have at most %d images", maxGallery)
	}
	for _, src := range p.Gallery {
		if !isImage(src) {
			return errors.New("gallery images must be uploaded images or http(s) URLs")
		}
	}
	return nil
}

func validateExperience(e *store.Experience) error {
	trim(&e.Org, &e.Meta)
	e.Highlights = cleanList(e.Highlights)
	roles := []store.Role{}
	for _, role := range e.Roles {
		trim(&role.Title, &role.Period)
		if role.Title != "" {
			roles = append(roles, role)
		}
	}
	e.Roles = roles
	if e.Org == "" {
		return errors.New("org is required")
	}
	if len(e.Roles) == 0 {
		return errors.New("at least one role is required")
	}
	return nil
}

// validateStar checks the visitor ID looks like the UUID the pages generate.
func validateStar(b *starBody) error {
	if len(b.Visitor) < 16 || len(b.Visitor) > 64 || strings.Trim(b.Visitor, "0123456789abcdefABCDEF-") != "" {
		return errors.New("visitor must be the browser's random ID")
	}
	return nil
}

const (
	maxCommentName = 40
	maxCommentBody = 1000
)

// validateComment names anonymous comments "Anonymous" and checks lengths, counted in characters.
func validateComment(c *commentBody) error {
	trim(&c.Name, &c.Body)
	if c.Name == "" {
		c.Name = "Anonymous"
	}
	if utf8.RuneCountInString(c.Name) > maxCommentName {
		return fmt.Errorf("name can be at most %d characters", maxCommentName)
	}
	if c.Body == "" {
		return errors.New("comment is empty")
	}
	if utf8.RuneCountInString(c.Body) > maxCommentBody {
		return fmt.Errorf("comments can be at most %d characters", maxCommentBody)
	}
	return nil
}

func trim(fields ...*string) {
	for _, field := range fields {
		*field = strings.TrimSpace(*field)
	}
}

func cleanList(list []string) []string {
	clean := []string{}
	for _, item := range list {
		if item = strings.TrimSpace(item); item != "" {
			clean = append(clean, item)
		}
	}
	return clean
}

// isLink reports whether s is empty or an absolute http(s) URL, which rules out javascript: and data: links.
func isLink(s string) bool {
	if s == "" {
		return true
	}
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func isImage(s string) bool {
	return isLink(s) || (strings.HasPrefix(s, "/uploads/") && !strings.Contains(s, ".."))
}
