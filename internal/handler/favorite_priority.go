package handler

import (
	"strings"

	"caldo/internal/model"
)

func applyFavoritePriorityRule(base model.VTODOFields, patch *model.VTODOPatch, priorityTouched bool) {
	categories := base.Categories
	if patch.Categories != nil {
		categories = patch.Categories
	}
	priority := patchPriority(base, *patch)

	if priorityTouched {
		normalizedCategories, err := model.CategoriesWithFavoriteFromPriority(categories, priority)
		if err == nil && (patch.Categories != nil || !sameCategories(categories, normalizedCategories)) {
			patch.Categories = normalizedCategories
		}
		return
	}

	normalizedCategories, normalizedPriority, err := model.NormalizeFavoritePriorityFields(categories, priority)
	if err != nil {
		return
	}
	if patch.Categories != nil || !sameCategories(categories, normalizedCategories) {
		patch.Categories = normalizedCategories
	}
	if !samePriority(priority, normalizedPriority) {
		patch.Priority = normalizedPriority
		patch.ClearPriority = normalizedPriority == nil
	}
}

func patchPriority(base model.VTODOFields, patch model.VTODOPatch) *int {
	if patch.ClearPriority {
		return nil
	}
	if patch.Priority != nil {
		return patch.Priority
	}
	return base.Priority
}

func samePriority(left *int, right *int) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func sameCategories(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[string]int, len(left))
	for _, category := range left {
		counts[strings.TrimSpace(category)]++
	}
	for _, category := range right {
		key := strings.TrimSpace(category)
		if counts[key] == 0 {
			return false
		}
		counts[key]--
	}
	return true
}
