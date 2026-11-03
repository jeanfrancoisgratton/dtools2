# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) since version 2.7.1.

## [3.3.1] - 2026-11-03

### Changed

- `dtools stop/stopall` and `dtools kill/killall` used to share the same output: `container XYZ STOPPED`. Now the proper verb (STOPPED or KILLED) is used for each case:
  - container XYZ **STOPPED**
  - container XYZ **KILLED**

## [3.3.0] - 2026-11-02

### Added

- Unit tests across all packages (fake docker daemon in `src/internal/testutil`)
- `inspv`, `inspn` and `inspi` shortcuts for the volume, network and image `inspect` (`insv` is kept)

### Changed

- Failing commands now exit with code 1, like docker (they used to exit 0)
- `version`, `blacklist`, `env`, `get`, `login` and `completion` no longer contact the daemon; "Connected to" is only shown once connected
- Bare `inspect` now always means containers (it could resolve to the volume, network or image `inspect`)
- `run -v` now creates a missing host directory (like docker), resolves relative and `~` paths, and keeps mode options (`:z`, `:Z`...)
- Public images on an explicit registry (quay.io, ghcr.io...) can be pulled without a stored login

### Fixed

- `login` no longer drops the other settings of `~/.docker/config.json` (`credsStore`, `currentContext`, `proxies`, ...)
- `load` never uploaded the archive
- `pause` / `unpause` always failed with HTTP 404
- Crash of `rmn` on a network still in use; daemon errors are now shown as-is
- `-B` now works in quiet mode for `rmv`, `rmi` and `rmn`, so `prune -B` and `clean -B` remove blacklisted resources
- Blacklist: `rmi nginx` is now blocked when `nginx` was blacklisted; names with uppercase letters are now protected; `image` and `images` behave the same
- `run -p` accepts IPv6 host addresses (`[::1]:8080:80`)
- `lsc --format Name` now prints the container names
- `prune` and `clean` restore quiet mode after a failure; `clean` honours `-q` and reports accurate counts
- Registry commands honour `$DOCKER_CONFIG`
- `inspv` works as documented
- `lsi --json` no longer outputs malformed keys such as `"ParentId.omitempty"`

### Security

- `cp` from a container could write files outside the destination through a symlink in the archive

## [3.2.1] - 2026-11-02

### Changed

- `startall` now only targets non-running containers
- `restartall` now only restarts the containers that were running when invoked; it used to also start containers that were already stopped

### Fixed

- `restart -k` / `restartall -k`: containers were killed but never restarted, as the daemon still reported them running right after the kill; dtools now waits for them to exit before starting them
- `start` no longer silently ignores a container that is already running (warning) or that does not exist (error)

## [3.2.0] - 2026-10-18

### Added

- New `dtools volume backup` and `dtools volume restore` commands, folded in from `dvol` (which is being phased out); they use dtools' own REST client, so `-H`, TLS, `-V` and the timeout flags apply
- Backups can now be xz-compressed (`.xz`/`.txz`), not only gzipped

### Changed

- Unlike `dvol`, every daemon response is validated; on failure, the temp container and partial archive are cleaned up, and stopped containers are restarted unless the volume was already reset
- Restoring a blacklisted volume issues a warning and proceeds

### Fixed

- Restoring a volume still referenced by stopped containers now wipes its content instead of merging the archive into the old data
- The archive is opened before the volume is reset, so a wrong path no longer destroys the volume

## [3.1.1] - 2026-09-21

### Added

- New `dtools backend active` command to show the currently selected backend on demand

### Changed

- The routine "Using backend: ..." message is no longer printed for the expected case (explicit `--runtime`, or auto-detect resolving to docker/podman); it's now only shown when auto-detect falls back to containerd, since that's the case worth knowing about

## [3.0.2] - 2026-09-21

### Changed

- The backend probe and _normal_ clients now use different fast-fail http timeouts

## [3.0.1] - 2026-09-21

### Added

- New `containerd` backend (`--runtime containerd`), auto-detected as a fallback when no docker/podman daemon is reachable
- New `Backend` abstraction: `container`/`image`/`system` commands now work against docker, podman, or containerd; `network`, `volume`, `run`, `build`, `exec`, `logs` and `cp` remain docker/podman-only and report "not supported" on containerd
- New flags: `--runtime`, `--containerd-socket`, `--namespace`/`-N`, `--all-namespaces`/`-A` (containerd namespace selection, list-only)

### Changed

- **Breaking:** `-A` is now `--all-namespaces` (was `--api-version`); `--api-version` moved to `-V`

### Removed

- **Breaking:** the `-V`/`--version` flag; use `dtools version` instead

## [2.8.0] - 2026-08-10

### Fixed

- `--rm`, `--network` and a host of other ignored flags in the `run` subcommand
- The `build` command now fully works with podman and docker's BuildKit

## [2.7.2] - 2026-08-08

### Changed

- `lsb` output is now displayed in JSON instead of tables

### Removed

- Shell completion packaging scripts

## [2.7.1] - 2026-07-17

### Added

- Test files

### Changed

- Version numbering now aligned with SemVer
- Packaging scripts enhancements

## [2.70.00] - 2026-05-01

### Changed

- Blacklist subcommands behaviour changes
- Modified output of `lsc`

## [2.60.00] - 2026-03-31

### Added

- Extra capabilities (flags) to `dtools run`

### Changed

- GO version bump -> 1.26.1
- Upgraded to helperFunctions/v5

## [2.52.00] - 2026-03-03

### Changed

- Temp closure of development of dtools, with `dtools build` still not working

## [2.51.00] - 2026-02-14

### Fixed

- `dtools run` and `dtools build`

## [2.50.00] - 2026-02-01

### Added

- `inspect` feature on containers (more objects to come)

### Changed

- Updated to GO 1.26

### Fixed

- http timeouts on long http transactions
- http 409 on `dtools kill`

## [2.40.01] - 2026-01-30

### Added

- Completed the `commit`, `load`, and `save` commands

### Changed

- Removed duplicate code

## [2.31.00] - 2026-01-30

### Added

- Environment support (`-e` flag) in `dtools run`
- New feature: `dtools attach`

## [2.30.00] - 2026-01-29

### Added

- Completed the `cp` command

### Changed

- Moved `sys catalog` and `sys tags` to the new `get` subcommand

## [2.21.01] - 2026-01-07

### Added

- The `--json [--format] --file` output handlers

## [2.20.00] - 2026-01-06

### Added

- New features: `rm`, `clean` and `sys info`

## [2.12.00] - 2026-01-04

### Added

- The forgotten `volume create` command

## [2.11.00] - 2026-01-04

### Added

- The (forgotten !) `build` command

## [2.00.00] - 2026-01-03

### Changed

- Complete rewrite

## [0.10.00] - 2025-11-11

### Added

- Initial version
