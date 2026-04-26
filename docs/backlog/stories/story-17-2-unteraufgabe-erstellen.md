# Story 17.2 — Unteraufgabe erstellen

## Name
Story 17.2 — Unteraufgabe erstellen

## Ziel
Der Nutzer kann direkte Unteraufgaben anlegen.

## Eingangszustand
Neue Tasks sind nur Wurzelaufgaben.

## Ausgangszustand
Eine Unteraufgabe ist lokal und in Nextcloud als solche sichtbar.

## Akzeptanzkriterien
* Unteraufgaben werden nur über „Unteraufgabe hinzufügen“ erstellt.
* Quick Add erstellt keine Unteraufgaben.
* Unteraufgaben erhalten Parent-Referenz im VTODO.
* Unteraufgaben können selbst keine Unteraufgaben haben.
* Entsprechende UI-Aktion ist deaktiviert.
* Erstellung schreibt sofort zu CalDAV.
* Nextcloud-Integrationstest bestätigt Sichtbarkeit.

---
