// file: cmd/extract/extract.go

package extract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ha1tch/plus3/pkg/diskimg"
)

// ExtractOptions configures the file extraction operation
type ExtractOptions struct {
	StripHeader bool   // Remove PLUS3DOS header if present
	OutputDir   string // Directory to extract files to
	Overwrite   bool   // Allow overwriting existing files
	Quiet       bool   // Suppress non-error output
	PreserveCAS bool   // Preserve Sinclair BASIC encoding
}

// DefaultExtractOptions returns default options for Extract
func DefaultExtractOptions() *ExtractOptions {
	return &ExtractOptions{
		StripHeader: false,
		OutputDir:   "",
		Overwrite:   false,
		Quiet:       false,
		PreserveCAS: false,
	}
}

// Extract copies a file from the disk image to the host filesystem
func Extract(diskPath string, filename string, opts *ExtractOptions) error {
	// Validate options
	if opts == nil {
		opts = DefaultExtractOptions()
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

	// Validate/create output directory
	if opts.OutputDir != "" {
		if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Determine output path
	outPath := filename
	if opts.OutputDir != "" {
		outPath = filepath.Join(opts.OutputDir, filename)
	}

	// Check if output file exists
	if !opts.Overwrite {
		if _, err := os.Stat(outPath); err == nil {
			return fmt.Errorf("output file already exists: %s (use overwrite to replace)", outPath)
		}
	}

	// Open disk image
	disk, err := diskimg.LoadFromFile(diskPath)
	if err != nil {
		return fmt.Errorf("failed to open disk: %w", err)
	}

	// Verify file exists on disk
	dir, err := disk.GetDirectory()
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	entry, _, err := dir.FindFile(filename)
	if err != nil {
		return fmt.Errorf("file not found: %s", filename)
	}

	// Extract based on file type and extension
	ext := strings.ToLower(filepath.Ext(filename))
	var extractErr error

	switch {
	case ext == ".bas" && !opts.PreserveCAS:
		extractErr = disk.ExtractBasic(filename, outPath)
	case ext == ".scr":
		extractErr = disk.ExportScreen(filename, outPath)
	case ext == ".bin" && entry.FileType == diskimg.FileTypeCode:
		extractErr = disk.ExportFile(filename, outPath, opts.StripHeader)
	default:
		// Generic file export
		extractErr = disk.ExportFile(filename, outPath, opts.StripHeader)
	}

	if extractErr != nil {
		// Clean up partial output file on error
		os.Remove(outPath)
		return fmt.Errorf("failed to extract file: %w", extractErr)
	}

	if !opts.Quiet {
		fmt.Printf("Extracted %s to %s\n", filename, outPath)
	}

	return nil
}

// ExtractAll extracts all files from the disk image
func ExtractAll(diskPath string, opts *ExtractOptions) error {
	if opts == nil {
		opts = DefaultExtractOptions()
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

	// Extract each file
	extractedCount := 0
	for _, entry := range dir.Entries {
		if !entry.IsDeleted() && !entry.IsUnused() {
			filename := entry.GetFilename()
			if err := Extract(diskPath, filename, opts); err != nil {
				return fmt.Errorf("failed to extract %s: %w", filename, err)
			}
			extractedCount++
		}
	}

	if !opts.Quiet {
		fmt.Printf("Extracted %d files from disk image\n", extractedCount)
	}

	return nil
}