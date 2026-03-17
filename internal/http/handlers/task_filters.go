package handlers

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"caldo/internal/http/middleware"
	"caldo/internal/http/render"
	"caldo/internal/service"
)

func parseTaskFilter(r *http.Request) render.TaskFilterState {
	return render.TaskFilterState{
		Priority: r.URL.Query()["priority"],
		Status:   r.URL.Query()["status"],
		DueFrom:  strings.TrimSpace(r.URL.Query().Get("due_from")),
		DueTo:    strings.TrimSpace(r.URL.Query().Get("due_to")),
		Folder:   strings.TrimSpace(r.URL.Query().Get("folder")),
		Context:  strings.TrimSpace(r.URL.Query().Get("context")),
		Goal:     strings.TrimSpace(r.URL.Query().Get("goal")),
		Tags:     strings.TrimSpace(r.URL.Query().Get("tags")),
		Star:     strings.TrimSpace(r.URL.Query().Get("star")),
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
	}
}

func applyTaskFilter(rows []render.TaskRow, f render.TaskFilterState) []render.TaskRow {
	out := make([]render.TaskRow, 0, len(rows))
	for _, row := range rows {
		if len(f.Priority) > 0 {
			ok := false
			for _, p := range f.Priority {
				if strings.TrimSpace(p) == strconv.Itoa(row.Priority) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		if len(f.Status) > 0 {
			ok := false
			for _, s := range f.Status {
				if strings.EqualFold(strings.TrimSpace(s), row.Status) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		if f.Folder != "" && !strings.EqualFold(f.Folder, row.Folder) {
			continue
		}
		if f.Context != "" && !strings.EqualFold(f.Context, row.Context) {
			continue
		}
		if f.Goal != "" && !strings.EqualFold(f.Goal, row.Goal) {
			continue
		}
		if strings.EqualFold(f.Star, "yes") && !row.IsStarred {
			continue
		}
		if strings.EqualFold(f.Star, "no") && row.IsStarred {
			continue
		}
		if strings.TrimSpace(f.Tags) != "" {
			tags := strings.Split(strings.ToLower(row.Categories), ",")
			want := strings.Split(strings.ToLower(f.Tags), ",")
			all := true
			for _, w := range want {
				needle := strings.TrimSpace(w)
				if needle == "" {
					continue
				}
				found := false
				for _, t := range tags {
					if strings.EqualFold(strings.TrimSpace(t), needle) {
						found = true
						break
					}
				}
				if !found {
					all = false
					break
				}
			}
			if !all {
				continue
			}
		}
		if f.DueFrom != "" || f.DueTo != "" {
			d, err := time.Parse("2006-01-02", strings.TrimSpace(row.DueInput))
			if err != nil {
				continue
			}
			if f.DueFrom != "" {
				from, err := time.Parse("2006-01-02", f.DueFrom)
				if err == nil && d.Before(from) {
					continue
				}
			}
			if f.DueTo != "" {
				to, err := time.Parse("2006-01-02", f.DueTo)
				if err == nil && d.After(to) {
					continue
				}
			}
		}
		out = append(out, row)
	}
	return out
}

func (h *TasksHandler) SaveFilter(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentifizierungsheader fehlt", http.StatusUnauthorized)
		return
	}
	if h.SavedFiltersService == nil {
		http.Redirect(w, r, "/tasks", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültiges Formular", http.StatusBadRequest)
		return
	}
	err := h.SavedFiltersService.Save(r.Context(), service.SavedFilterInput{
		PrincipalID: principal,
		Name:        strings.TrimSpace(r.FormValue("name")),
		ListID:      strings.TrimSpace(r.FormValue("list")),
		Priority:    r.Form["priority"],
		Status:      r.Form["status"],
		DueFrom:     strings.TrimSpace(r.FormValue("due_from")),
		DueTo:       strings.TrimSpace(r.FormValue("due_to")),
		Folder:      strings.TrimSpace(r.FormValue("folder")),
		Context:     strings.TrimSpace(r.FormValue("context")),
		Goal:        strings.TrimSpace(r.FormValue("goal")),
		Tags:        strings.TrimSpace(r.FormValue("tags")),
		Star:        strings.TrimSpace(r.FormValue("star")),
		Query:       strings.TrimSpace(r.FormValue("q")),
	})
	if err != nil {
		http.Error(w, "Filter konnte nicht gespeichert werden", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/tasks?view=search&list="+r.FormValue("list"), http.StatusSeeOther)
}

func (h *TasksHandler) PageSavedFilter(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentifizierungsheader fehlt", http.StatusUnauthorized)
		return
	}
	if h.SavedFiltersService == nil {
		http.NotFound(w, r)
		return
	}
	slug := strings.TrimSpace(r.PathValue("slug"))
	saved, found, err := h.SavedFiltersService.Get(r.Context(), principal, slug)
	if err != nil || !found {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	q.Set("view", "search")
	if saved.ListID != "" {
		q.Set("list", saved.ListID)
	}
	replaceList(q, "priority", saved.Priority)
	replaceList(q, "status", saved.Status)
	setOptional(q, "due_from", saved.DueFrom)
	setOptional(q, "due_to", saved.DueTo)
	setOptional(q, "folder", saved.Folder)
	setOptional(q, "context", saved.Context)
	setOptional(q, "goal", saved.Goal)
	setOptional(q, "tags", saved.Tags)
	setOptional(q, "star", saved.Star)
	setOptional(q, "q", saved.Query)
	r.URL.RawQuery = q.Encode()
	h.Page(w, r)
}

func setOptional(q url.Values, key, value string) {
	if strings.TrimSpace(value) == "" {
		q.Del(key)
		return
	}
	q.Set(key, strings.TrimSpace(value))
}

func replaceList(q url.Values, key string, values []string) {
	q.Del(key)
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			q.Add(key, trimmed)
		}
	}
}

func hotlistScore(row render.TaskRow, now time.Time) float64 {
	score := 0.0
	if row.IsStarred {
		score += 3
	}
	score += float64(priorityWeight(row.Priority))
	score += float64(dueDateWeight(row.DueInput, now))
	if strings.EqualFold(strings.TrimSpace(row.Status), "Next Action") {
		score += 1
	}
	return score
}

func priorityWeight(priority int) int {
	switch {
	case priority >= 8:
		return 4
	case priority >= 6:
		return 3
	case priority >= 4:
		return 2
	case priority >= 2:
		return 1
	default:
		return 0
	}
}

func dueDateWeight(due string, now time.Time) int {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(due))
	if err != nil {
		return 0
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if t.Before(today) {
		return 5
	}
	if t.Equal(today) {
		return 4
	}
	if t.Equal(today.AddDate(0, 0, 1)) {
		return 3
	}
	if !t.After(today.AddDate(0, 0, 7)) {
		return 2
	}
	return 0
}

func sortByHotlist(rows []render.TaskRow) {
	now := time.Now()
	sort.SliceStable(rows, func(i, j int) bool {
		li := hotlistScore(rows[i], now)
		lj := hotlistScore(rows[j], now)
		if li == lj {
			return rows[i].Summary < rows[j].Summary
		}
		return li > lj
	})
}
