package store

// Shapes match the PORTFOLIO object the pages render, so GET /api/portfolio
// can be handed to the front end as is.

type Portfolio struct {
	Profile    Profile      `json:"profile"`
	Skills     []Skill      `json:"skills"`
	Projects   []Project    `json:"projects"`
	Experience []Experience `json:"experience"`
	Education  Education    `json:"education"`
}

type Profile struct {
	Name     string `json:"name"`
	Greeting string `json:"greeting"`
	Handle   string `json:"handle"`
	Initials string `json:"initials"`
	Photo    string `json:"photo"`
	Tagline  string `json:"tagline"`
	Status   string `json:"status"`
	Location string `json:"location"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	GitHub   string `json:"github"`
	LinkedIn string `json:"linkedin"`
}

type Education struct {
	Degree string `json:"degree"`
	School string `json:"school"`
	Year   string `json:"year"`
}

type Skill struct {
	ID    int64    `json:"id"`
	Label string   `json:"label"`
	Items []string `json:"items"`
}

type Project struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Language    string   `json:"language"`
	Description string   `json:"description"`
	Stack       []string `json:"stack"`
	Image       string   `json:"image"`
	URL         string   `json:"url"`
	Stars       int      `json:"stars"` // read-only: set by visitors through SetStar, ignored on save
}

type Experience struct {
	ID         int64    `json:"id"`
	Org        string   `json:"org"`
	Meta       string   `json:"meta"`
	Roles      []Role   `json:"roles"`
	Highlights []string `json:"highlights"`
}

type Role struct {
	Title  string `json:"title"`
	Period string `json:"period"`
}
