// file: pkg/diskimg/reader.go

package diskimg

import (
	"errors"
	"io"
	"os"

	"github.com/ha1tch/plus3/internal"
)

// LoadFromFile loads a DSK image from a file.
func LoadFromFile(filename string) (*DiskImage, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return Load(file)
}

// Load reads a DSK image (standard "MV - CPC" or "EXTENDED CPC") from a reader.
//
// The two container variants differ in how track sizes are recorded:
//   - standard: a single track size in the disc-information block (offset 0x32);
//     every track is that size.
//   - extended: the disc-information block track-size field is 0, and a per-track
//     size table starts at offset 0x34 (one byte per track, value*256 = bytes;
//     0 means the track is absent).
//
// Real +3 disks (including those written by emulators and CPDRead) are almost
// always the extended variant, so both must be handled.
func Load(r io.Reader) (*DiskImage, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.New("failed to read disk image")
	}
	if len(raw) < 256 {
		return nil, errors.New("disk image too small")
	}

	di := &DiskImage{
		sectorMap: internal.NewSectorMap(),
		directory: Directory{Entries: make([]DirectoryEntry, MaxDirectoryEntries)},
	}

	// Parse the 256-byte disc information block.
	copy(di.Header.Signature[:], raw[0:34])
	copy(di.Header.Creator[:], raw[34:48])
	di.Header.TracksNum = raw[48]
	di.Header.SidesNum = raw[49]
	di.Header.TrackSize = uint16(raw[50]) | uint16(raw[51])<<8

	extended := string(raw[0:8]) == "EXTENDED"
	if !extended && string(raw[0:8]) != "MV - CPC" {
		return nil, errors.New("invalid disk image signature")
	}

	if err := di.validateHeader(extended); err != nil {
		return nil, err
	}

	trackCount := int(di.Header.TracksNum) * int(di.Header.SidesNum)

	// Determine each track's byte size.
	trackSizes := make([]int, trackCount)
	if extended {
		// Per-track size table at offset 0x34, one byte per track (value * 256).
		table := raw[0x34:]
		if len(table) < trackCount {
			return nil, errors.New("extended track size table truncated")
		}
		for i := 0; i < trackCount; i++ {
			trackSizes[i] = int(table[i]) * 256
		}
	} else {
		for i := 0; i < trackCount; i++ {
			trackSizes[i] = int(di.Header.TrackSize)
		}
	}

	totalSectors := trackCount * SectorsPerTrack
	di.allocation = newSectorAllocation(totalSectors)
	di.fileAlloc = newFileAllocation(di)
	di.Tracks = make([][]byte, trackCount)

	// Track data starts at offset 0x100; each track block is its table size.
	off := 0x100
	for i := 0; i < trackCount; i++ {
		size := trackSizes[i]
		if size == 0 {
			// Absent track (extended format) - store an empty placeholder.
			di.Tracks[i] = nil
			continue
		}
		if off+size > len(raw) {
			return nil, errors.New("track data extends past end of image")
		}
		block := make([]byte, size)
		copy(block, raw[off:off+size])
		di.Tracks[i] = block
		off += size

		// Light sanity check: the track information block signature. Match only
		// the "Track-Info" prefix - the spec specifies "Track-Info\r\n" but real
		// writers (e.g. some emulators) pad with NULs instead of CR/LF.
		if size >= 10 && string(block[0:10]) != "Track-Info" {
			return nil, errors.New("invalid track information block signature")
		}
	}

	// Populate the in-memory directory from the disk so file operations
	// (add/find/delete) see the existing entries and free slots.
	if entries, err := di.GetDirectory(); err == nil {
		copy(di.directory.Entries, entries)
	}

	di.Modified = false
	return di, nil
}

// validateHeader checks the disc-information block for a plausible +3 disk.
func (di *DiskImage) validateHeader(extended bool) error {
	// The standard +3 logical format is 40 tracks, but real .dsk images carry
	// physical tracks beyond that (commonly 40-43, up to ~45). Accept the range.
	if di.Header.TracksNum < TracksPerSide || di.Header.TracksNum > MaxTracksPerSide {
		return errors.New("invalid number of tracks for +3 format")
	}
	if di.Header.SidesNum != SidesPerDisk {
		return errors.New("invalid number of sides for +3 format")
	}
	// For the standard variant the header track size must be the +3 track size;
	// for the extended variant the header field is 0 and sizes live in the table.
	if !extended {
		expected := 256 + BytesPerSector*SectorsPerTrack // track info block + sector data
		if int(di.Header.TrackSize) != expected {
			return errors.New("invalid track size for +3 format")
		}
	}
	return nil
}
