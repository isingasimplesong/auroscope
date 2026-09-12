# AURoscope

**Un avis de sécurité sur les recettes AUR, avant de les construire avec Paru.**

AURoscope s'utilise dans le terminal à la place de `paru` pour rechercher, installer
ou mettre à jour des paquets. Avant une construction AUR, il fait examiner la recette
par Codex CLI, affiche son évaluation et vous laisse décider de poursuivre ou non.
Paru conserve la recherche, la sélection, les dépendances, la construction et
l'installation avec Pacman. Les paquets des dépôts officiels ne sont jamais audités.

## Ce que l'audit couvre — et ne garantit pas

L'audit porte sur le `PKGBUILD`, les fichiers suivis de la recette, les scripts
auxiliaires et leurs changements : provenance des sources, contrôles d'intégrité,
commandes, permissions, services et accès sensibles. Il inclut le contexte complet
de la recette et, lorsqu'une installation précédente a réussi, le diff depuis
cette référence. AURoscope n'exécute pas ces fichiers pour préparer l'audit.

**Un avis favorable n'est pas une garantie de sécurité.** Le modèle peut se tromper
ou manquer un problème. Il ne certifie ni le logiciel amont ni les binaires téléchargés.
AURoscope n'isole pas la construction dans un bac à sable et ne protège pas d'un
compte local compromis. Le contrôle final refuse une recette différente de celle
approuvée, sans supprimer toute possibilité de modification ultérieure.

Les constructions de recettes locales (`-B`, `--build`, chemins locaux) sont hors
périmètre. Codex CLI est actuellement le seul moteur d'audit implémenté ; les autres
fournisseurs décrits dans les décisions d'architecture ne sont pas encore disponibles.

## Prérequis

- Arch Linux **x86_64**, avec `base-devel` et `git` pour construire le paquet.
- Un Paru compatible, actuellement testé avec `paru-git` au commit
  `9ac3578807a87858651e81a02586ceb947686e7c`.
  **Paru stable 2.1.0 est incompatible** : il mélange le menu de recherche et les
  cibles sélectionnées. Le nom du paquet ou sa version ne suffit pas à prouver la
  compatibilité ; AURoscope vérifie aussi les sorties requises lors de l'utilisation.
- `codex` accessible dans le `PATH` et authentifié pour votre utilisateur.
  La bannière attendue est `codex-cli MAJOR.MINOR.PATCH`, avec une version stable
  **au moins égale à 0.150.1**. Il n'y a pas de plafond, mais accepter une version
  ne signifie pas qu'elle a été testée. Les essais réels de 0.153.4 et leurs limites
  sont décrits dans le [contrat Codex](docs/dependency-notes/codex-cli-contract.md).

## Installer

La recette est hébergée dans ce dépôt, pas publiée sur `aur.archlinux.org`.
Exécutez les commandes de construction avec votre utilisateur habituel, **pas root**.
Pacman demandera les droits nécessaires à l'installation des dépendances et du paquet.

Si Paru est déjà installé, commencez par installer le fournisseur compatible :

```sh
sudo pacman -S --needed base-devel git
paru -S paru-git
```

Si vous n'avez pas encore Paru, construisez d'abord `paru-git` depuis sa recette AUR,
après l'avoir examinée :

```sh
git clone https://aur.archlinux.org/paru-git.git
cd paru-git
makepkg -si
cd ..
```

Installez ensuite Codex par la méthode de votre choix. Par exemple, avec Node.js/npm
et un emplacement global npm accessible à votre utilisateur :

```sh
npm install -g @openai/codex
codex login
codex --version
```

Codex n'est volontairement pas une dépendance Pacman du paquet AURoscope : son
installation et son authentification restent à votre charge.

Enfin, examinez la recette puis construisez AURoscope, sans lui demander de s'auditer
pour sa propre installation :

```sh
git clone https://git.2027a.net/2027a/auroscope.git
cd auroscope/packaging/aur
makepkg -si
command -v auroscope
pacman -Q auroscope
```

`makepkg` utilise un instantané immuable des sources, identifié dans le `PKGBUILD`,
pas automatiquement tout le code du clone. Voir le
[guide du paquet](packaging/aur/README.md)
pour les vérifications de livraison et les problèmes de dépendance Paru.

## Utiliser au quotidien

Lancez AURoscope explicitement lors des premiers essais ; ne remplacez pas encore
`paru` par un alias. Les exemples suivants conservent les interactions de Paru :

```sh
auroscope                 # mise à jour officielle, puis mises à jour AUR auditées
auroscope -Syu            # même déroulement de mise à jour
auroscope visual studio   # recherche interactive et sélection numérotée par Paru
auroscope -S paru-git     # installation explicite d'un paquet AUR
auroscope -S git          # paquet officiel : installation native, sans audit
auroscope -Q              # consultation des paquets installés, sans audit
```

Les opérations qui ne construisent pas de recette AUR sont transmises à Paru.
`auroscope --version` affiche donc la version de **Paru** ; utilisez
`pacman -Q auroscope` pour connaître la version du paquet AURoscope installé.

### Pendant un audit

Pour chaque base de paquet AUR, y compris les dépendances AUR, AURoscope affiche
un résumé et un niveau de risque. Le menu accepte le mot anglais, son initiale,
ou son numéro dans l'ordre affiché :

| Choix | Effet |
| --- | --- |
| `approve` / `a` / `1` | Approuver cette recette exacte pour la transaction. |
| `inspect` / `i` / `2` | Afficher constats, incertitudes et diff, puis choisir à nouveau. |
| `edit` / `e` / `3` | Ouvrir la recette dans `$EDITOR`, puis refaire l'audit. |
| `skip` / `s` / `4` | Ne pas construire cette base de paquet dans la transaction. |
| `cancel` / `c` / `5` | Annuler la phase AUR. |

Paru reprend ensuite la résolution et conserve sa confirmation finale d'installation.
Une dépendance ignorée peut empêcher l'installation des autres paquets. La référence
utilisée pour les prochains diffs n'avance qu'après la réussite de Paru.

Si Codex échoue ou renvoie un rapport invalide, choisissez `retry`, `skip` ou `cancel`.
Il n'existe pas de poursuite automatique sans audit valide. Si le contrôle final
constate une recette modifiée après approbation, `retry` la récupère à nouveau,
relance l'audit et exige une nouvelle approbation avant de réessayer.

Une annulation de la phase AUR ne revient pas sur une mise à jour officielle déjà
réussie. Des paquets AUR différés peuvent alors nécessiter une mise à jour pour
rester compatibles avec les bibliothèques officielles.

## Configuration actuelle

Aucun fichier de configuration propre à AURoscope n'est encore implémenté.
L'authentification de Codex reste gérée par Codex. AURoscope ne propose pas encore
ses propres réglages de modèle ou de prompt et ne transmet pas d'option `--model`.
Les réglages disponibles dans le code actuel sont des variables d'environnement :

- `EDITOR` : exécutable lancé sur le dossier de recette pour `edit`, sans arguments
  intégrés dans la valeur. Exemple : `export EDITOR=nvim`.
- `AUROSCOPE_PARU` et `AUROSCOPE_CODEX` : chemins des exécutables ; valeurs par
  défaut `paru` et `codex`, recherchées dans le `PATH`.
- `AUROSCOPE_STATE` : chemin de la base SQLite des audits et références réussies.
  Par défaut : `$HOME/.local/state/auroscope/state.sqlite3`.
  `XDG_STATE_HOME` n'est pas lu par l'implémentation actuelle.
- `AUROSCOPE_CLONE_DIR` : dossier de travail des recettes. Par défaut :
  `auroscope-clones` dans le répertoire temporaire système (`/tmp` habituellement).

Paru conserve sa configuration native ; AURoscope ajoute une configuration privée
pour le dossier des recettes et le contrôle final. Un audit Codex dispose d'un délai
fixe de cinq minutes. Aucun réglage supplémentaire n'est nécessaire pour commencer.

## Mettre à jour AURoscope

Depuis la racine de votre clone, après publication d'une nouvelle recette sur `main` :

```sh
git pull --ff-only
cd packaging/aur
makepkg -si
pacman -Q auroscope
```

Forcer la reconstruction d'une ancienne recette ne change pas son instantané source.

## Documentation et aide

- [Documentation technique et historique](docs/README.md)
- [Spécification](docs/specification.md)
- [Architecture minimale](docs/architecture/minimal-v1.md)
- [Décisions d'architecture et statut](docs/decisions/README.md)
- [Installation et validation du paquet Arch](packaging/aur/README.md)
- [Signaler un problème](https://git.2027a.net/2027a/auroscope/issues)

Pour un signalement, indiquez la commande, les versions retournées par
`pacman -Q auroscope`, `paru --version` et `codex --version`, puis le message d'erreur.
Ne joignez ni identifiants d'authentification ni données privées.
