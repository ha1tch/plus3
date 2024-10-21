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

// DPB represents the Disk Parameter Block.
type DPB struct {
    SectorsPerTrack int
    BlockSize       int
    DirectorySize   int
}

// XDPB is the Extended Disk Parameter Block containing additional disk info.
type XDPB struct {
    DPB           DPB
    DiskType      byte
    TracksPerSide int
    Sides         int
    SectorSize    int
}

// Disk represents the entire +3DOS disk, including its structure and directory.
type Disk struct {
    Tracks      []Track
    Directory   Directory
    XDPB        XDPB
    FreeSectors []int
}

// NewDisk initializes a new +3DOS disk with a given format.
func NewDisk() (*Disk, error) {
    disk := &Disk{
        XDPB: XDPB{
            DiskType:      0,  // Default type (Spectrum +3 format)
            TracksPerSide: 40, // Example: 40 tracks per side
            Sides:         1,  // Single-sided disk
            SectorSize:    512,
        },
    }

    // Initialize tracks and sectors
    for i := 0; i < disk.XDPB.TracksPerSide*disk.XDPB.Sides; i++ {
        track := Track{
            Sectors: make([]Sector, disk.XDPB.DPB.SectorsPerTrack),
        }
        disk.Tracks = append(disk.Tracks, track)
    }

    // Initialize free sectors
    totalSectors := len(disk.Tracks) * disk.XDPB.DPB.SectorsPerTrack
    for i := 0; i < totalSectors; i++ {
        disk.FreeSectors = append(disk.FreeSectors, i)
    }

    return disk, nil
}

// AllocateFreeSector finds and allocates a free sector on the disk.
func (disk *Disk) AllocateFreeSector() (int, error) {
    if len(disk.FreeSectors) == 0 {
        return -1, fmt.Errorf("no free sectors available")
    }
    sector := disk.FreeSectors[0]
    disk.FreeSectors = disk.FreeSectors[1:]
    return sector, nil
}

// FreeSector releases a sector back to the free pool.
func (disk *Disk) FreeSector(sector int) {
    disk.FreeSectors = append(disk.FreeSectors, sector)
}

