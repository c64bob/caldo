package view

import (
	"net/url"
	"strconv"
)

// SavedFilterCreateFormView contains state for the saved-filter create form.
type SavedFilterCreateFormView struct {
	Error      string
	Name       string
	Query      string
	IsFavorite bool
}

// SavedFilterItemView contains one saved filter management row.
type SavedFilterItemView struct {
	ID                string
	Name              string
	Query             string
	IsFavorite        bool
	ServerVersion     int
	EditError         string
	EditName          string
	EditQuery         string
	EditFavorite      bool
	EditFavoriteValid bool
	DeleteError       string
}

func savedFilterPath(item SavedFilterItemView) string {
	return "/filters/" + url.PathEscape(item.ID)
}

func savedFilterExpectedVersion(item SavedFilterItemView) string {
	return strconv.Itoa(item.ServerVersion)
}

func savedFilterEditName(item SavedFilterItemView) string {
	if item.EditName != "" {
		return item.EditName
	}
	return item.Name
}

func savedFilterEditQuery(item SavedFilterItemView) string {
	if item.EditQuery != "" {
		return item.EditQuery
	}
	return item.Query
}

func savedFilterEditFavorite(item SavedFilterItemView) bool {
	if item.EditFavoriteValid {
		return item.EditFavorite
	}
	return item.IsFavorite
}

func savedFilterFavoriteLabel(item SavedFilterItemView) string {
	if item.IsFavorite {
		return "Favorit"
	}
	return "Nicht favorisiert"
}
