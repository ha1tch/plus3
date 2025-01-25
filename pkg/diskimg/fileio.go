// file: pkg/diskimg/fileio.go

package diskimg

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// File represents an open file on the disk image
type File struct {
	disk        *DiskImage
	entry       *DirectoryEntry
	header      *Plus3DosHeader
	blocks      []int
	position    int64
	size        int64
	readOnly    bool
	isHeadered  bool
}

// OpenFile opens or creates a file on the disk image
func (di *DiskImage) OpenFile(filename string, createNew bool) (*File, error) {
	entry, _, err := di.directory.FindFile(filename)
	if err != nil && !createNew {
		return nil, err
	}

	if err != nil && createNew {
		// Create new file
		entry, err = di.directory.AddFile(filename)
		if err != nil {
			return nil, err
		}
	}

	// Create file struct
	f := &File{
		disk:     di,
		entry:    entry,
		position: 0,
		readOnly: false,
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
			// Allocate additional blocks
			newBlocks, err := f.disk.fileAlloc.AllocateFileSpace(int(endPos - f.size))
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

		// Write to sectors in block
		block := f.blocks[blockIdx]
		firstSector := f.disk.fileAlloc.blockMap[block]
		sectorOffset := blockOffset / BytesPerSector
		sectorNum := firstSector + sectorOffset

		track := sectorNum / SectorsPerTrack
		sector := sectorNum % SectorsPerTrack

		err = f.disk.SetSectorData(track, sector, 0, p[written:written+writeSize])
		if err != nil {
			return written, err
		}

		written += writeSize
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

		// Read from sectors in block
		block := f.blocks[blockIdx]
		firstSector := f.disk.fileAlloc.blockMap[block]
		sectorOffset := blockOffset / BytesPerSector
		sectorNum := firstSector + sectorOffset

		track := sectorNum / SectorsPerTrack
		sector := sectorNum % SectorsPerTrack

		data, err := f.disk.GetSectorData(track, sector, 0)
		if err != nil {
			return read, err
		}

		copy(p[read:read+readSize], data)
		read += readSize
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

	// Update directory entry
	f.entry.LogicalSize = uint16((f.size + 127) / 128) // Size in 128-byte records
	binary.LittleEndian.PutUint16(f.entry.AllocationBlocks[:], uint16(len(f.blocks)))
	f.disk.directory.modified = true

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}