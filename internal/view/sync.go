package view

import (
	"context"
	"strings"
)

func syncState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "running":
		return "running"
	case "error":
		return "error"
	default:
		return "idle"
	}
}

func syncStateLabel(ctx context.Context, state string) string {
	text := Text(ctx)
	switch syncState(state) {
	case "running":
		return text.SettingsSyncRunning
	case "error":
		return text.SettingsSyncError
	default:
		return text.SettingsSyncIdle
	}
}

func syncActionClass(state string) string {
	return "caldo-icon-button caldo-sync-action caldo-sync-action-" + syncState(state)
}

func syncTooltipTemplate(ctx context.Context, state string) string {
	text := Text(ctx)
	return text.SettingsManualSync + ". " + text.SettingsSyncStatus + ": " + syncStateLabel(ctx, state) + ". " + text.SettingsSyncLastOK + ": {last}"
}

func syncTooltipLabel(ctx context.Context, state string, lastSuccess LocalDateTimeView) string {
	return strings.Replace(syncTooltipTemplate(ctx, state), "{last}", lastSuccess.Text, 1)
}
