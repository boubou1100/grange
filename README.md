# Grange

Gestionnaire de dépôts d'artefacts **100 % libre**, avec une API compatible Nexus
et (à terme) une interface web générique pour tous les formats.

> Statut : squelette. Seul le format **raw** est implémenté, avec stockage fichiers
> et métadonnées en mémoire.

## Démarrage

```bash
make run                      # serveur sur :8081, jeton admin = "dev"

# créer un dépôt
curl -X POST localhost:8081/service/rest/v1/repositories \
  -H 'Authorization: Bearer dev' -d '{"name":"files","format":"raw"}'

# envoyer / télécharger un fichier
curl -H 'Authorization: Bearer dev' --upload-file notes.txt localhost:8081/repository/files/notes.txt
curl localhost:8081/repository/files/notes.txt
```

## Architecture

| Dossier | Rôle |
|---|---|
| `internal/model` | Repository / Component / Asset, communs à tous les formats |
| `internal/storage` | Blobs adressés par SHA-256 (FS fourni, S3 à venir) |
| `internal/meta` | Métadonnées (mémoire fournie, PostgreSQL à venir, cf. `migrations/`) |
| `internal/format` | Interface de plugin de format + registre |
| `internal/format/raw` | Premier format |
| `internal/server` | API REST compatible Nexus + routage vers les formats |

## Ajouter un format

1. Créer `internal/format/<nom>/` et implémenter `format.Format` (`Name`, `Handler`, `UI`).
2. L'enregistrer dans `cmd/grange/main.go` avec `reg.Register(...)`.
3. Ajouter des tests qui lancent le vrai client (`mvn`, `npm`...) contre le serveur.

## Feuille de route

1. [x] Squelette, raw, API Nexus (sous-ensemble)
2. [ ] PostgreSQL + utilisateurs, jetons, RBAC par dépôt
3. [ ] Maven (hosted + proxy), puis npm, PyPI
4. [ ] Dépôts proxy et group, politiques de nettoyage, GC des blobs
5. [ ] OCI/Docker
6. [ ] Interface web (consomme uniquement l'API REST)
7. [ ] S3/MinIO, importeur de migration depuis Nexus

## Licence

À choisir (Apache-2.0 ou AGPL-3.0) : ajouter le fichier `LICENSE` correspondant.
Remplacer `github.com/boubou1100/grange` dans `go.mod` par ton vrai chemin de dépôt.
