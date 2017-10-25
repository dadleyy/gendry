package gendry

import "io"
import "fmt"
import "strings"
import "math/rand"
import "encoding/hex"

const (
	projectKeyLength = 32
)

var store *inMemoryStore

// Project defines an interaface that stores reports.
type Project interface {
	StoreReport(string, string, string) error
	FindReport(string) (io.Reader, io.Reader, error)
}

// ProjectStore defines a backend for storing projects & their reports.
type ProjectStore interface {
	// FindProject searches the store for a project given a name/key combination.
	FindProject(string, string) (Project, error)
	// CreateProject attempts to allocate a project based on the given name, returning the project and it's key.
	CreateProject(string) (Project, string, error)
}

// ReportStore defines report lookup functions based on tag/project.
type ReportStore interface {
	FindReport(string, string) (io.Reader, io.Reader, error)
}

// NewReportStore returns an implementation of the report store based on the connection string provided.
func NewReportStore() ReportStore {
	if store == nil {
		projects := make(map[string]*inMemoryProject)
		store = &inMemoryStore{projects: projects}
	}
	return store
}

// NewProjectStore returns an implementation of the project store based on the connection string provided.
func NewProjectStore() ProjectStore {
	if store == nil {
		projects := make(map[string]*inMemoryProject)
		store = &inMemoryStore{projects: projects}
	}
	return store
}

type inMemoryProject struct {
	key     string
	reports map[string][]string
}

func (p *inMemoryProject) StoreReport(tag string, html string, text string) error {
	p.reports[tag] = []string{html, text}
	return nil
}

func (p *inMemoryProject) FindReport(tag string) (io.Reader, io.Reader, error) {
	readers, ok := p.reports[tag]

	if !ok || len(readers) != 2 {
		return nil, nil, fmt.Errorf("not-found")
	}

	return strings.NewReader(readers[0]), strings.NewReader(readers[1]), nil
}

type inMemoryStore struct {
	projects map[string]*inMemoryProject
}

func (s *inMemoryStore) FindReport(project string, tag string) (io.Reader, io.Reader, error) {
	p, ok := s.projects[project]

	if !ok {
		return nil, nil, fmt.Errorf("not-found")
	}

	r, ok := p.reports[tag]

	if !ok {
		return nil, nil, fmt.Errorf("not-found")
	}

	return strings.NewReader(r[0]), strings.NewReader(r[1]), nil
}

func (s *inMemoryStore) CreateProject(name string) (Project, string, error) {
	if _, dupe := s.projects[name]; dupe == true {
		return nil, "", fmt.Errorf("duplicate")
	}

	key := make([]byte, projectKeyLength)

	if _, e := rand.Read(key); e != nil {
		return nil, "", e
	}

	project := &inMemoryProject{
		key:     hex.EncodeToString(key),
		reports: make(map[string][]string),
	}

	s.projects[name] = project

	return project, project.key, nil
}

func (s *inMemoryStore) FindProject(query string, key string) (Project, error) {
	p, ok := s.projects[query]

	if !ok {
		return nil, fmt.Errorf("not-found")
	}

	if p.key != key {
		return nil, fmt.Errorf("invalid-key")
	}

	return p, nil
}
