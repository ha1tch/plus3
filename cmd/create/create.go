// file: cmd/create/create.go

package create

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ha1tch/plus3/pkg/diskimg"
)

// FormatType specifies the disk format
type FormatType int

const (
	// Format3DOS standard +3DOS format (default)
	Format3DOS FormatType = iota
	// FormatCPCData CPC data-only format
	FormatCPCData
	// FormatCPCSystem CPC system format
	FormatCPCSystem
)

// CreateOptions configures the disk creation
type CreateOptions struct {
	Format    FormatType // Disk format to use
	Label     string     // Optional disk label
	Boot      bool       // Create bootable disk
	Force     bool       // Overwrite existing file
	Quiet     bool       // Suppress non-error output
}

// DefaultCreateOptions returns default options for Create
func DefaultCreateOptions() *CreateOptions {
	return &CreateOptions{
		Format:    Format3DOS,
		Label:     "",
		Boot:      false,
		Force:     false,
		Quiet:     false,
	}
}

// Create creates a new disk image
func Create(outPath string, opts *CreateOptions) error {
	// Validate options
	if opts == nil {
		opts = DefaultCreateOptions()
	}

	// Clean and validate path
	outPath = filepath.Clean(outPath)
	
	// Check if file exists
	if !opts.Force {
		if _, err := os.Stat(outPath); err == nil {
			return fmt.Errorf("file already exists: %s (use force to overwrite)", outPath)
		}
	}

	// Ensure directory exists
	if dir := filepath.Dir(outPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Create new disk image
	disk := diskimg.NewDiskImage()
	if disk == nil {
		return fmt.Errorf("failed to create disk image")
	}

	// Apply format-specific settings
	switch opts.Format {
	case FormatCPCData:
		disk.Header.DiskType = 2 // CPC data-only format
	case FormatCPCSystem:
		disk.Header.DiskType = 1 // CPC system format
	default:
		disk.Header.DiskType = 0 // Standard +3DOS format
	}

	// Set disk label if provided
	if opts.Label != "" {
		if err := setDiskLabel(disk, opts.Label); err != nil {
			return fmt.Errorf("failed to set disk label: %w", err)
		}
	}

	// Initialize disk directory
	if err := disk.InitializeDirectory(); err != nil {
		return fmt.Errorf("failed to initialize directory: %w", err)
	}

	// Set up boot sector if requested
	if opts.Boot {
		if err := setupBootSector(disk); err != nil {
			return fmt.Errorf("failed to set up boot sector: %w", err)
		}
	}

	// Save disk image
	if err := disk.SaveToFile(outPath); err != nil {
		// Clean up partial file on error
		os.Remove(outPath)
		return fmt.Errorf("failed to save disk image: %w", err)
	}

	// Verify the created image
	if err := verifyDiskImage(outPath); err != nil {
		// Clean up invalid file
		os.Remove(outPath)
		return fmt.Errorf("disk image verification failed: %w", err)
	}

	if !opts.Quiet {
		format := "3DOS"
		switch opts.Format {
		case FormatCPCData:
			format = "CPC data"
		case FormatCPCSystem:
			format = "CPC system"
		}
		fmt.Printf("Created %s format disk image: %s\n", format, outPath)
		if opts.Boot {
			fmt.Println("Disk is bootable")
		}
		if opts.Label != "" {
			fmt.Printf("Disk label: %s\n", opts.Label)
		}
	}

	return nil
}

// setDiskLabel sets the disk volume label
func setDiskLabel(disk *diskimg.DiskImage, label string) error {
	if len(label) > 11 { // 8.3 format maximum
		return fmt.Errorf("disk label too long (maximum 11 characters)")
	}

	dir, err := disk.GetDirectory()
	if err != nil {
		return err
	}

	// Create volume label entry
	entry := diskimg.DirectoryEntry{
		Status: 0x00,
	}
	copy(entry.Name[:], label)

	// Set volume label attribute
	attrs := &diskimg.FileAttributes{
		System:   true,
		ReadOnly: true,
	}
	attrs.ApplyToDirectoryEntry(&entry)

	dir.Entries[0] = entry
	return nil
}

// setupBootSector prepares a bootable disk
func setupBootSector(disk *diskimg.DiskImage) error {
	// Get first sector
	sector, err := disk.GetSectorData(0, 0, 0)
	if err != nil {
		return err
	}

	// Clear boot sector
	for i := range sector {
		sector[i] = 0
	}

	// Set disk parameters in boot sector
	sector[0] = 0 // Standard +3DOS format
	sector[1] = 0 // Single sided
	sector[2] = 40 // Tracks per side
	sector[3] = 9 // Sectors per track
	sector[4] = 2 // Sector size (512 = 2^(7+2))
	sector[5] = 1 // Reserved tracks
	sector[6] = 3 // Block size (1K = 2^(7+3))
	sector[7] = 2 // Directory blocks

	// Calculate checksum
	var sum byte
	for i := 0; i < 255; i++ {
		sum += sector[i]
	}
	sector[255] = byte(3 - sum) // Make sum == 3

	// Write boot sector back
	return disk.SetSectorData(0, 0, 0, sector)
}

// verifyDiskImage checks if the created image is valid
func verifyDiskImage(path string) error {
	// Try to load the disk image
	disk, err := diskimg.LoadFromFile(path)
	if err != nil {
		return err
	}

	// Run validation checks
	validator := diskimg.NewDiskCheck(disk, diskimg.ValidationStrict)
	if errors := validator.Validate(); len(errors) > 0 {
		return fmt.Errorf("disk validation errors: %v", errors[0])
	}

	return nil
}