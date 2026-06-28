// file: pkg/diskimg/writer.go

package diskimg

import (
	"errors"
	"io"
	"os"
)

// SaveToFile writes the disk image to a file.
func (di *DiskImage) SaveToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return di.Save(f)
}

// Save writes the disk image as a standard ("MV - CPC") DSK.
//
// The in-memory model stores each track as a complete block (256-byte track
// information block followed by sector data). For a uniform-geometry +3 disk
// every track is the same size, so the standard container is sufficient and
// simplest; tracks are written verbatim from the stored blocks.
func (di *DiskImage) Save(w io.Writer) error {
	// Persist the in-memory directory to the directory sectors before writing.
	if err := di.FlushDirectory(); err != nil {
		return err
	}

	trackCount := int(di.Header.TracksNum) * int(di.Header.SidesNum)
	trackSize := 256 + SectorsPerTrack*BytesPerSector

	// Disc information block (256 bytes).
	dib := make([]byte, 256)
	copy(dib[0:], "MV - CPCEMU Disk-File\r\nDisk-Info\r\n")
	creator := di.Header.Creator[:]
	if len(creator) == 0 || creator[0] == 0 {
		creator = []byte("plus3")
	}
	copy(dib[0x22:0x30], creator)
	dib[0x30] = byte(trackCount)
	dib[0x31] = di.Header.SidesNum
	dib[0x32] = byte(trackSize & 0xFF)
	dib[0x33] = byte(trackSize >> 8)
	if _, err := w.Write(dib); err != nil {
		return errors.New("failed to write disc information block")
	}

	// Track blocks, verbatim.
	for i := 0; i < trackCount; i++ {
		block := di.Tracks[i]
		if block == nil {
			// Absent track - emit a formatted empty track.
			block = make([]byte, trackSize)
			copy(block[0:], "Track-Info\r\n")
			for j := 256; j < trackSize; j++ {
				block[j] = 0xE5
			}
		}
		if len(block) != trackSize {
			// Normalise to the standard track size.
			nb := make([]byte, trackSize)
			copy(nb, block)
			block = nb
		}
		if _, err := w.Write(block); err != nil {
			return errors.New("failed to write track data")
		}
	}
	return nil
}
