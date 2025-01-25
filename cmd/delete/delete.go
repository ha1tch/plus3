// file: cmd/delete/delete.go

package delete

import (
	"fmt"
	"os"
	"strings"

	"github.com/ha1tch/plus3/pkg/diskimg"
)

// DeleteOptions configures the deletion operation
type DeleteOptions struct {
	Force     bool // Skip confirmation
	Quiet     bool // Suppress non-error output
	NoRecycle bool // Don't preserve deleted file info
}

// DefaultDeleteOptions returns default options for Delete
func DefaultDeleteOptions() *DeleteOptions {
	return &DeleteOptions{
		Force:     false,
		Quiet:     false,
		NoRecycle: false,
	}
}

// Delete removes a file from the disk image
func Delete(diskPath string, filename string, opts *DeleteOptions) error {
	// Validate options
	if opts == nil {
		opts = DefaultDeleteOptions()
	}

	// Normalize filename
	filename = strings.ToUpper(strings.TrimSpace(filename))
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Validate disk exists
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		return fmt.Errorf("disk image does not exist: %w", err)
	}

	// Open disk image
	disk, err := diskimg.LoadFromFile(diskPath)
	if err != nil {
		return fmt.Errorf("failed to open disk: %w", err)
	}

	// Get directory
	dir, err := disk.GetDirectory()
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Verify file exists
	entry, idx, err := dir.FindFile(filename)
	if err != nil {
		return fmt.Errorf("file not found: %s", filename)
	}

	// Verify file is not read-only unless forced
	attrs := &diskimg.FileAttributes{}
	attrs.ReadFromDirectoryEntry(entry)
	if attrs.ReadOnly && !opts.Force {
		return fmt.Errorf("file is read-only: %s (use force to delete)", filename)
	}

	// Confirm deletion unless forced
	if !opts.Force {
		fmt.Printf("Delete %s? (y/N) ", filename)
		var response string
		fmt.Scanln(&response)
		if !strings.HasPrefix(strings.ToLower(response), "y") {
			if !opts.Quiet {
				fmt.Println("Deletion cancelled")
			}
			return nil
		}
	}

	// Perform deletion
	var deleteErr error
	if opts.NoRecycle {
		// Clear directory entry completely
		dir.Entries[idx] = diskimg.DirectoryEntry{}
	} else {
		// Mark as deleted but preserve info
		deleteErr = dir.DeleteFile(filename)
	}

	if deleteErr != nil {
		return fmt.Errorf("failed to delete file: %w", deleteErr)
	}

	// Save disk changes
	if err := disk.SaveToFile(diskPath); err != nil {
		return fmt.Errorf("failed to save disk: %w", err)
	}

	if !opts.Quiet {
		fmt.Printf("Deleted %s\n", filename)
	}

	return nil
}