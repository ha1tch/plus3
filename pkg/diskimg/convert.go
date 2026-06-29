// file: pkg/diskimg/convert.go

package diskimg

import (
	"errors"
	"io"

	"github.com/ha1tch/zentools/pkg/tap"
)

// ConvertTAPtoDisk converts a single-file TAP image (a header block followed by
// its data block) into a +3DOS file written to diskPath. TAP parsing and
// checksum verification are delegated to zentools/pkg/tap, the verified
// interchange implementation.
func (di *DiskImage) ConvertTAPtoDisk(r io.Reader, diskPath string) error {
	image, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	blocks, err := tap.Decode(image)
	if err != nil {
		return err
	}

	// Find the header block and the data block that follows it.
	var header *tap.Block
	var data *tap.Block
	for i := range blocks {
		if blocks[i].IsHeader {
			header = &blocks[i]
			if i+1 < len(blocks) {
				data = &blocks[i+1]
			}
			break
		}
	}
	if header == nil {
		return errors.New("no TAP header block found")
	}
	if data == nil {
		return errors.New("TAP header block has no following data block")
	}
	if !header.ChecksumOK {
		return errors.New("TAP header block checksum mismatch")
	}
	if !data.ChecksumOK {
		return errors.New("TAP data block checksum mismatch")
	}

	// Build the +3DOS header from the TAP header fields.
	plus3Header := NewPlus3DosHeader()
	switch header.Type {
	case tap.TypeProgram:
		err = plus3Header.SetBasicHeader(FileTypeProgram, header.DataLength, header.Param1, header.Param2)
	case tap.TypeCode:
		err = plus3Header.SetBasicHeader(FileTypeCode, header.DataLength, header.Param1, 0)
	default:
		return errors.New("unsupported TAP file type")
	}
	if err != nil {
		return err
	}
	// Record the total file length (header + data) so the file is recognised as
	// headered when reopened, then compute the header checksum. Both are
	// required for OpenFile to accept the header, matching ImportFile.
	plus3Header.FileLength = uint32(HeaderSize) + uint32(len(data.Data))
	plus3Header.UpdateChecksum()

	f, err := di.OpenFile(diskPath, true)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(plus3Header.toBytes()); err != nil {
		return err
	}
	if _, err := f.Write(data.Data); err != nil {
		return err
	}
	return nil
}

// ConvertDiskToTAP converts a headered +3DOS file at diskPath into a TAP image
// (header block plus data block) written to w. TAP encoding, including the
// header layout and both block checksums, is delegated to zentools/pkg/tap.
func (di *DiskImage) ConvertDiskToTAP(diskPath string, w io.Writer) error {
	f, err := di.OpenFile(diskPath, false)
	if err != nil {
		return err
	}
	defer f.Close()

	if !f.isHeadered {
		return errors.New("file has no header")
	}

	fileType, length, param1, param2 := f.header.GetBasicHeader()

	// Read the file data, skipping the 128-byte +3DOS header.
	data := make([]byte, length)
	if _, err := f.Seek(HeaderSize, io.SeekStart); err != nil {
		return err
	}
	if _, err := io.ReadFull(f, data); err != nil {
		return err
	}

	name := trimName(f.entry.Name[:])

	var image []byte
	switch fileType {
	case FileTypeProgram:
		// param1 is the autostart LINE, param2 the program length.
		image = tap.EncodeProgram(name, data, param1)
	case FileTypeCode:
		// param1 is the load address.
		image = tap.EncodeCode(name, data, param1)
	default:
		return errors.New("unsupported +3DOS file type for TAP conversion")
	}

	_ = param2 // program length is recomputed by EncodeProgram from the data
	_, err = w.Write(image)
	return err
}

// trimName converts a fixed-width, space-padded +3DOS name field into a string
// with trailing spaces removed.
func trimName(name []byte) string {
	end := len(name)
	for end > 0 && name[end-1] == ' ' {
		end--
	}
	return string(name[:end])
}
