package diskimg

import (
	"fmt"
	"io"

	zbasic "github.com/ha1tch/zentools/pkg/basic"
)

// ReadBasicText reads a BASIC program file from the disk and returns its
// detokenised text. The file must have a PLUS3DOS header identifying it as a
// BASIC program (file type 0); the 128-byte header is skipped before decoding.
func (di *DiskImage) ReadBasicText(diskPath string) (string, error) {
	f, err := di.OpenFile(diskPath, false)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if !f.isHeadered {
		return "", fmt.Errorf("%s has no PLUS3DOS header; not a BASIC program", diskPath)
	}
	if ftype, _, _, _ := f.header.GetBasicHeader(); ftype != FileTypeProgram {
		return "", fmt.Errorf("%s is not a BASIC program (file type %d)", diskPath, ftype)
	}

	if _, err := f.Seek(HeaderSize, io.SeekStart); err != nil {
		return "", err
	}
	body, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return DetokeniseBasic(body)
}

// DetokeniseBasic converts a tokenised Sinclair BASIC program (the raw program
// bytes, with no PLUS3DOS header) into readable text. It is the inverse of
// TokeniseBasic. Detokenisation is provided by zentools/pkg/basic, the verified
// interchange tokeniser; it handles keywords, numbers, strings, and the
// statement separator.
func DetokeniseBasic(prog []byte) (string, error) {
	return zbasic.Detokenise(prog)
}
