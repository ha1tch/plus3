// file: pkg/diskimg/hostio.go

package diskimg

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ImportOptions configures file import behavior
type ImportOptions struct {
	AddHeader bool   // Add PLUS3DOS header
	FileType  byte   // BASIC/CODE/etc for header
	LoadAddr  uint16 // Load address for CODE files
	Line      uint16 // LINE parameter for BASIC
}

// ImportFile imports a file from the host filesystem into the disk image
func (di *DiskImage) ImportFile(hostPath string, diskPath string, opts *ImportOptions) error {
	// Open source file
	src, err := os.Open(hostPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// Get file size
	info, err := src.Stat()
	if err != nil {
		return err
	}

	// Validate size
	maxSize := 8 * 1024 * 1024 // 8MB max file size
	if info.Size() > int64(maxSize) {
		return errors.New("file too large for +3DOS (max 8MB)")
	}

	// Create destination file
	dst, err := di.OpenFile(diskPath, true)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Add header if requested
	if opts != nil && opts.AddHeader {
		header := NewPlus3DosHeader()
		switch opts.FileType {
		case FileTypeProgram:
			err = header.SetBasicHeader(FileTypeProgram, uint16(info.Size()), opts.Line, uint16(info.Size()))
		case FileTypeCode:
			err = header.SetBasicHeader(FileTypeCode, uint16(info.Size()), opts.LoadAddr, 0)
		default:
			err = errors.New("unsupported file type for header")
		}
		if err != nil {
			return err
		}

		// The PLUS3DOS header's FileLength is the TOTAL on-disk length: the
		// 128-byte header record plus the data. Set it and the checksum before
		// writing, otherwise +3DOS sees a zero-length / invalid header.
		header.FileLength = uint32(HeaderSize) + uint32(info.Size())
		header.UpdateChecksum()

		headerData := header.toBytes()
		_, err = dst.Write(headerData)
		if err != nil {
			return err
		}
	}

	// Copy file data
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	return nil
}

// ImportBasicProgram imports an already-tokenised BASIC program with the
// appropriate PLUS3DOS header. The host file is stored verbatim; use
// ImportBasicText (or the add command's source mode) to tokenise plain-text
// BASIC source instead.
func (di *DiskImage) ImportBasicProgram(hostPath string, line uint16) error {
	// Determine destination filename
	base := filepath.Base(hostPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if len(name) > 8 {
		name = name[:8]
	}
	diskPath := name + ".BAS"

	opts := &ImportOptions{
		AddHeader: true,
		FileType:  FileTypeProgram,
		Line:      line,
	}

	return di.ImportFile(hostPath, diskPath, opts)
}

// ImportBasicText tokenises plain-text BASIC source and imports it as a BASIC
// program with the appropriate PLUS3DOS header.
func (di *DiskImage) ImportBasicText(hostPath string, line uint16) error {
	base := filepath.Base(hostPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if len(name) > 8 {
		name = name[:8]
	}
	diskPath := name + ".BAS"

	data, err := os.ReadFile(hostPath)
	if err != nil {
		return err
	}
	tokenised, err := TokeniseBasic(string(data))
	if err != nil {
		return fmt.Errorf("tokenise BASIC source: %w", err)
	}
	return di.importBasicBytes(diskPath, tokenised, line)
}

// importBasicBytes writes already-tokenised BASIC bytes to the disk with a
// PLUS3DOS BASIC header.
func (di *DiskImage) importBasicBytes(diskPath string, data []byte, line uint16) error {
	dst, err := di.OpenFile(diskPath, true)
	if err != nil {
		return err
	}
	defer dst.Close()

	header := NewPlus3DosHeader()
	if err := header.SetBasicHeader(FileTypeProgram, uint16(len(data)), line, uint16(len(data))); err != nil {
		return err
	}
	// FileLength is the total on-disk length: the 128-byte header plus the data.
	header.FileLength = uint32(HeaderSize) + uint32(len(data))
	header.UpdateChecksum()

	if _, err := dst.Write(header.toBytes()); err != nil {
		return err
	}
	if _, err := dst.Write(data); err != nil {
		return err
	}
	return nil
}

// IsBasicProgram reports whether the named file on the disk carries a PLUS3DOS
// header identifying it as a tokenised BASIC program (file type 0). It returns
// false for headerless files or files of any other type. This is the
// authoritative type signal (the file's own header), used to warn when a BASIC
// program is about to be extracted as raw bytes rather than detokenised.
func (di *DiskImage) IsBasicProgram(diskPath string) bool {
	f, err := di.OpenFile(diskPath, false)
	if err != nil {
		return false
	}
	defer f.Close()
	return f.isHeadered && f.header != nil && f.header.HeaderData[0] == FileTypeProgram
}

// ImportCode imports binary/CODE file with load address
func (di *DiskImage) ImportCode(hostPath string, loadAddr uint16) error {
	// Determine destination filename
	base := filepath.Base(hostPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if len(name) > 8 {
		name = name[:8]
	}
	diskPath := name + ".BIN"

	opts := &ImportOptions{
		AddHeader: true,
		FileType:  FileTypeCode,
		LoadAddr:  loadAddr,
	}

	return di.ImportFile(hostPath, diskPath, opts)
}

// ImportScreen imports a screen$ file (6912 bytes) with standard load address
func (di *DiskImage) ImportScreen(hostPath string) error {
	// Validate file size
	info, err := os.Stat(hostPath)
	if err != nil {
		return err
	}
	if info.Size() != 6912 {
		return errors.New("invalid screen$ file size (must be 6912 bytes)")
	}

	// Determine destination filename
	base := filepath.Base(hostPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if len(name) > 8 {
		name = name[:8]
	}
	diskPath := name + ".SCR"

	opts := &ImportOptions{
		AddHeader: true,
		FileType:  FileTypeCode,
		LoadAddr:  16384, // Standard screen$ address
	}

	return di.ImportFile(hostPath, diskPath, opts)
}

// ImportRaw imports a file without any header or conversion
func (di *DiskImage) ImportRaw(hostPath string) error {
	base := filepath.Base(hostPath)
	if len(base) > 12 { // 8+1+3
		base = base[:12]
	}
	return di.ImportFile(hostPath, base, nil)
}

// ExportFile exports a file from the disk image to the host filesystem
func (di *DiskImage) ExportFile(diskPath, hostPath string, stripHeader bool) error {
	src, err := di.OpenFile(diskPath, false)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(hostPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if stripHeader && src.isHeadered {
		_, err = src.Seek(HeaderSize, io.SeekStart)
		if err != nil {
			return err
		}
	}

	_, err = io.Copy(dst, src)
	return err
}

// ExportScreen exports a screen$ file, validating size and format
func (di *DiskImage) ExportScreen(diskPath, hostPath string) error {
	f, err := di.OpenFile(diskPath, false)
	if err != nil {
		return err
	}
	defer f.Close()

	if !f.isHeadered {
		return errors.New("not a valid screen$ file (no header)")
	}

	fileType, size, loadAddr, _ := f.header.GetBasicHeader()
	if fileType != FileTypeCode || size != 6912 || loadAddr != 16384 {
		return errors.New("not a valid screen$ file")
	}

	return di.ExportFile(diskPath, hostPath, true)
}

// ExtractBasic exports a BASIC program, stripping the header
func (di *DiskImage) ExtractBasic(diskPath, hostPath string) error {
	f, err := di.OpenFile(diskPath, false)
	if err != nil {
		return err
	}
	defer f.Close()

	if !f.isHeadered {
		return errors.New("not a BASIC program (no header)")
	}

	fileType, _, _, _ := f.header.GetBasicHeader()
	if fileType != FileTypeProgram {
		return errors.New("not a BASIC program")
	}

	return di.ExportFile(diskPath, hostPath, true)
}
