# Story 5.1 — Setup-Gate für Erststart

## Name
Story 5.1 — Setup-Gate für Erststart

## Ziel
Die normale App ist vor abgeschlossenem Setup nicht erreichbar.

## Eingangszustand
Unkonfigurierte Installationen könnten normale Routen verwenden.

## Ausgangszustand
`setup_complete=false` blockiert Normalbetrieb hart.

## Akzeptanzkriterien
* Bei `setup_complete=false` sind nur Setup-Routen und `/health` erreichbar.
* Andere Routen leiten nach `/setup`.
* Setup-Routen laufen durch Proxy-Auth.
* Mutierende Setup-Routen laufen durch CSRF.
* Der Wizard-Zustand liegt serverseitig in Settings.
* Normalbetrieb und Setup teilen DB und Router, sind aber durch Gate getrennt.

---
