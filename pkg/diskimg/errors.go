// file: pkg/diskimg/errors.go

package diskimg

import "errors"

var (
	ErrInvalidTrack          = errors.New("invalid track number")
	ErrInvalidSide          = errors.New("invalid side number")
	ErrInvalidSector        = errors.New("invalid sector number")
	ErrInvalidSectorSize    = errors.New("invalid sector size")
	ErrInvalidSectorCount   = errors.New("invalid sectors per track")
	ErrInvalidTrackNum      = errors.New("track number mismatch")
	ErrInvalidSectorID      = errors.New("invalid sector ID")
	ErrInvalidTrackSignature = errors.New("invalid track signature")
	ErrReadOnly             = errors.New("disk or file is read-only")
	ErrFileNotFound         = errors.New("file not found")
	ErrDirectoryFull        = errors.New("directory is full")
	ErrDiskFull            = errors.New("disk is full")
	ErrInvalidFilename      = errors.New("invalid filename")
	ErrFileExists          = errors.New("file already exists")
	ErrInvalidHeader       = errors.New("invalid file header")
	ErrInvalidChecksum     = errors.New("invalid checksum")
)