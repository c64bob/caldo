package handlers

import (
	"net/http"
	"strings"
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
