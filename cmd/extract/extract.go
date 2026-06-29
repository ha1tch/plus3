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
	Basic       bool   // Detokenise a BASIC program to readable text
}

// DefaultExtractOptions returns default options for Extract
func DefaultExtractOptions() *ExtractOptions {
	return &ExtractOptions{
		StripHeader: false,
		OutputDir:   "",
		Overwrite:   false,
		Quiet:       false,
		PreserveCAS: false,
		Basic:       false,
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

	found := false
	for i := range dir {
		if dir[i].IsUnused() {
			continue
		}
		if strings.EqualFold(dir[i].GetFilename(), filename) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("file not found: %s", filename)
	}

	// --basic: detokenise the BASIC program to readable text. By default the text
	// is printed to stdout (handy for a quick look at a loader); if an output
	// directory was given, it is written there as <name>.txt instead.
	if opts.Basic {
		text, err := disk.ReadBasicText(filename)
		if err != nil {
			return fmt.Errorf("failed to detokenise %s: %w", filename, err)
		}
		if opts.OutputDir == "" {
			fmt.Print(text)
			return nil
		}
		txtPath := filepath.Join(opts.OutputDir, filename+".txt")
		if !opts.Overwrite {
			if _, err := os.Stat(txtPath); err == nil {
				return fmt.Errorf("output file already exists: %s (use overwrite to replace)", txtPath)
			}
		}
		if err := os.WriteFile(txtPath, []byte(text), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", txtPath, err)
		}
		if !opts.Quiet {
			fmt.Printf("Detokenised %s to %s\n", filename, txtPath)
		}
		return nil
	}

	// Heuristic warning: the file's PLUS3DOS header says it is a BASIC program,
	// but it is being extracted as bytes (this branch is reached only when
	// --basic was not given). This is advisory only - the extraction proceeds
	// exactly as asked - but it nudges the user toward --basic if a readable
	// listing was what they wanted.
	ext := strings.ToLower(filepath.Ext(filename))
	if !opts.Quiet && disk.IsBasicProgram(filename) {
		fmt.Fprintf(os.Stderr,
			"Warning: %s is a tokenised BASIC program; extracting it as bytes. "+
				"Use --basic to detokenise it to readable text.\n", filename)
	}

	// Extract based on file extension.
	var extractErr error

	switch {
	case ext == ".bas" && !opts.PreserveCAS:
		extractErr = disk.ExtractBasic(filename, outPath)
	case ext == ".scr":
		extractErr = disk.ExportScreen(filename, outPath)
	default:
		// Generic file export (CODE/binary and anything else).
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
