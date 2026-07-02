package handler

import "testing"

func TestEffectiveCalendarBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		baseURL         string
		calendarHomeSet string
		want            string
	}{
		{
			name:            "empty home set keeps configured url",
			baseURL:         " https://nextcloud.example/remote.php/dav ",
			calendarHomeSet: "",
			want:            "https://nextcloud.example/remote.php/dav",
		},
		{
			name:            "empty home set infers parent from nextcloud calendar collection",
			baseURL:         "https://nextcloud.example/remote.php/dav/calendars/alice/tasks/",
			calendarHomeSet: "",
			want:            "https://nextcloud.example/remote.php/dav/calendars/alice/",
		},
		{
			name:            "empty home set infers parent from collection without trailing slash",
			baseURL:         "https://nextcloud.example/remote.php/dav/calendars/alice/tasks",
			calendarHomeSet: "",
			want:            "https://nextcloud.example/remote.php/dav/calendars/alice/",
		},
		{
			name:            "empty home set keeps calendar home url",
			baseURL:         "https://nextcloud.example/remote.php/dav/calendars/alice/",
			calendarHomeSet: "",
			want:            "https://nextcloud.example/remote.php/dav/calendars/alice/",
		},
		{
			name:            "absolute path resolves against configured origin",
			baseURL:         "https://nextcloud.example/remote.php/dav",
			calendarHomeSet: "/remote.php/dav/calendars/alice/",
			want:            "https://nextcloud.example/remote.php/dav/calendars/alice/",
		},
		{
			name:            "relative path is scoped under configured path",
			baseURL:         "https://nextcloud.example/remote.php/dav",
			calendarHomeSet: "calendars/alice/",
			want:            "https://nextcloud.example/remote.php/dav/calendars/alice/",
		},
		{
			name:            "absolute url wins",
			baseURL:         "https://nextcloud.example/remote.php/dav",
			calendarHomeSet: "https://dav.example/calendars/alice/",
			want:            "https://dav.example/calendars/alice/",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := effectiveCalendarBaseURL(tt.baseURL, tt.calendarHomeSet)
			if got != tt.want {
				t.Fatalf("effectiveCalendarBaseURL()=%q want %q", got, tt.want)
			}
		})
	}
}
