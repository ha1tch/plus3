# Using the `diskimg` package from your own Go programs

`github.com/ha1tch/plus3/pkg/diskimg` is the library that backs the `plus3`
command-line tool. It can be embedded directly in other Go programs that need to
read or write +3DOS disk images -- an emulator loading a disk, an assembler writing
its output straight onto an image, a build tool packaging a release, and so on.

This guide is the consumer-facing companion to the API. For the format itself and
its pitfalls, see [`PLUS3DOS-PITFALLS.md`](PLUS3DOS-PITFALLS.md); this document is
about the Go API.

```go
import "github.com/ha1tch/plus3/pkg/diskimg"
```

Requires Go 1.25 or later.

---

## Maturity: which paths are proven

The package exposes more than the command-line tool uses. To set expectations
honestly:

- **Proven** -- exercised by the CLI and covered by tests, and verified against real
  +3 hardware: creating an image, importing CODE and BASIC files, listing the
  directory, exporting files byte-exactly, deleting files, detokenising BASIC, and
  raw sector read/write. Build on these with confidence.
- **Present but less travelled** -- screen import/export and TAP<->disk conversion
  exist and compile, but have not had the same hardware verification. Treat them as
  usable starting points, not settled contracts, and test against real targets
  before relying on them.

When in doubt, the rule that governed the whole project applies: verify against a
real disk or a real machine, because a reader and writer that share an assumption
will agree with each other even when both are wrong.

---

## The core type: `DiskImage`

A `*DiskImage` is the in-memory handle for one disk image. You obtain one in one of
three ways:

```go
di := diskimg.NewDiskImage()                       // a fresh, formatted blank disk
di, err := diskimg.LoadFromFile("game.dsk")        // load from a path
di, err := diskimg.Load(reader)                    // load from any io.Reader
```

`NewDiskImage` returns a fully formatted single-sided +3 image (every track has a
track-information block and 0xE5-filled sectors, and the directory area is
initialised). You can import files into it immediately.

Changes are in memory until you write them out:

```go
err := di.SaveToFile("game.dsk")                   // write to a path
err := di.Save(writer)                             // write to any io.Writer
```

`Save` flushes the in-memory directory to its sectors before writing, so you do not
need to call `FlushDirectory` yourself in the normal path.

---

## Common tasks

### Create a disk and add a machine-code file

This is the most common path for an assembler or build tool: take a binary, write it
onto a disk as a CODE file with a correct PLUS3DOS header.

```go
di := diskimg.NewDiskImage()

// Import a host file as a CODE file with the given load address.
// The on-disk name is derived from the host filename (first 8 chars + ".BIN").
if err := di.ImportCode("build/game.bin", 0x8000); err != nil {
    return err
}

if err := di.SaveToFile("game.dsk"); err != nil {
    return err
}
```

The resulting file has file type 3 (CODE), the load address in the first header
parameter, and the conventions a real +3 expects (see the pitfalls document for why
those details matter).

### Add a BASIC program

```go
// line is the auto-run line number, or 0x8000 for "no auto-run".
err := di.ImportBasicProgram("loader.bas", 10)
```

This imports an **already-tokenised** BASIC file. To import plain-text BASIC
source and have it tokenised on the way in, use `ImportBasicText`:

```go
err := di.ImportBasicText("loader.txt", 10) // tokenises, then stores
```

Or tokenise in memory directly with `TokeniseBasic` (the inverse of
`DetokeniseBasic`):

```go
tok, err := diskimg.TokeniseBasic(`10 PRINT "HELLO"`)
```

The tokeniser covers keywords, integer constants (0-65535), string literals, and
REM comments; it does not produce floating-point literals, DEF FN calculator
slots, or embedded colour-control bytes.

### List the catalogue

`GetDirectory` returns a snapshot slice of all 64 entries. Filter out the empty
ones:

```go
entries, err := di.GetDirectory()
if err != nil {
    return err
}
for _, e := range entries {
    if e.IsUnused() || e.GetFilename() == "" {
        continue
    }
    fmt.Println(e.GetFilename())                    // e.g. "GAME.BIN"
}
```

`DirectoryEntry` is read via its methods: `GetFilename()` (the 8.3 name),
`IsUnused()`, `IsDeleted()`, and `GetAttributes()` (read-only / hidden / system).
The raw fields (`Status`, `RecordCount`, `AllocationBlocks`) are also exported if you
need them, but prefer the methods.

The returned slice is a copy; mutating it does not change the disk. Use the
`DiskImage` methods (`ImportCode`, `DeleteFile`, ...) to modify the image.

### Extract a file back to the host

```go
// stripHeader removes the 128-byte PLUS3DOS header so you get just the data.
// The second argument is a host FILE path (the file is created), not a directory.
err := di.ExportFile("GAME.BIN", "outfile.bin", true)
```

Extraction uses the PLUS3DOS header's length field, so it is byte-exact rather than
rounded to the record size.

### Detokenise a BASIC program to text

```go
text, err := di.ReadBasicText("LOADER.BAS")        // reads + detokenises in one call
fmt.Print(text)
```

Or, if you already have the raw (header-stripped) program bytes in memory -- for
example bytes you read some other way -- call the decoder directly:

```go
text, err := diskimg.DetokeniseBasic(programBytes)
```

`DetokeniseBasic` handles keyword tokens (including the 128K `PLAY` and `SPECTRUM`),
numeric constants (skipping the hidden 5-byte binary form), strings, and statement
separators. It does not fully reconstruct embedded colour-control argument bytes.

### Delete a file

```go
err := di.DeleteFile("GAME.BIN")                   // frees blocks, flushes directory
```

---

## Lower-level access: sectors and the File handle

Two lower-level surfaces are available when the file-level API is not enough.

### Raw sector I/O

Direct sector read/write, addressed by track and sector. Sectors are 512 bytes;
sector indices passed here are 0-based within the track. This is what an emulator's
FDC layer would sit on top of.

```go
data, err := di.GetSectorData(track, sector, side) // returns 512 bytes
err = di.SetSectorData(track, sector, side, data)  // data must be 512 bytes
```

For the geometry (track 0 reserved, directory on track 1, the block-to-sector
mapping), see the pitfalls document -- those rules matter if you compute sector
addresses yourself.

### Streaming file access with `*File`

`OpenFile` returns a `*File` that implements the standard `io` interfaces
(`io.Reader`, `io.Writer`, `io.Seeker`, plus `ReaderAt`/`WriterAt`), so you can
stream rather than load whole files:

```go
f, err := di.OpenFile("GAME.BIN", false)           // false = must already exist
if err != nil {
    return err
}
defer f.Close()

buf := make([]byte, 256)
n, err := f.Read(buf)
```

Pass `createNew = true` to create the file if it does not exist. After writing,
`Close` updates the directory entry. Remember to `SaveToFile` / `Save` afterwards to
persist the image.

---

## Validation

```go
err := di.DiskCheck()                               // structural sanity check
ok  := di.IsPlus3Format()                           // is this the +3 format?
```

`DiskCheck` is the same check the `info` command runs. It is a sanity check on the
image structure, not a guarantee of +3DOS acceptance -- the only guarantee of that
is a real +3 (which is the lesson the pitfalls document exists to pass on).

---

## A complete example

Create a disk, add a loader and a code file, and save it -- the shape an ecosystem
build step would take:

```go
package main

import (
    "log"

    "github.com/ha1tch/plus3/pkg/diskimg"
)

func main() {
    di := diskimg.NewDiskImage()

    if err := di.ImportCode("build/game.bin", 0x8000); err != nil {
        log.Fatalf("import code: %v", err)
    }
    if err := di.ImportBasicProgram("build/loader.bas", 10); err != nil {
        log.Fatalf("import loader: %v", err)
    }

    if err := di.SaveToFile("dist/game.dsk"); err != nil {
        log.Fatalf("save: %v", err)
    }

    // Confirm what landed on the image.
    entries, _ := di.GetDirectory()
    for _, e := range entries {
        if !e.IsUnused() && e.GetFilename() != "" {
            log.Printf("on disk: %s", e.GetFilename())
        }
    }
}
```

---

## Notes for emulator and assembler integration

A few points specific to the two most likely first consumers:

- **Emulator (loading a disk):** the proven read path is `LoadFromFile` ->
  `GetDirectory` -> `ExportFile`/`OpenFile`, or `GetSectorData` for raw FDC-level
  access. The reader handles both the standard (`MV - CPC`) and extended
  (`EXTENDED CPC`) `.dsk` container variants; real disks are usually extended. The
  writer emits the standard variant, which round-trips and is accepted by a real +3.

- **Assembler (writing output onto a disk):** the proven write path is
  `NewDiskImage` -> `ImportCode` (or build a header and use `ImportFile` with
  `ImportOptions`) -> `SaveToFile`. If you need the CODE header parameters set a
  particular way, `ImportOptions{AddHeader: true, FileType: FileTypeCode, LoadAddr:
  addr}` gives you control; the package fills in the header-version and
  second-parameter conventions a real +3 expects.

- **Both:** the package tokenises plain-text BASIC source into the on-disk form
  (`TokeniseBasic` / `ImportBasicText`) and detokenises the other way
  (`DetokeniseBasic` / `ReadBasicText`). The tokeniser covers keywords, integer
  constants, strings, and REM; it does not emit floating-point literals or DEF FN
  calculator slots, so programs relying on those should be tokenised with a full
  toolchain and added with `-t basic`.
