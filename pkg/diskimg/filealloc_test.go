// file: pkg/diskimg/filealloc_test.go

package diskimg

import (
	"testing"
)

func setupTestFileAllocation(t *testing.T) (*DiskImage, *FileAllocation) {
	disk := NewDiskImage()
	if disk == nil {
		t.Fatal("Failed to create test disk image")
	}
	
	fa := newFileAllocation(disk)
	if fa == nil {
		t.Fatal("Failed to create file allocation")
	}
	
	return disk, fa
}

func TestNewFileAllocation(t *testing.T) {
	disk, fa := setupTestFileAllocation(t)

	// Check initialization
	expectedBlocks := (int(disk.Header.TracksNum) * int(disk.Header.SidesNum) * SectorsPerTrack) / (BlockSize / BytesPerSector)
	if len(fa.blockMap) != expectedBlocks {
		t.Errorf("Wrong block count: got %d, want %d", len(fa.blockMap), expectedBlocks)
	}

	// Check system blocks are reserved
	for i := 0; i < ReservedBlocks+BlocksPerDir; i++ {
		if fa.freeBlocks[i] {
			t.Errorf("System block %d should be marked as allocated", i)
		}
	}

	// Check user blocks are free
	for i := ReservedBlocks + BlocksPerDir; i < len(fa.freeBlocks); i++ {
		if !fa.freeBlocks[i] {
			t.Errorf("User block %d should be marked as free", i)
		}
	}
}

func TestAllocateFileSpace(t *testing.T) {
	_, fa := setupTestFileAllocation(t)

	// Test allocating a small file
	blocks, err := fa.AllocateFileSpace(512)
	if err != nil {
		t.Fatalf("Failed to allocate small file: %v", err)
	}
	if len(blocks) != 1 {
		t.Errorf("Wrong number of blocks allocated: got %d, want 1", len(blocks))
	}

	// Test allocating a larger file
	blocks, err = fa.AllocateFileSpace(BlockSize * 3)
	if err != nil {
		t.Fatalf("Failed to allocate larger file: %v", err)
	}
	if len(blocks) != 3 {
		t.Errorf("Wrong number of blocks allocated: got %d, want 3", len(blocks))
	}

	// Test max file size limit
	_, err = fa.AllocateFileSpace(BlockSize * (MaxBlocks + 1))
	if err == nil {
		t.Error("Should fail when exceeding max file size")
	}
}

func TestFreeBlocks(t *testing.T) {
	_, fa := setupTestFileAllocation(t)

	// Allocate some blocks
	blocks, err := fa.AllocateFileSpace(BlockSize * 2)
	if err != nil {
		t.Fatalf("Failed to allocate test blocks: %v", err)
	}

	// Free the blocks
	err = fa.FreeBlocks(blocks)
	if err != nil {
		t.Fatalf("Failed to free blocks: %v", err)
	}

	// Verify blocks are marked as free
	for _, block := range blocks {
		if !fa.freeBlocks[block] {
			t.Errorf("Block %d should be marked as free", block)
		}
	}

	// Try freeing invalid block
	err = fa.FreeBlocks([]int{len(fa.blockMap)})
	if err == nil {
		t.Error("Should fail when freeing invalid block")
	}
}

func TestContiguousAllocation(t *testing.T) {
	_, fa := setupTestFileAllocation(t)

	// Allocate blocks until fragmented
	var allocatedBlocks [][]int
	for i := 0; i < 5; i++ {
		blocks, err := fa.AllocateFileSpace(BlockSize)
		if err != nil {
			t.Fatalf("Failed to allocate block %d: %v", i, err)
		}
		allocatedBlocks = append(allocatedBlocks, blocks)
	}

	// Free alternate blocks to create fragmentation
	for i := 0; i < len(allocatedBlocks); i += 2 {
		err := fa.FreeBlocks(allocatedBlocks[i])
		if err != nil {
			t.Fatalf("Failed to free blocks: %v", err)
		}
	}

	// Try to allocate contiguous blocks
	blocks, err := fa.AllocateFileSpace(BlockSize * 2)
	if err != nil {
		t.Fatalf("Failed to allocate contiguous blocks: %v", err)
	}

	// Verify blocks are contiguous
	for i := 1; i < len(blocks); i++ {
		if blocks[i] != blocks[i-1]+1 {
			t.Error("Allocated blocks are not contiguous")
		}
	}
}

func TestDefragmentFile(t *testing.T) {
	_, fa := setupTestFileAllocation(t)

	// Create a fragmented file
	fragmented, err := fa.AllocateFileSpace(BlockSize * 3)
	if err != nil {
		t.Fatalf("Failed to allocate initial blocks: %v", err)
	}

	// Make it fragmented by allocating blocks between
	spacer, err := fa.AllocateFileSpace(BlockSize)
	if err != nil {
		t.Fatalf("Failed to allocate spacer block: %v", err)
	}

	// Defragment the file
	defragged, err := fa.DefragmentFile(fragmented)
	if err != nil {
		t.Fatalf("Failed to defragment file: %v", err)
	}

	// Verify defragmented blocks are contiguous
	for i := 1; i < len(defragged); i++ {
		if defragged[i] != defragged[i-1]+1 {
			t.Error("Defragmented blocks are not contiguous")
		}
	}

	// Original blocks should be freed
	for _, block := range fragmented {
		if !fa.freeBlocks[block] {
			t.Errorf("Original block %d not freed after defrag", block)
		}
	}

	// Spacer blocks should still be allocated
	for _, block := range spacer {
		if fa.freeBlocks[block] {
			t.Errorf("Spacer block %d incorrectly freed", block)
		}
	}
}

func TestGetFreeBlocks(t *testing.T) {
	_, fa := setupTestFileAllocation(t)
	
	initialFree := fa.GetFreeBlocks()
	
	// Allocate some blocks
	blocks, err := fa.AllocateFileSpace(BlockSize * 2)
	if err != nil {
		t.Fatalf("Failed to allocate test blocks: %v", err)
	}

	newFree := fa.GetFreeBlocks()
	if newFree != initialFree-2 {
		t.Errorf("Wrong free block count after allocation: got %d, want %d", 
			newFree, initialFree-2)
	}

	// Free the blocks
	err = fa.FreeBlocks(blocks)
	if err != nil {
		t.Fatalf("Failed to free blocks: %v", err)
	}

	finalFree := fa.GetFreeBlocks()
	if finalFree != initialFree {
		t.Errorf("Wrong free block count after freeing: got %d, want %d",
			finalFree, initialFree)
	}
}