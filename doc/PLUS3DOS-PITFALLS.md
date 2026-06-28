# +3DOS disk images: the parts that bite you

This document collects the things that were obscure, badly documented, or simply
absent from the references while implementing a +3DOS disk-image writer and reader.
The official Part-27 guide (`plus3dos.txt`) is the reference; this is the companion
that explains where it is easy to go wrong.

The recurring lesson runs through everything below: **a disk image that round-trips
perfectly through your own reader and writer can still be rejected by a real
Spectrum +3.** A reader and writer that share the same wrong assumption will agree
with each other indefinitely. The only authority is real +3DOS — either a physical
machine, or a disk written by one that you can diff against. Several of the points
below were invisible to software testing and only surfaced on hardware.

The format is the standard 180K single-sided +3 / PCW (CP/M Plus) layout: 40
tracks, 9 sectors per track, 512-byte sectors, 1 KB allocation blocks, a 64-entry
directory. Everything here assumes that layout.

---

## 1. The directory is on track 1, not track 0

Track 0 is the reserved system track (XDPB `OFF = 1`). The directory lives at the
start of the **data area**, which begins on track 1. It is easy to assume the
directory is "at the beginning" and write it to track 0; a real +3 then reports the
disk as empty because it looks for the catalogue on track 1.

The directory occupies the first 4 sectors of track 1 (2 KB = 64 entries x 32
bytes).

## 2. A directory entry is exactly 32 bytes — and the field layout is rigid

The canonical CP/M directory entry is 32 bytes:

```
offset size field
  0      1   status / user number
  1      8   filename (space-padded)
  9      3   extension (space-padded; high bits are attribute flags)
 12      1   Xl  - extent number, low byte
 13      1   Bc  - byte count in the last record
 14      1   Xh  - extent number, high byte
 15      1   Rc  - record count in this extent (number of 128-byte records)
 16     16   Al  - allocation: the block NUMBERS used by this extent
```

That sums to 32. If you model the entry with extra convenience fields (a separate
"first record" byte, a 16-bit "logical size", etc.) and then serialise the struct
directly, you get a 33-, 34-, or 35-byte entry. Every entry after the first is then
shifted, and the whole directory is corrupt — but your own reader, using the same
oversized struct, will parse it back without complaint. Serialise to exactly 32
bytes and verify it (a one-line test: serialise an empty entry, assert length 32).

The file's byte-exact length is **not** in the directory. `Rc` gives the length
only to the nearest 128-byte record. For an exact length, read the PLUS3DOS header
(see section 6).

## 3. The status byte is the user number — and the default is 0, not 1

This one is genuinely a trap. The status byte (offset 0) is the CP/M **user area**
(0-15), with `0xE5` meaning "unused/deleted". The +3 catalogues **user 0** by
default. If you write a file with status `0x01` (a natural-looking "this slot is in
use" value), the file is physically present and correct, but `CAT` shows nothing —
the +3 is looking in user 0 and your file is in user 1. Write files as user **0**
(`0x00`).

The consequence ripples: because a valid file's status is `0x00`, you cannot use
"status == 0x00" to mean "free slot". A free slot is `0xE5` **or** a zero entry with
a blank name. A real user-0 file has status `0x00` *and a non-blank name* and must
not be treated as free, or your next "add" will overwrite it.

## 4. The allocation field holds block NUMBERS, not a count

`Al` (offset 16, 16 bytes) is a list of the allocation-block numbers the file
occupies, in order — e.g. `03 04 05 06 07 08 09 00 ...` for a file in blocks 3-9.
It is **not** a count of blocks and **not** a length. Storing the block count there
(a single small number) produces a directory entry that points at the wrong place;
the file data is written correctly elsewhere, but the +3 follows the bogus pointer
and finds filler. For 1 KB blocks the numbers fit one per byte.

## 5. Block-to-sector mapping: blocks are numbered from the data area

An allocation block is two 512-byte sectors. Block numbering starts at the data
area (track 1), **not** at the physical start of the disk. The mapping that a real
disk actually uses is:

```
linear_sector = block * 2          (+ 0 or 1 for the two sectors in the block)
track  = 1 + linear_sector / 9     (the 1 accounts for the reserved track 0)
sector = linear_sector % 9         (0-based here; physical sector IDs are 1..9)
```

The way to be sure is to take a real disk, read a file's `Al` block number from its
directory entry, and confirm that mapping lands on the sectors that actually hold
the file's data (the first of which begins with the `PLUS3DOS` header). Reasoning
from the spec alone is where the off-by-one in the reserved track creeps in.

Sectors are physically numbered **1 to 9** (R=1..9), not 0 to 8. Off-by-one here
silently reads or writes the wrong sector.

## 6. The PLUS3DOS file header: version 0, and what "wrong file type" really means

Files may carry a 128-byte header record in their first 128 bytes:

```
0..7    'PLUS3DOS' signature
8       0x1A  (soft-EOF)
9       issue number
10      version number
11..14  total file length, 32-bit little-endian (header + data)
15..22  the 8-byte +3 BASIC header (see below)
23..126 reserved (zero)
127     checksum = sum of bytes 0..126, modulo 256
```

Two things here cost real debugging time:

**The version must be 0.** The spec says "the version number must be less than or
equal to the software's version number." +3DOS is version 0, so a header declaring
version **1** is rejected. The failure does not say "bad version" — it surfaces as
**"Wrong file type"**, because a rejected header makes +3DOS treat the file as
headerless / not what the command expected. If `LOAD ""CODE` returns "Wrong file
type" on a file you believe is correct, check the version byte before anything else.
Real +3-written files use issue 1, version 0.

**The length field is the total, including the header.** Bytes 11..14 are
`128 + data_length`, not the data length alone. This is the authoritative file
length; use it for byte-exact extraction rather than `Rc * 128`, which rounds up.

The checksum must be present and correct, or the header is not recognised as a
header at all (and again you get headerless/wrong-type behaviour rather than a
checksum error).

## 7. The 8-byte BASIC sub-header, especially for CODE files

Bytes 15..22 of the PLUS3DOS header are the same 8 bytes the Spectrum's tape header
uses:

```
15      file type   (0 = BASIC program, 1 = numeric array,
                     2 = character array, 3 = CODE/bytes)
16..17  length of the data
18..19  parameter 1
20..21  parameter 2
```

For **CODE** files (type 3), parameter 1 is the load address. Parameter 2 is the
subtle one: real +3DOS writes **0x8000 (32768)** there. Leaving it 0 is not
obviously wrong and may even load, but it does not match what the machine itself
produces. The way this was pinned down was by saving a CODE file from a real +3 onto
an image and diffing its header — parameter 2 came back as 0x8000 every time. When
in doubt about a header field, let the +3 write one and diff it; that beats guessing
from the spec.

For BASIC programs (type 0), parameter 1 is the auto-run line (or 0x8000 / >=32768
meaning "no auto-run") and parameter 2 is the length of the program without
variables.

## 8. The `.dsk` container: standard vs extended

The on-disk `.dsk` file is a host-side container, distinct from the CP/M filesystem
inside it. There are two variants and real +3 disks are almost always the second:

- **Standard** — signature begins `MV - CPC`. A single track size in the header
  applies to every track.
- **Extended** — signature begins `EXTENDED CPC`. A per-track size table starts at
  offset 0x34, one byte per track, each multiplied by 256 to get that track's size;
  a value of 0 means the track is absent.

A reader that only handles the standard variant will fail on most real disks. Note
also that the track-information block within each track is identified by a
`Track-Info` signature that real writers pad with `NUL` (not `CR`/`LF`), so match on
the prefix rather than an exact string.

A standard +3 data disk leaves the disk-specification/boot sector as `0xE5` filler
and logs on via the default XDPB; you do **not** need to synthesise a boot/spec
sector for a plain data disk. The bootable-disk checksum (the byte that makes
sector 0 sum to 3 modulo 256) only applies to actually-bootable disks; do not apply
that check to ordinary data disks whose first sector is `0xE5` filler.

## 9. Tokenised BASIC: the hidden bytes inside a number

Detokenising a BASIC program (turning the stored bytes back into readable text) is
mostly a simple walk:

```
each line:  [line number: 2 bytes, BIG-endian]
            [text length: 2 bytes, little-endian]
            [text ... ]
            [0x0D]
```

Note the line number is **big-endian** while the length is little-endian — a mix
that is easy to get wrong. Keyword tokens are single bytes 0xA3..0xFF.

The trap is numbers. When a program contains a numeric constant, it is stored as its
**visible ASCII digits** followed by a marker byte `0x0E` and then **5 bytes** of
the binary value. A detokeniser that does not know to skip those 5 bytes after a
`0x0E` will dump them as garbage characters into the output. The reason this is easy
to miss: a `REM`-only line, or any line without numbers, decodes perfectly without
handling `0x0E` at all — so the code looks correct until the first real program with
a number in it. Test the detokeniser against a program that actually contains
numbers (a loader with `BORDER 0`, `CLEAR 32767`, `RANDOMIZE USR x`), not just text.

Two keyword values are 128K-only and sit just below the main range: `SPECTRUM`
(0xA3) and `PLAY` (0xA4). A 48K-only token table will mis-decode 128K programs that
use them.

---

## How to verify any of this

The single most useful technique throughout was **diffing against ground truth**:

1. Take a real disk a +3 reads, or have a +3 write a file onto an image you made.
2. Read the bytes back and compare field by field against what your code produces.
3. Where they differ, the +3 is right.

Software round-trips confirm internal consistency, which is necessary but not
sufficient. The directory geometry, the user-area default, the header version, and
the CODE parameter were all caught this way and not by any amount of
write-then-read-back testing.
