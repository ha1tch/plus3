# plus3

[![CI](https://github.com/ha1tch/plus3/actions/workflows/ci.yml/badge.svg)](https://github.com/ha1tch/plus3/actions/workflows/ci.yml)

```
       .__               ________  
______ |  |  __ __  _____\_____  \ 
\____ \|  | |  |  \/  ___/ _(__  < 
|  |_> >  |_|  |  /\___ \ /       \
|   __/|____/____//____  >______  /
|__|                   \/       \/ 

```

## +3DOS Disk Management Tool

plus3 is a command-line utility for creating, managing, and inspecting virtual
disk images in the +3DOS format used by the ZX Spectrum +3. It can:

- **Create** blank +3DOS disk images with the standard +3 / PCW layout.
- **Add** files to an image (CODE, BASIC, screen, or raw), writing a correct
  PLUS3DOS header and managing block allocation and the directory.
- **List** the catalogue of files on an image.
- **Show** disk usage and file counts (`info`).
- **Extract** a file back to the host, byte-exact (with or without its header).
- **Detokenise** a BASIC program to readable text (`extract --basic`).
- **Delete** a file, freeing its blocks.

The disk writer produces images that a real ZX Spectrum +3 accepts; the format
handling was verified against disks written by physical +3 hardware. It is useful
for retro-computing enthusiasts and developers working with ZX Spectrum emulators.

## Requirements

- Go 1.25 or later

plus3 has a single third-party dependency: github.com/ha1tch/zentools, used by the
TAP<->disk conversion in pkg/diskimg for verified TAP encoding and decoding. The
disk-image core (the bulk of the library) depends only on the Go standard library.

## Building

```
make build         # produces ./plus3, version injected from VERSION
```

`make help` lists all targets (`build`, `test`, `check`, `cross`, `release`, ...).
Or use the script or `go` directly:

```
sh build.sh
go build -o plus3 ./cmd
```

## Testing

```
go test ./...
```

The suite is small and behaviour-focused: a byte-exact create/add/list/extract
round-trip, a read of a disk written by a real +3, the BASIC detokeniser (including
numeric-constant handling and the 128K `PLAY`/`SPECTRUM` tokens), and compliance
checks that lock in the format details a real +3 verified (user area 0, block-number
allocation, header version 0, the CODE header convention, and the 32-byte directory
entry).

## Usage

```
plus3 create disk.dsk                              # create a blank +3 disk image
plus3 add disk.dsk game.bin -t code --load-addr N  # add a CODE file (loads at N)
plus3 add disk.dsk prog.bas -t basic --line 10     # add a BASIC program
plus3 list disk.dsk                                # list the catalog
plus3 info disk.dsk                                # disk usage and file count
plus3 extract disk.dsk GAME.BIN -o outdir            # extract a file (byte-exact)
plus3 extract disk.dsk GAME.BIN -o outdir --strip-header  # without the +3DOS header
plus3 extract disk.dsk LOADER.BAS --basic           # detokenise BASIC to text (stdout)
plus3 delete disk.dsk GAME.BIN --force             # delete a file
plus3 --version                                    # show the version
```

File types for `add` are `code`, `basic` (tokenised), `basictext` (plain-text source, tokenised on import), `screen`, `raw`, or `auto` (by extension).

For the full reference on every command and flag, see
[`doc/MANUAL.md`](doc/MANUAL.md).

## Disk format

plus3 reads and writes the standard single-sided +3 / PCW (CP/M Plus) format:
40 tracks, 9 sectors per track, 512-byte sectors, 1 KB allocation blocks, and a
64-entry directory at the start of the data area (track 1; track 0 is the reserved
system track). The reader handles both the standard (`MV - CPC`) and extended
(`EXTENDED CPC`) `.dsk` container variants; the writer emits the standard variant.
Files carry a PLUS3DOS header.

For the obscure and easily-misread parts of the +3DOS format -- the traps that a
real Spectrum +3 catches but a software round-trip does not -- see
[`doc/PLUS3DOS-PITFALLS.md`](doc/PLUS3DOS-PITFALLS.md).

## Using as a library

The `pkg/diskimg` package can be embedded in other Go programs (emulators,
assemblers, build tools) to read and write +3DOS images directly. See
[`doc/LIBRARY-USAGE.md`](doc/LIBRARY-USAGE.md) for the API guide and worked
examples.

## Versioning and releases

The canonical version lives in the `VERSION` file (`MAJOR.MINOR.PATCH`).
`tools/syncver.sh` propagates it to `internal/version/version.go`, and
`release.sh <version>` runs the full validate -> build -> verify -> package
pipeline, producing a guarded checkpoint zip.

Continuous integration runs build, vet, and test on every push and pull request.
Pushing a version tag (`vMAJOR.MINOR.PATCH`) triggers a release workflow that
cross-compiles binaries for Linux, macOS, Windows, and the BSDs (FreeBSD, OpenBSD,
NetBSD) across amd64 and arm64 where applicable, and attaches them to a GitHub
Release.

## Contact

Email: h@ual.li

https://oldbytes.space/@haitchfive

## License

Copyright 2026 h@ual.li

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
