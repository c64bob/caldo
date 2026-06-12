package view

// SearchSaveFilterView contains the optional search-to-saved-filter form state.
type SearchSaveFilterView struct {
	Enabled    bool
	Query      string
	IsFavorite bool
}
