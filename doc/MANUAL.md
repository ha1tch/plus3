# plus3 manual

plus3 is a command-line tool for creating, managing, and extracting files from
+3DOS disk images (`.dsk`) as used by the ZX Spectrum +3. This manual documents
every command and flag. For the library API see
[`LIBRARY-USAGE.md`](LIBRARY-USAGE.md); for format details see
[`PLUS3DOS-PITFALLS.md`](PLUS3DOS-PITFALLS.md).

## Synopsis

```
plus3 <command> [flags] [arguments]
```

Flags may appear before or after the positional arguments, so both
`plus3 create disk.dsk --force` and `plus3 create --force disk.dsk` are accepted.

Run `plus3 <command> -h` to see the flags for any command, `plus3 --help` for the
command list, and `plus3 --version` for the version.

Numbers for `--load-addr` and `--line` accept decimal (`32768`) or hexadecimal
(`0x8000`).

## Commands

- [`create`](#create) - create a new blank disk image
- [`add`](#add) - add a file to a disk image
- [`list`](#list) - list the catalogue
- [`info`](#info) - show disk usage and details
- [`extract`](#extract) - extract a file to the host (or detokenise BASIC)
- [`delete`](#delete) - delete a file

---

### create

Create a new, blank +3DOS disk image in the standard single-sided +3 format
(40 tracks, 9 sectors per track, 512-byte sectors, 1 KB blocks, 64-entry
directory).

```
plus3 create [flags] <disk.dsk>
```

| Flag | Default | Description |
|------|---------|-------------|
| `--label <text>` | (none) | Disk label, maximum 11 characters. |
| `--boot` | off | Create a bootable disk rather than a plain data disk. |
| `--force` | off | Overwrite the output file if it already exists. |
| `--quiet` | off | Suppress non-error output. |

Examples:

```
plus3 create game.dsk
plus3 create game.dsk --label MYGAME --force
```

---

### add

Add a host file to a disk image, writing a correct PLUS3DOS header and managing
block allocation and the directory.

```
plus3 add [flags] <disk.dsk> <file>
```

| Flag | Default | Description |
|------|---------|-------------|
| `-t`, `--type <type>` | `auto` | File type: `basic`, `basictext`, `code`, `screen`, `raw`, or `auto`. |
| `--load-addr <n>` | `32768` | Load address for CODE files (decimal or `0x` hex). |
| `--line <n>` | `10` | Auto-run line number for BASIC programs. |
| `--force` | off | Overwrite an existing file of the same name. |
| `--quiet` | off | Suppress non-error output. |

`-t` and `--type` are equivalent. With `auto`, the type is chosen from the host
file's extension:

| Extension | Type |
|-----------|------|
| `.bas` | basic |
| `.bin` | code |
| `.scr` | screen |
| anything else | raw |

Type notes:

- **code** - a machine-code block. `--load-addr` sets the address it loads to.
- **basic** - an **already-tokenised** BASIC program, stored verbatim. `--line`
  sets the auto-run line (use a value of 32768 or above for no auto-run).
- **basictext** - **plain-text** BASIC source, which is tokenised on import. Each
  source line must begin with a line number, e.g.
  `10 CLEAR 32767: LOAD "game"CODE: RANDOMIZE USR 32768`. The tokeniser covers
  keywords, integer constants (0-65535), string literals, and REM comments;
  keywords inside strings and after REM are left literal. It does not produce
  floating-point literals, DEF FN calculator slots, or embedded colour-control
  bytes, so for programs that rely on those, tokenise with a full toolchain and
  add the result with `-t basic`.
- **screen** - a SCREEN$ dump. The host file must be exactly 6912 bytes (6144
  pixel bytes plus 768 attribute bytes); other sizes are rejected.
- **raw** - the bytes are stored as-is.

As a safeguard, `add` prints an advisory warning (to standard error, suppressed by
`--quiet`) when the input looks like the wrong BASIC form for the chosen type: if
`-t basictext` is given input that already parses as tokenised BASIC, or `-t basic`
is given plain-text source. The operation still proceeds exactly as asked; the
warning only flags a likely mistake.

The on-disk name is derived from the host filename (8.3, upper-cased).

Examples:

```
plus3 add game.dsk loader.bas -t basic     --line 10   # already tokenised
plus3 add game.dsk loader.txt -t basictext --line 10   # plain-text source
plus3 add game.dsk game.bin   -t code --load-addr 0x8000
plus3 add game.dsk title.scr  -t screen
plus3 add game.dsk data.dat   -t raw --force
```

---

### list

List the catalogue of files on a disk image.

```
plus3 list [flags] <disk.dsk>
```

| Flag | Default | Description |
|------|---------|-------------|
| `--sort <key>` | `name` | Sort by `name`, `size`, or `type`. |
| `--reverse` | off | Reverse the sort order. |
| `--format <fmt>` | `dos` | Output style: `dos`, `ls`, or `cpm`. |
| `--pattern <glob>` | `*` | Show only names matching the pattern, e.g. `*.BAS`. |
| `--long` | off | Show detailed per-file information. |
| `--json` | off | Output as JSON. |
| `--show-deleted` | off | Include deleted files in the listing. |
| `--show-system` | off | Include system files in the listing. |

Examples:

```
plus3 list game.dsk
plus3 list game.dsk --sort size --reverse
plus3 list game.dsk --pattern '*.BAS' --long
plus3 list game.dsk --json
```

---

### info

Show summary information about a disk image: file count, space used and free, and
format details.

```
plus3 info [flags] <disk.dsk>
```

| Flag | Default | Description |
|------|---------|-------------|
| `--validate` | on | Run a structural validation of the image. |
| `--verbose` | off | Show additional details. |
| `--json` | off | Output as JSON. |
| `--show-deleted` | off | Include information about deleted files. |

`--validate` is on by default; the check is a structural sanity check on the image,
not a guarantee that a real +3 will accept every file.

Examples:

```
plus3 info game.dsk
plus3 info game.dsk --verbose
plus3 info game.dsk --json
```

---

### extract

Extract a file from a disk image back to the host, or detokenise a BASIC program
to readable text.

```
plus3 extract [flags] <disk.dsk> <name>
```

| Flag | Default | Description |
|------|---------|-------------|
| `-o`, `--output-dir <dir>` | (current dir) | Directory to write the extracted file into. |
| `--strip-header` | off | Remove the 128-byte PLUS3DOS header, leaving just the data. |
| `--overwrite` | off | Allow overwriting an existing host file. |
| `--basic` | off | Detokenise a BASIC program to text instead of extracting raw bytes. |
| `--quiet` | off | Suppress non-error output. |

`-o` and `--output-dir` are equivalent and name a **directory** (it is created if
needed); the output filename is taken from the disk file's name. Extraction uses
the PLUS3DOS header's length field, so it is byte-exact rather than rounded to the
record size.

With `--basic`, the program is detokenised to readable text. Without `-o` the text
goes to standard output; with `-o <dir>` it is written to `<name>.txt` in that
directory.

If a file whose header marks it as a tokenised BASIC program is extracted without
`--basic`, `extract` prints an advisory warning (suppressed by `--quiet`)
suggesting `--basic`. The extraction still proceeds as asked.

Examples:

```
plus3 extract game.dsk GAME.BIN -o outdir
plus3 extract game.dsk GAME.BIN -o outdir --strip-header
plus3 extract game.dsk LOADER.BAS --basic
plus3 extract game.dsk LOADER.BAS --basic -o outdir
```

---

### delete

Delete a file from a disk image, freeing its blocks.

```
plus3 delete [flags] <disk.dsk> <name>
```

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | off | Delete without asking for confirmation. |
| `--no-recycle` | off | Do not preserve the deleted file's directory information. |
| `--quiet` | off | Suppress non-error output. |

Examples:

```
plus3 delete game.dsk GAME.BIN --force
```

---

## Exit status

plus3 returns a non-zero exit status and prints an `Error:` message to standard
error when a command fails (for example, a missing disk image, a wrong-sized
screen file, or an existing output file without `--force` / `--overwrite`). On
success it returns zero.
