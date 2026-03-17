package handlers

import (
	"log"
	"net/http"
	"regexp"
	"strings"
)

var (
	urlUserInfoPattern = regexp.MustCompile(`https?://[^\s/@:]+:[^\s/@]*@`)
	bearerPattern      = regexp.MustCompile(`(?i)bearer\s+[a-z0-9._\-+/=]+`)
)

func taskLoadError(err error) (string, int) {
	if err == nil {
		return "", http.StatusOK
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "nicht erreichbar") {
		return "CalDAV-Server nicht erreichbar. Bitte Netzwerk und URL prüfen.", http.StatusBadGateway
	}
	if strings.Contains(message, "tls") || strings.Contains(message, "x509") {
		return "TLS-Fehler bei der CalDAV-Verbindung. Zertifikat/Truststore prüfen.", http.StatusBadGateway
	}
	if strings.Contains(message, "unauthorized") || strings.Contains(message, "forbidden") || strings.Contains(message, "anmeldung") {
		return "CalDAV-Authentifizierung fehlgeschlagen. Bitte Zugangsdaten prüfen.", http.StatusBadGateway
	}
	return "Tasks konnten nicht geladen werden", http.StatusBadGateway
}

func logTaskLoadError(scope, principalID, listID string, err error) {
	if err == nil {
		return
	}
	principal := strings.TrimSpace(principalID)
	if principal == "" {
		principal = "unknown"
	}
	list := strings.TrimSpace(listID)
	if list == "" {
		list = "default"
	}
	sanitizedErr := sanitizeLogError(err.Error())
	log.Printf("task load failed scope=%s app_principal=%s list=%s err=%s", scope, principal, list, sanitizedErr)
}

func sanitizeLogError(in string) string {
	out := urlUserInfoPattern.ReplaceAllString(in, "https://***:***@")
	out = bearerPattern.ReplaceAllString(out, "Bearer ***")
	return strings.TrimSpace(out)
}
