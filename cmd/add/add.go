// file: cmd/add/add.go

package add

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ha1tch/plus3/pkg/diskimg"
)

// FileType defines the type of file being added to the disk image
type FileType int

const (
	// TypeAuto automatically determines file type from extension
	TypeAuto FileType = iota
	// TypeBasic indicates an already-tokenised BASIC program
	TypeBasic
	// TypeBasicText indicates plain-text BASIC source to be tokenised on import
	TypeBasicText
	// TypeCode indicates machine code or binary data
	TypeCode
	// TypeScreen indicates a screen dump
	TypeScreen
	// TypeRaw indicates data without special handling
	TypeRaw
)

// AddOptions configures the Add operation
type AddOptions struct {
	FileType FileType
	Line     uint16 // Line number for BASIC programs
	LoadAddr uint16 // Load address for CODE files
	Force    bool   // Allow overwriting existing files
	Quiet    bool   // Suppress non-error output
}

// DefaultAddOptions returns default options for Add
func DefaultAddOptions() *AddOptions {
	return &AddOptions{
		FileType: TypeAuto,
		Line:     10,    // Standard default for BASIC
		LoadAddr: 32768, // Standard default address
		Force:    false,
		Quiet:    false,
	}
}

// determineFileType identifies file type from extension
func determineFileType(path string) FileType {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".bas":
		return TypeBasic
	case ".bin":
		return TypeCode
	case ".scr":
		return TypeScreen
	default:
		return TypeRaw
	}
}

// Add imports a file into the disk image
func Add(diskPath string, filePath string, opts *AddOptions) error {
	// Validate options
	if opts == nil {
		opts = DefaultAddOptions()
	}

	// Validate input file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %w", err)
	}

	// Check input file size
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	if info.Size() > 8*1024*1024 { // +3DOS 8MB limit
		return fmt.Errorf("file too large for +3DOS (max 8MB)")
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

	// Determine file type if auto
	fileType := opts.FileType
	if fileType == TypeAuto {
		fileType = determineFileType(filePath)
	}

	// Check if file already exists unless force is true
	if !opts.Force {
		dir, err := disk.GetDirectory()
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}

		destName := strings.ToUpper(filepath.Base(filePath))
		for i := range dir {
			if dir[i].IsUnused() {
				continue
			}
			if strings.ToUpper(dir[i].GetFilename()) == destName {
				return fmt.Errorf("file already exists: %s (use force to overwrite)", destName)
			}
		}
	}

	// Import based on file type
	var importErr error
	switch fileType {
	case TypeBasic:
		// Advisory: if the input does not parse as tokenised BASIC (e.g. it is
		// plain-text source), -t basic will store it verbatim without
		// tokenising, which will not run on the +3. Warn but proceed as asked.
		if !opts.Quiet {
			if data, rerr := os.ReadFile(filePath); rerr == nil && len(data) > 0 && !diskimg.LooksTokenised(data) && looksLikeText(data) {
				fmt.Fprintf(os.Stderr,
					"Warning: %s does not look like tokenised BASIC; -t basic stores it "+
						"verbatim. If this is plain-text source, use -t basictext.\n", filepath.Base(filePath))
			}
		}
		importErr = disk.ImportBasicProgram(filePath, opts.Line)
	case TypeBasicText:
		// Advisory: if the input already parses as tokenised BASIC, the user
		// likely meant -t basic (store verbatim) rather than -t basictext
		// (tokenise). Warn but proceed as asked.
		if !opts.Quiet {
			if data, rerr := os.ReadFile(filePath); rerr == nil && diskimg.LooksTokenised(data) {
				fmt.Fprintf(os.Stderr,
					"Warning: %s already looks like tokenised BASIC, but -t basictext will "+
						"tokenise it again. Did you mean -t basic?\n", filepath.Base(filePath))
			}
		}
		importErr = disk.ImportBasicText(filePath, opts.Line)
	case TypeCode:
		importErr = disk.ImportCode(filePath, opts.LoadAddr)
	case TypeScreen:
		importErr = disk.ImportScreen(filePath)
	default:
		importErr = disk.ImportRaw(filePath)
	}

	if importErr != nil {
		return fmt.Errorf("failed to import file: %w", importErr)
	}

	// Save disk changes
	if err := disk.SaveToFile(diskPath); err != nil {
		return fmt.Errorf("failed to save disk: %w", err)
	}

	if !opts.Quiet {
		fmt.Printf("Added %s to disk image\n", filepath.Base(filePath))
	}

	return nil
}

// looksLikeText reports whether data is plausibly plain-text BASIC source: it
// begins with an ASCII digit (a line number) and is predominantly printable
// ASCII. Used only to decide whether to show an advisory warning.
func looksLikeText(data []byte) bool {
	if len(data) == 0 || data[0] < '0' || data[0] > '9' {
		return false
	}
	printable := 0
	for _, b := range data {
		if b == '\n' || b == '\r' || b == '\t' || (b >= 0x20 && b < 0x7F) {
			printable++
		}
	}
	// Overwhelmingly printable suggests text rather than tokenised bytes.
	return printable*10 >= len(data)*9
}
