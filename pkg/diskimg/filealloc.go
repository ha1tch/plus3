// file: pkg/diskimg/filealloc.go

package diskimg

import (
	"errors"
	"fmt"
)

const (
	BlockSize    = 1024  // +3DOS uses 1K blocks
	MaxBlocks    = 256   // Maximum number of blocks per file
	BlocksPerDir = 2     // Directory takes 2 blocks
	ReservedBlocks = 1   // Boot sector block
)

// FileAllocation handles file space allocation on disk
type FileAllocation struct {
	disk       *DiskImage
	allocation *SectorAllocation
	blockMap   []int      // Maps block numbers to first sector number
	freeBlocks []bool     // Tracks which blocks are free
}

// newFileAllocation creates a new file allocation manager
func newFileAllocation(disk *DiskImage) *FileAllocation {
	sectorsPerBlock := BlockSize / BytesPerSector
	totalBlocks := (disk.Header.TracksNum * disk.Header.SidesNum * uint8(SectorsPerTrack)) / uint8(sectorsPerBlock)

	fa := &FileAllocation{
		disk:       disk,
		allocation: disk.allocation,
		blockMap:   make([]int, totalBlocks),
		freeBlocks: make([]bool, totalBlocks),
	}

	// Initialize block map
	for i := range fa.blockMap {
		fa.blockMap[i] = i * sectorsPerBlock
		fa.freeBlocks[i] = true
	}

	// Mark system blocks as allocated
	for i := 0; i < ReservedBlocks+BlocksPerDir; i++ {
		fa.freeBlocks[i] = false
	}

	return fa
}

// AllocateFileSpace allocates blocks for a file
func (fa *FileAllocation) AllocateFileSpace(size int) ([]int, error) {
	blocksNeeded := (size + BlockSize - 1) / BlockSize
	if blocksNeeded > MaxBlocks {
		return nil, fmt.Errorf("file size exceeds maximum (%d blocks needed, max is %d)", 
			blocksNeeded, MaxBlocks)
	}

	blocks := make([]int, 0, blocksNeeded)
	sectorsPerBlock := BlockSize / BytesPerSector

	// Try to find contiguous blocks first
	startBlock := fa.findContiguousBlocks(blocksNeeded)
	if startBlock >= 0 {
		// Allocate contiguous blocks
		for i := 0; i < blocksNeeded; i++ {
			block := startBlock + i
			fa.freeBlocks[block] = false
			
			// Allocate sectors for this block
			firstSector := fa.blockMap[block]
			err := fa.allocation.AllocateSectors(firstSector, sectorsPerBlock)
			if err != nil {
				fa.FreeBlocks(blocks) // Rollback on error
				return nil, err
			}
			blocks = append(blocks, block)
		}
		return blocks, nil
	}

	// Fall back to fragmented allocation
	for i := 0; i < blocksNeeded; i++ {
		block := fa.findFreeBlock()
		if block < 0 {
			fa.FreeBlocks(blocks) // Rollback
			return nil, errors.New("no free blocks available")
		}

		fa.freeBlocks[block] = false
		firstSector := fa.blockMap[block]
		err := fa.allocation.AllocateSectors(firstSector, sectorsPerBlock)
		if err != nil {
			fa.FreeBlocks(blocks) // Rollback
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// FreeBlocks releases allocated blocks
func (fa *FileAllocation) FreeBlocks(blocks []int) error {
	sectorsPerBlock := BlockSize / BytesPerSector
	
	for _, block := range blocks {
		if block >= len(fa.blockMap) {
			return fmt.Errorf("invalid block number: %d", block)
		}
		
		fa.freeBlocks[block] = true
		firstSector := fa.blockMap[block]
		err := fa.allocation.FreeSectors(firstSector, sectorsPerBlock)
		if err != nil {
			return err
		}
	}
	return nil
}

// findContiguousBlocks looks for a sequence of free blocks
func (fa *FileAllocation) findContiguousBlocks(count int) int {
	consecutive := 0
	startBlock := -1

	for i, free := range fa.freeBlocks {
		if free {
			if consecutive == 0 {
				startBlock = i
			}
			consecutive++
			if consecutive == count {
				return startBlock
			}
		} else {
			consecutive = 0
			startBlock = -1
		}
	}
	return -1
}

// findFreeBlock finds a single free block
func (fa *FileAllocation) findFreeBlock() int {
	for i, free := range fa.freeBlocks {
		if free {
			return i
		}
	}
	return -1
}

// GetFreeBlocks returns number of free blocks
func (fa *FileAllocation) GetFreeBlocks() int {
	count := 0
	for _, free := range fa.freeBlocks {
		if free {
			count++
		}
	}
	return count
}

// DefragmentFile attempts to make file blocks contiguous
func (fa *FileAllocation) DefragmentFile(oldBlocks []int) ([]int, error) {
	if len(oldBlocks) == 0 {
		return nil, nil
	}

	// Calculate total size
	totalSize := len(oldBlocks) * BlockSize

	// Try to find contiguous space
	newBlocks, err := fa.AllocateFileSpace(totalSize)
	if err != nil {
		return nil, err
	}

	// Copy blocks to new location
	sectorsPerBlock := BlockSize / BytesPerSector
	
	for i, oldBlock := range oldBlocks {
		newBlock := newBlocks[i]
		
		// Copy each sector in the block
		for s := 0; s < sectorsPerBlock; s++ {
			oldSector := fa.blockMap[oldBlock] + s
			newSector := fa.blockMap[newBlock] + s
			
			// Read old sector
			data, err := fa.disk.GetSectorData(
				oldSector/SectorsPerTrack,
				oldSector%SectorsPerTrack,
				0)
			if err != nil {
				fa.FreeBlocks(newBlocks)  // Rollback
				return nil, err
			}

			// Write to new sector
			err = fa.disk.SetSectorData(
				newSector/SectorsPerTrack,
				newSector%SectorsPerTrack,
				0,
				data)
			if err != nil {
				fa.FreeBlocks(newBlocks)  // Rollback
				return nil, err
			}
		}
	}

	// Free old blocks
	fa.FreeBlocks(oldBlocks)

	return newBlocks, nil
}