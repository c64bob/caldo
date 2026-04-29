package model

import "time"

// MergeResult describes whether a field-level merge between local and remote VTODO variants succeeded.
type MergeResult struct {
	Merged      bool
	Conflict    bool
	MergedVTODO string
}

// MergeVTODOFields performs a three-way merge using base, local, and remote VTODO payloads.
func MergeVTODOFields(baseVTODO string, localVTODO string, remoteVTODO string) MergeResult {
	if baseVTODO == "" {
		return MergeResult{Conflict: true}
	}

	base := ParseVTODOFields(baseVTODO)
	local := ParseVTODOFields(localVTODO)
	remote := ParseVTODOFields(remoteVTODO)

	patch := VTODOPatch{}
	if !mergeString(base.Title, local.Title, remote.Title, &patch.Summary) ||
		!mergeString(base.Description, local.Description, remote.Description, &patch.Description) ||
		!mergeString(base.Status, local.Status, remote.Status, &patch.Status) ||
		!mergeString(base.RRule, local.RRule, remote.RRule, &patch.RRule) ||
		!mergeDate(base.DueDate, local.DueDate, remote.DueDate, &patch.DueDate) ||
		!mergeTime(base.DueAt, local.DueAt, remote.DueAt, &patch.DueAt) ||
		!mergeTime(base.CompletedAt, local.CompletedAt, remote.CompletedAt, &patch.CompletedAt) ||
		!mergeInt(base.Priority, local.Priority, remote.Priority, &patch.Priority) ||
		!mergeCategories(base.Categories, local.Categories, remote.Categories, &patch.Categories) {
		return MergeResult{Conflict: true}
	}

	return MergeResult{Merged: true, MergedVTODO: PatchVTODO(baseVTODO, patch)}
}

func mergeString(base, local, remote string, target **string) bool {
	merged, changed, ok := mergeComparable(base, local, remote)
	if !ok {
		return false
	}
	if changed {
		v := merged
		*target = &v
	}
	return true
}
func mergeDate(base, local, remote *string, target **string) bool {
	merged, changed, ok := mergeComparable(base, local, remote)
	if !ok {
		return false
	}
	if changed {
		*target = merged
	}
	return true
}
func mergeTime(base, local, remote *time.Time, target **time.Time) bool {
	merged, changed, ok := mergeComparable(base, local, remote)
	if !ok {
		return false
	}
	if changed {
		*target = merged
	}
	return true
}
func mergeInt(base, local, remote *int, target **int) bool {
	merged, changed, ok := mergeComparable(base, local, remote)
	if !ok {
		return false
	}
	if changed {
		*target = merged
	}
	return true
}

func mergeCategories(base, local, remote []string, target *[]string) bool {
	bs := joinSorted(base)
	ls := joinSorted(local)
	rs := joinSorted(remote)
	merged, changed, ok := mergeComparable(bs, ls, rs)
	if !ok {
		return false
	}
	if !changed {
		return true
	}
	if merged == "" {
		*target = []string{}
		return true
	}
	*target = splitJoined(merged)
	return true
}
