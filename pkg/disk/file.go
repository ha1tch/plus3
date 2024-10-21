package disk

import (
    "io"
)

// DiskFile represents a file on the +3DOS disk and implements io.Reader and io.Writer.
type DiskFile struct {
    disk     *Disk      // Reference to the disk
    entry    *FileEntry // Pointer to the file entry in the directory
    position int        // Current position within the file
}

// Read implements io.Reader to read bytes from the file.
func (f *DiskFile) Read(p []byte) (n int, err error) {
    if f.position >= len(f.entry.Sectors) {
        return 0, io.EOF // End of file reached
    }

    sectorIndex := f.entry.Sectors[f.position]
    sector := f.disk.Tracks[sectorIndex/disk.XDPB.DPB.SectorsPerTrack].Sectors[sectorIndex%disk.XDPB.DPB.SectorsPerTrack]
    n = copy(p, sector.Data[:])
    f.position += n
    return n, nil
}

// Write implements io.Writer to write bytes to the file.
func (f *DiskFile) Write(p []byte) (n int, err error) {
    sectorIndex, err := f.disk.AllocateFreeSector()
    if err != nil {
        return 0, err
    }

    f.entry.Sectors = append(f.entry.Sectors, sectorIndex)
    sector := &f.disk.Tracks[sectorIndex/disk.XDPB.DPB.SectorsPerTrack].Sectors[sectorIndex%disk.XDPB.DPB.SectorsPerTrack]
    n = copy(sector.Data[:], p)
    f.position += n
    return n, nil
}

// Close commits the file changes to disk.
func (f *DiskFile) Close() error {
    // Normally we'd update the directory and flush changes.
    return nil
}

// CreateFile creates a new file in the disk's directory.
func (disk *Disk) CreateFile(fileName string) (*DiskFile, error) {
    file := &FileEntry{
        FileName: fileName,
        Size:     0,
        Sectors:  []int{},
    }
    disk.Directory.Entries = append(disk.Directory.Entries, *file)

    return &DiskFile{
        disk:     disk,
        entry:    file,
        position: 0,
    }, nil
}

