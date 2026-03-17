package domain

import "time"

type Task struct {
	UID             string
	ParentUID       string
	Goal            string
	CollectionID    string
	CollectionHref  string
	Href            string
	ETag            string
	Summary         string
	Description     string
	Status          string
	Priority        int
	PercentComplete int
	Categories      []string
	Due             *time.Time
	DueKind         string
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
}
