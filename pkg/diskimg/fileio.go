// file: pkg/diskimg/fileio.go

package diskimg

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// File represents an open file on the disk image
type File struct {
	disk       *DiskImage
	entry      *DirectoryEntry
	header     *Plus3DosHeader
	blocks     []int
	position   int64
	size       int64
	readOnly   bool
	isHeadered bool
}

// OpenFile opens or creates a file on the disk image
func (di *DiskImage) OpenFile(filename string, createNew bool) (*File, error) {
	fileEntry, err := di.directory.FindFile(filename)
	if err != nil && !createNew {
		return nil, err
	}

	if err != nil && createNew {
		// Create a new file. Split the filename into CP/M 8.3 form, space-padded.
		name, ext := splitFilename(filename)
		newEntry := DirectoryEntry{
			Status: 0x00, // user 0
		}
		copy(newEntry.Name[:], name[:])
		copy(newEntry.Extension[:], ext[:])
		if err := di.directory.AddFile(newEntry); err != nil {
			return nil, err
		}
		fileEntry, err = di.directory.FindFile(filename)
		if err != nil {
			return nil, err
		}
	}

	// Create file struct
	f := &File{
		disk:     di,
		entry:    fileEntry,
		position: 0,
		readOnly: false,
	}

	// For an existing file, populate the block list and size from its directory
	// entry so the read path knows where the data is and how much there is.
	// (For a newly created file these stay empty until data is written.)
	if fileEntry.RecordCount > 0 || fileEntry.AllocationBlocks[0] != 0 {
		for _, b := range fileEntry.AllocationBlocks {
			if b != 0 {
				f.blocks = append(f.blocks, int(b))
			}
		}
		f.size = int64(fileEntry.RecordCount) * 128
	}

	// Try to read header if it exists
	headerData := make([]byte, HeaderSize)
	n, err := f.ReadAt(headerData, 0)
	if err == nil && n == HeaderSize {
		header := &Plus3DosHeader{}
		if err := header.FromBytes(headerData); err == nil {
			if err := header.Validate(); err == nil {
				f.header = header
				f.isHeadered = true
				f.position = HeaderSize
				// The PLUS3DOS header records the exact total file length
				// (header + data); prefer it over the record-rounded size so
				// reads and exports are byte-exact.
				if header.FileLength > 0 {
					f.size = int64(header.FileLength)
				}
			}
		}
	}

	return f, nil
}

// Write implements io.Writer
func (f *File) Write(p []byte) (n int, err error) {
	if f.readOnly {
		return 0, errors.New("file is read-only")
	}

	return f.WriteAt(p, f.position)
}

// WriteAt implements io.WriterAt
func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	if f.readOnly {
		return 0, errors.New("file is read-only")
	}

	// Calculate required blocks
	endPos := off + int64(len(p))
	if endPos > f.size {
		blocksNeeded := (int(endPos) + BlockSize - 1) / BlockSize
		currentBlocks := len(f.blocks)

		if blocksNeeded > currentBlocks {
			// Allocate exactly the shortfall, in whole blocks. Sizing by the byte
			// delta re-rounds on every incremental write and over-allocates.
			extraBlocks := blocksNeeded - currentBlocks
			newBlocks, err := f.disk.fileAlloc.AllocateFileSpace(extraBlocks * BlockSize)
			if err != nil {
				return 0, fmt.Errorf("failed to allocate space: %v", err)
			}
			f.blocks = append(f.blocks, newBlocks...)
		}
		f.size = endPos
	}

	// Write data to blocks
	written := 0
	for written < len(p) {
		blockIdx := int(off+int64(written)) / BlockSize
		if blockIdx >= len(f.blocks) {
			break
		}

		blockOffset := int(off+int64(written)) % BlockSize
		blockRemaining := BlockSize - blockOffset
		writeSize := min(len(p)-written, blockRemaining)

		// Map the allocation block to a physical track/sector. Allocation blocks
		// are numbered from the start of the data area (track 1, the reserved
		// system track being track 0). Each block is two 512-byte sectors.
		block := f.blocks[blockIdx]
		linearSector := block*SectorsPerBlock + blockOffset/BytesPerSector
		track := DirectoryTrack + linearSector/SectorsPerTrack
		sector := linearSector % SectorsPerTrack

		// Sector writes must be full 512-byte sectors; for a partial write,
		// read-modify-write the sector so surrounding bytes are preserved.
		secOff := blockOffset % BytesPerSector
		cur, err := f.disk.GetSectorData(track, sector, 0)
		if err != nil {
			cur = make([]byte, BytesPerSector)
			for i := range cur {
				cur[i] = 0xE5
			}
		}
		nWrite := writeSize
		if secOff+nWrite > BytesPerSector {
			nWrite = BytesPerSector - secOff
		}
		copy(cur[secOff:secOff+nWrite], p[written:written+nWrite])
		if err = f.disk.SetSectorData(track, sector, 0, cur); err != nil {
			return written, err
		}

		written += nWrite
	}

	f.position = off + int64(written)
	return written, nil
}

// Read implements io.Reader
func (f *File) Read(p []byte) (n int, err error) {
	n, err = f.ReadAt(p, f.position)
	f.position += int64(n)
	return
}

// ReadAt implements io.ReaderAt
func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= f.size {
		return 0, io.EOF
	}

	toRead := min(len(p), int(f.size-off))
	read := 0

	for read < toRead {
		blockIdx := int(off+int64(read)) / BlockSize
		if blockIdx >= len(f.blocks) {
			break
		}

		blockOffset := int(off+int64(read)) % BlockSize
		blockRemaining := BlockSize - blockOffset
		readSize := min(toRead-read, blockRemaining)

		// Map the allocation block to a physical track/sector (see WriteAt).
		block := f.blocks[blockIdx]
		linearSector := block*SectorsPerBlock + blockOffset/BytesPerSector
		track := DirectoryTrack + linearSector/SectorsPerTrack
		sector := linearSector % SectorsPerTrack

		data, err := f.disk.GetSectorData(track, sector, 0)
		if err != nil {
			return read, err
		}
		secOff := blockOffset % BytesPerSector
		nRead := readSize
		if secOff+nRead > BytesPerSector {
			nRead = BytesPerSector - secOff
		}
		copy(p[read:read+nRead], data[secOff:secOff+nRead])
		read += nRead
	}

	if read < len(p) {
		err = io.EOF
	}
	return read, err
}

// Seek implements io.Seeker
func (f *File) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = f.position + offset
	case io.SeekEnd:
		abs = f.size + offset
	default:
		return 0, errors.New("invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("negative position")
	}
	f.position = abs
	return abs, nil
}

// Close implements io.Closer
func (f *File) Close() error {
	if f.readOnly {
		return nil
	}

	if f.isHeadered {
		// Update header
		f.header.FileLength = uint32(f.size)
		f.header.UpdateChecksum()
		headerData := f.header.toBytes()
		_, err := f.WriteAt(headerData, 0)
		if err != nil {
			return fmt.Errorf("failed to write header: %v", err)
		}
	}

	// Update directory entry. The CP/M Al field holds the block NUMBERS used by
	// this extent (up to 16 entries), not the count. For 1K blocks the numbers fit
	// in a single byte per slot.
	f.entry.RecordCount = uint8((f.size + 127) / 128) // 128-byte records in this extent
	for i := range f.entry.AllocationBlocks {
		f.entry.AllocationBlocks[i] = 0
	}
	for i, blk := range f.blocks {
		if i >= len(f.entry.AllocationBlocks) {
			break
		}
		f.entry.AllocationBlocks[i] = uint8(blk)
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// splitFilename splits "NAME.EXT" into an 8-char name and 3-char extension,
// upper-cased and space-padded to CP/M form.
func splitFilename(filename string) (name [8]byte, ext [3]byte) {
	for i := range name {
		name[i] = ' '
	}
	for i := range ext {
		ext[i] = ' '
	}
	fn := strings.ToUpper(filename)
	dot := strings.LastIndex(fn, ".")
	base := fn
	var e string
	if dot >= 0 {
		base = fn[:dot]
		e = fn[dot+1:]
	}
	for i := 0; i < len(base) && i < 8; i++ {
		name[i] = base[i]
	}
	for i := 0; i < len(e) && i < 3; i++ {
		ext[i] = e[i]
	}
	return name, ext
}
