# Story 6.3 — CalDAV-Operationen mit Timeout und Retry-Policy

## Name
Story 6.3 — CalDAV-Operationen mit Timeout und Retry-Policy

## Ziel
CalDAV-Zugriffe sind robust und kontrolliert.

## Eingangszustand
Remote-Operationen haben keine einheitliche Fehlerpolitik.

## Ausgangszustand
CalDAV-Operationen folgen festen Timeouts, Retry-Regeln und Backoff.

## Akzeptanzkriterien
* PROPFIND, REPORT, GET, PUT, DELETE, MKCALENDAR und Full-Scan haben definierte Timeouts.
* Sichere idempotente Operationen werden bis maximal 3 Versuche wiederholt.
* PUT Create wird nicht blind wiederholt.
* PUT Update mit `If-Match` darf wiederholt werden.
* `412 Precondition Failed` führt nicht zu Retry, sondern Konfliktbehandlung.
* DELETE mit `404` gilt als Erfolg.
* Backoff nutzt Jitter.

---
