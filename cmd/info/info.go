// file: cmd/info/info.go

package info

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ha1tch/plus3/pkg/diskimg"
)

// DiskInfo represents disk information in a structured format
type DiskInfo struct {
	Path        string    `json:"path"`
	Format      string    `json:"format"`
	Files       int       `json:"files"`
	UsedSpace   int64     `json:"used_space"`
	FreeSpace   int64     `json:"free_space"`
	TotalSpace  int64     `json:"total_space"`
	Modified    time.Time `json:"modified_time,omitempty"`
	Validation  []string  `json:"validation_issues,omitempty"`
}

// InfoOptions configures the information display
type InfoOptions struct {
	JSON         bool // Output in JSON format
	Verbose      bool // Show additional details
	Validate     bool // Perform disk validation
	Quiet        bool // Suppress non-error output
	ShowDeleted  bool // Include information about deleted files
}

// DefaultInfoOptions returns default options for Info
func DefaultInfoOptions() *InfoOptions {
	return &InfoOptions{
		JSON:        false,
		Verbose:     false,
		Validate:    true,
		Quiet:       false,
		ShowDeleted: false,
	}
}

// Info displays information about a disk image
func Info(diskPath string, opts *InfoOptions) error {
	// Validate options
	if opts == nil {
		opts = DefaultInfoOptions()
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

	// Get disk information
	info := &DiskInfo{
		Path:       diskPath,
		Format:     "+3DOS",
		TotalSpace: int64(diskimg.TracksPerSide * diskimg.SectorsPerTrack * diskimg.BytesPerSector),
	}

	// Get directory information
	dir, err := disk.GetDirectory()
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Calculate file and space information
	for _, entry := range dir.Entries {
		if !entry.IsDeleted() && !entry.IsUnused() {
			info.Files++
			info.UsedSpace += int64(entry.LogicalSize) * 128 // Convert records to bytes
		}
	}

	info.FreeSpace = info.TotalSpace - info.UsedSpace

	// Get file modification time
	if stat, err := os.Stat(diskPath); err == nil {
		info.Modified = stat.ModTime()
	}

	// Perform validation if requested
	if opts.Validate {
		validator := diskimg.NewDiskCheck(disk, diskimg.ValidationStrict)
		if errors := validator.Validate(); len(errors) > 0 {
			for _, err := range errors {
				info.Validation = append(info.Validation, err.Error())
			}
		}
	}

	// Output information
	if opts.JSON {
		return outputJSON(info)
	}
	return outputText(info, opts)
}

// outputJSON writes disk information in JSON format
func outputJSON(info *DiskInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}

// outputText writes disk information in human-readable format
func outputText(info *DiskInfo, opts *InfoOptions) error {
	if opts.Quiet && len(info.Validation) == 0 {
		return nil
	}

	fmt.Printf("Disk Image: %s\n\n", info.Path)
	fmt.Printf("Format:     %s\n", info.Format)
	fmt.Printf("Files:      %d\n", info.Files)
	fmt.Printf("Used:       %dK\n", info.UsedSpace/1024)
	fmt.Printf("Free:       %dK\n", info.FreeSpace/1024)
	fmt.Printf("Total:      %dK\n", info.TotalSpace/1024)

	if !info.Modified.IsZero() {
		fmt.Printf("Modified:   %s\n", info.Modified.Format(time.RFC1123))
	}

	if opts.Verbose {
		fmt.Printf("\nDisk Parameters:\n")
		fmt.Printf("Tracks:     %d\n", diskimg.TracksPerSide)
		fmt.Printf("Sectors:    %d per track\n", diskimg.SectorsPerTrack)
		fmt.Printf("Sides:      %d\n", diskimg.SidesPerDisk)
		fmt.Printf("Sector Size: %d bytes\n", diskimg.BytesPerSector)
	}

	if len(info.Validation) > 0 {
		fmt.Printf("\nWarnings:\n")
		for _, warning := range info.Validation {
			fmt.Printf("- %s\n", warning)
		}
	}

	return nil
}