# Changelog

All notable changes to plus3 are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.9.4] - 2026-06-28

### Added

- GitHub Actions CI (`.github/workflows/ci.yml`): build, vet, and test on every
  push and pull request.
- GitHub Actions release workflow (`.github/workflows/release.yml`): on a version
  tag, cross-compiles binaries for Linux, macOS, Windows, and the BSDs (FreeBSD,
  OpenBSD, NetBSD) across amd64/arm64 where applicable, with the version injected
  from the tag, and attaches them to a GitHub Release.
- CI status badge in the README.

### Changed

- `release.sh` source allowlist now includes `.github`.

## [0.9.3] - 2026-06-28

### Added

- `doc/LIBRARY-USAGE.md` - a guide to using the `pkg/diskimg` package from other
  Go programs (emulator and assembler integration in mind): constructors, the
  common file operations, raw sector and streaming `*File` access, validation, a
  complete worked example, and an honest note on which paths are hardware-proven
  versus less travelled. Every code example was compiled and run against the API.

## [0.9.2] - 2026-06-28

### Changed

- README brought into sync with the current tool: full command list (create, add,
  list, info, extract, delete, and `extract --basic`), a testing section, and a
  corrected disk-format note (the directory is on track 1; track 0 is the reserved
  system track).

## [0.9.1] - 2026-06-28

### Added

- `doc/PLUS3DOS-PITFALLS.md` - a companion to the Part-27 reference covering the
  obscure, badly-documented, or undocumented parts of the +3DOS format: the
  directory-on-track-1 placement, the 32-byte entry layout, the user-area-0
  default, block-number allocation, the data-area block mapping, the header
  version-0 requirement and what "wrong file type" really means, the CODE
  parameter-2 convention, the standard/extended container split, and the hidden
  5-byte number encoding in tokenised BASIC. Every factual claim was verified
  against real +3-written disks.

## [0.9.0] - 2026-06-28

### Added

- `extract --basic` detokenises a Sinclair BASIC program to readable text,
  printing to stdout (or writing `<name>.txt` with `-o`). Handles keyword tokens
  (including the 128K-only PLAY and SPECTRUM), numeric constants (skipping the
  hidden 0x0E + 5-byte binary form), strings, and statement separators. Verified
  against a real +3-written program and a loader-style program containing numbers.

## [0.8.0] - 2026-06-28

Cleanup and test-hardening release. The disk format is hardware-verified and the
codebase is consolidated, formatted, and covered by a focused test suite.

### Added

- Behaviour tests for the directory add operation (first-free-slot placement,
  user-area-0 enforcement, no overwrite of an existing file, full-directory error),
  written test-first.

### Changed

- Consolidated the directory add path: `Directory.AddEntry` removed (it duplicated
  `Directory.AddFile` and had no callers); `AddFile` is the single implementation.
- `go.mod` toolchain set to Go 1.25.
- Entire tree formatted with `gofmt`.

### Removed

- Dead `pkg/disk` package (superseded by `pkg/diskimg`).
- Stale pre-reconciliation tests (replaced by the current suite).

## [0.1.0] - 2026-06-28

First working release. The disk writer produces +3DOS disk images that a real
ZX Spectrum +3 accepts (verified on hardware).

### Added

- CLI with six commands: `create`, `add`, `list`, `info`, `extract`, `delete`.
- `create` formats a blank single-sided +3 disk image (40 tracks, 9x512 sectors,
  1K blocks, 64-entry directory on the reserved system track).
- `add` writes CODE, BASIC, screen, or raw files with a correct PLUS3DOS header
  (signature, issue/version, file length, BASIC sub-header, checksum).
- `extract` reads a file back out byte-exactly, using the PLUS3DOS header length.
- `delete` frees the file's blocks, marks the directory entry unused, and flushes.
- Reader supports both standard ("MV - CPC") and extended ("EXTENDED CPC") DSK
  containers; real +3 disks are almost always extended.
- Version reporting via `plus3 --version`, synced from the root `VERSION` file.

### Fixed (hardware-verified +3DOS compliance)

- Directory entries written in user area 0 (were user 1, making files invisible
  to the +3 catalog).
- Directory allocation field stores block numbers, not the block count.
- File allocation no longer over-allocates by one block on incremental writes.
- PLUS3DOS header version set to 0 (a higher version is rejected by +3DOS).
- CODE files set the second BASIC-header parameter to 0x8000, matching +3DOS.
- Directory entry corrected to the canonical 32-byte CP/M layout.
- Directory placed on track 1 (the reserved system track is track 0).

### Tests

- High-level round-trip test (create -> import -> save -> reload -> list ->
  export) that transitively exercises allocation, sector mapping, directory
  persistence, and the PLUS3DOS header.
- Reader test against a disk written by a real ZX Spectrum +3.
- Compliance tests locking in the hardware-verified invariants (user area 0,
  block-number allocation, no over-allocation, header version 0, CODE p2 = 0x8000,
  32-byte directory entry).
- Behaviour tests for the directory add operation (first-free-slot placement,
  user-area-0 enforcement, no overwrite of existing files, full-directory error).

### Notes

- The directory and header formats were verified against real disks, including a
  two-file data disk written by a physical +3 and a CODE file written by +3DOS
  into a plus3-created image.
