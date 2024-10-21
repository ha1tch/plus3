package disk

import (
	"fmt"
)

// Sector represents a 512-byte sector on the disk.
type Sector struct {
	Data [512]byte
}

// Track represents a list of sectors on the disk.
type Track struct {
	Sectors []Sector
}

// FileEntry represents an individual file in the directory.
type FileEntry struct {
	FileName   string
	FileType   string
	Size       int
	Attributes byte
	Sectors    []int
}

// Directory stores the list of files on the disk.
type Directory struct {
	Entries []FileEntry
}

// DPB represents the Disk Parameter Block (default values for a typical ZX Spectrum +3 disk).
type DPB struct {
	SectorsPerTrack int // Number of sectors per track
	BlockSize       int // Block size (in bytes)
	DirectorySize   int // Number of entries in the directory
}

// XDPB is the Extended Disk Parameter Block containing additional disk info.
type XDPB struct {
	DPB           DPB
	DiskType      byte // Disk type (default to +3DOS)
	TracksPerSide int  // Number of tracks per side
	Sides         int  // Number of sides (1 for single-sided)
	SectorSize    int  // Size of each sector in bytes (512 bytes)
}

// Disk represents the entire +3DOS disk, including its structure and directory.
type Disk struct {
	Tracks      []Track    // The tracks containing sectors
	Directory   Directory  // The directory that stores files
	XDPB        XDPB       // Extended Disk Parameter Block for the disk
	FreeSectors []int      // List of free sector indices
}

// NewDisk initializes a new +3DOS disk with default settings for a typical ZX Spectrum +3 disk.
func NewDisk() (*Disk, error) {
	// Initialize disk parameters (40 tracks per side, 9 sectors per track, single-sided, 512-byte sectors)
	disk := &Disk{
		XDPB: XDPB{
			DPB: DPB{
				SectorsPerTrack: 9,    // 9 sectors per track
				BlockSize:       512,  // Block size in bytes
				DirectorySize:   64,   // Maximum 64 directory entries
			},
			DiskType:      0,    // Default to +3DOS format
			TracksPerSide: 40,   // 40 tracks per side
			Sides:         1,    // Single-sided disk
			SectorSize:    512,  // Each sector is 512 bytes
		},
	}

	// Initialize the tracks and sectors for the disk
	for trackIndex := 0; trackIndex < disk.XDPB.TracksPerSide*disk.XDPB.Sides; trackIndex++ {
		track := Track{
			Sectors: make([]Sector, disk.XDPB.DPB.SectorsPerTrack),
		}
		disk.Tracks = append(disk.Tracks, track)
	}

	// Initialize the empty directory
	disk.Directory = Directory{
		Entries: make([]FileEntry, 0, disk.XDPB.DPB.DirectorySize), // Empty but can hold up to 64 files
	}

	// Mark all sectors as free (for sector allocation)
	totalSectors := len(disk.Tracks) * disk.XDPB.DPB.SectorsPerTrack
	for sectorIndex := 0; sectorIndex < totalSectors; sectorIndex++ {
		disk.FreeSectors = append(disk.FreeSectors, sectorIndex) // All sectors are free initially
	}

	return disk, nil
}

// Debug print function to show disk details
func (disk *Disk) PrintDetails() {
	fmt.Printf("Disk Type: %d\n", disk.XDPB.DiskType)
	fmt.Printf("Tracks Per Side: %d\n", disk.XDPB.TracksPerSide)
	fmt.Printf("Sides: %d\n", disk.XDPB.Sides)
	fmt.Printf("Sector Size: %d bytes\n", disk.XDPB.SectorSize)
	fmt.Printf("Total Free Sectors: %d\n", len(disk.FreeSectors))
	fmt.Printf("Directory Size: %d entries\n", len(disk.Directory.Entries))
}

