package sync

import (
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/model"
)

// MapCalendarObject normalizes one remote CalDAV object into Caldo's imported task shape.
func MapCalendarObject(object caldav.CalendarObject, projectName string) (db.ImportedTask, bool) {
	parsed := model.ParseVTODOFields(object.RawVTODO)
	if parsed.UID == "" {
		return db.ImportedTask{}, false
	}

	labels, favorite := model.CategoriesToLabelsAndFavorite(parsed.Categories)
	if favorite {
		labels = append(labels, model.ReservedFavoriteCategory)
	}

	title := parsed.Title
	if title == "" {
		title = parsed.UID
	}

	return db.ImportedTask{
		UID:         parsed.UID,
		Href:        object.Href,
		ETag:        object.ETag,
		Title:       title,
		Description: parsed.Description,
		Status:      parsed.Status,
		CompletedAt: formatTimePointer(parsed.CompletedAt),
		DueDate:     parsed.DueDate,
		DueAt:       formatTimePointer(parsed.DueAt),
		Priority:    parsed.Priority,
		RRule:       parsed.RRule,
		ParentUID:   parsed.ParentUID,
		RawVTODO:    object.RawVTODO,
		BaseVTODO:   object.RawVTODO,
		LabelNames:  labels,
		ProjectName: projectName,
	}, true
}

// MapCalendarObjects normalizes a remote calendar scan result.
func MapCalendarObjects(objects []caldav.CalendarObject, projectName string) []db.ImportedTask {
	tasks := make([]db.ImportedTask, 0, len(objects))
	for _, object := range objects {
		task, ok := MapCalendarObject(object, projectName)
		if !ok {
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func formatTimePointer(v *time.Time) *string {
	if v == nil {
		return nil
	}
	formatted := v.UTC().Format(time.RFC3339)
	return &formatted
}
