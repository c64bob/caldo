# Story 4.1 — CalDAV-Credentials verschlüsselt speichern

## Name
Story 4.1 — CalDAV-Credentials verschlüsselt speichern

## Ziel
Zugangsdaten werden niemals im Klartext persistiert.

## Eingangszustand
Es gibt keine Secret-Speicherung.

## Ausgangszustand
CalDAV-Passwörter werden mit AES-256-GCM gespeichert.

## Akzeptanzkriterien
* Der Schlüssel stammt aus `ENCRYPTION_KEY`.
* Speicherformat enthält Version, Nonce und Ciphertext.
* Das Passwort wird nicht im Klartext gespeichert.
* Credentials werden nicht geloggt.
* Formal ungültiger Key verhindert den Start.
* Formal gültiger, aber falscher Key verhindert nicht den Start, macht CalDAV aber unavailable.

---
