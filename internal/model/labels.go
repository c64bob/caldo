package model

import (
	"fmt"
	"sort"
	"strings"
)

const ReservedFavoriteCategory = "STARRED"

// NormalizeLabelName validates and normalizes a user-provided label name.
func NormalizeLabelName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("label name must not be empty")
	}
	if strings.EqualFold(trimmed, ReservedFavoriteCategory) {
		return "", fmt.Errorf("label name %q is reserved", ReservedFavoriteCategory)
	}

	return trimmed, nil
}

// CategoriesToLabelsAndFavorite converts VTODO CATEGORIES into local labels and favorite state.
func CategoriesToLabelsAndFavorite(categories []string) ([]string, bool) {
	labels := make([]string, 0, len(categories))
	seen := make(map[string]struct{}, len(categories))
	isFavorite := false

	for _, rawCategory := range categories {
		category := strings.TrimSpace(rawCategory)
		if category == "" {
			continue
		}
		if strings.EqualFold(category, ReservedFavoriteCategory) {
			isFavorite = true
			continue
		}

		key := strings.ToLower(category)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		labels = append(labels, category)
	}

	sort.Slice(labels, func(i, j int) bool {
		left := strings.ToLower(labels[i])
		right := strings.ToLower(labels[j])
		if left == right {
			return labels[i] < labels[j]
		}
		return left < right
	})

	return labels, isFavorite
}

// LabelsAndFavoriteToCategories converts local labels and favorite state into VTODO CATEGORIES.
func LabelsAndFavoriteToCategories(labels []string, isFavorite bool) ([]string, error) {
	categories := make([]string, 0, len(labels)+1)
	seen := make(map[string]struct{}, len(labels)+1)

	for _, label := range labels {
		normalizedLabel, err := NormalizeLabelName(label)
		if err != nil {
			return nil, err
		}

		key := strings.ToLower(normalizedLabel)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		categories = append(categories, normalizedLabel)
	}

	sort.Slice(categories, func(i, j int) bool {
		left := strings.ToLower(categories[i])
		right := strings.ToLower(categories[j])
		if left == right {
			return categories[i] < categories[j]
		}
		return left < right
	})

	if isFavorite {
		categories = append(categories, ReservedFavoriteCategory)
	}

	return categories, nil
}
