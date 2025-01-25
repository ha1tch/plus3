// file: cmd/list/list.go

package list

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ha1tch/plus3/pkg/diskimg"
)

// FileEntry represents a file in the directory listing
type FileEntry struct {
	Name       string    `json:"name"`
	Size       int       `json:"size"`
	Type       string    `json:"type"`
	Attributes []string  `json:"attributes"`
	Modified   time.Time `json:"modified,omitempty"`
}

// Format defines the listing output format
type Format int

const (
	FormatLS   Format = iota // Unix ls-style format
	FormatCPM               // Traditional CPM format
	FormatDOS               // DOS dir-style format
)

// ListOptions configures the directory listing
type ListOptions struct {
	DiskPath    string   // Path to disk image (needed for DOS format display)
	Format      Format   // Output format style
	JSON        bool     // Output in JSON format
	Long        bool     // Show detailed information
	Sort        string   // Sort order: name, size, type
	Reverse     bool     // Reverse sort order
	ShowSystem  bool     // Show system files
	ShowDeleted bool     // Show deleted files
	Pattern     string   // Filter by filename pattern
	Quiet       bool     // Suppress non-error output
	Human       bool     // Human-readable sizes
}

// DefaultListOptions returns default options for List
func DefaultListOptions() *ListOptions {
	return &ListOptions{
		Format:      FormatDOS,  // Default to familiar DOS format
		JSON:        false,
		Long:        false,
		Sort:        "name",
		Reverse:     false,
		ShowSystem:  false,
		ShowDeleted: false,
		Pattern:     "*",
		Quiet:       false,
		Human:       true,
	}
}

// List displays the contents of a disk image
func List(diskPath string, opts *ListOptions) error {
	// Validate options
	if opts == nil {
		opts = DefaultListOptions()
	}
	opts.DiskPath = diskPath

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

	// Collect file entries
	var files []FileEntry
	for _, entry := range dir.Entries {
		if shouldIncludeFile(&entry, opts) {
			file := fileEntryFromDirEntry(&entry)
			if matchesPattern(file.Name, opts.Pattern) {
				files = append(files, file)
			}
		}
	}

	// Sort files
	sortFiles(files, opts)

	// Output listing
	if opts.JSON {
		return outputJSON(files)
	}

	switch opts.Format {
	case FormatLS:
		return outputLS(files, opts)
	case FormatCPM:
		return outputCPM(files, opts)
	case FormatDOS:
		return outputDOS(files, opts)
	default:
		return fmt.Errorf("unknown format specified")
	}
}

func shouldIncludeFile(entry *diskimg.DirectoryEntry, opts *ListOptions) bool {
	if entry.IsUnused() {
		return false
	}
	if entry.IsDeleted() && !opts.ShowDeleted {
		return false
	}

	attrs := &diskimg.FileAttributes{}
	attrs.ReadFromDirectoryEntry(entry)
	
	if attrs.System && !opts.ShowSystem {
		return false
	}

	return true
}

func fileEntryFromDirEntry(entry *diskimg.DirectoryEntry) FileEntry {
	attrs := &diskimg.FileAttributes{}
	attrs.ReadFromDirectoryEntry(entry)

	var attrList []string
	if attrs.ReadOnly {
		attrList = append(attrList, "read-only")
	}
	if attrs.System {
		attrList = append(attrList, "system")
	}
	if attrs.Archived {
		attrList = append(attrList, "archived")
	}

	return FileEntry{
		Name:       entry.GetFilename(),
		Size:       int(entry.LogicalSize) * 128, // Convert records to bytes
		Type:       determineFileType(entry),
		Attributes: attrList,
	}
}

func determineFileType(entry *diskimg.DirectoryEntry) string {
	ext := strings.ToUpper(filepath.Ext(entry.GetFilename()))
	switch ext {
	case ".BAS":
		return "BASIC"
	case ".SCR":
		return "Screen$"
	case ".BIN":
		return "Code"
	default:
		return "Data"
	}
}

func matchesPattern(name, pattern string) bool {
	if pattern == "*" {
		return true
	}
	matched, err := filepath.Match(strings.ToUpper(pattern), strings.ToUpper(name))
	return err == nil && matched
}

func sortFiles(files []FileEntry, opts *ListOptions) {
	less := func(i, j int) bool {
		var result bool
		switch strings.ToLower(opts.Sort) {
		case "size":
			result = files[i].Size < files[j].Size
		case "type":
			result = files[i].Type < files[j].Type
		default: // "name"
			result = files[i].Name < files[j].Name
		}
		if opts.Reverse {
			return !result
		}
		return result
	}
	sort.Slice(files, less)
}

func outputJSON(files []FileEntry) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(files)
}

func outputCPM(files []FileEntry, opts *ListOptions) error {
	if len(files) == 0 {
		if !opts.Quiet {
			fmt.Println("No files found")
		}
		return nil
	}

	formatSize := func(size int) string {
		if opts.Human {
			return humanSize(size)
		}
		return fmt.Sprintf("%dK", (size+1023)/1024)
	}

	w := os.Stdout
	fmt.Fprintln(w, "Name     Bytes   Recs  Attributes")
	fmt.Fprintln(w, "----     -----   ----  ----------")
	
	for _, file := range files {
		recs := (file.Size + 127) / 128
		fmt.Fprintf(w, "%-8s  %6s  %4d   %s\n",
			file.Name,
			formatSize(file.Size),
			recs,
			strings.Join(file.Attributes, ", "))
	}

	return nil
}

func outputDOS(files []FileEntry, opts *ListOptions) error {
	if len(files) == 0 {
		if !opts.Quiet {
			fmt.Printf(" Directory of %s\n\n", opts.DiskPath)
			fmt.Println("File Not Found")
			fmt.Printf("\n    0 File(s)              0 bytes\n")
		}
		return nil
	}

	// Print header
	fmt.Printf("\n Directory of %s\n\n", opts.DiskPath)

	// Calculate totals
	var totalFiles int
	var totalBytes int64
	
	// Print each file
	for _, file := range files {
		// Get attributes string
		attrStr := ""
		for _, attr := range file.Attributes {
			switch attr {
			case "read-only":
				attrStr += "R"
			case "system":
				attrStr += "S"
			case "archived":
				attrStr += "A"
			default:
				attrStr += " "
			}
		}
		attrStr = fmt.Sprintf("%-5s", attrStr)

		// Format date/time (if available)
		timeStr := file.Modified.Format("02/01/2006  15:04")
		if file.Modified.IsZero() {
			timeStr = "                "
		}

		// Format size with comma separators
		sizeStr := formatWithCommas(file.Size)
		
		// Print entry
		if file.Type == "Directory" {
			fmt.Printf("%s  %s    <DIR>          %s\n", 
				timeStr, attrStr, file.Name)
		} else {
			fmt.Printf("%s  %s  %14s %s\n", 
				timeStr, attrStr, sizeStr, file.Name)
		}

		totalFiles++
		totalBytes += int64(file.Size)
	}

	// Print summary
	fmt.Printf("\n    %d File(s)    %14s bytes\n", 
		totalFiles, formatWithCommas(int(totalBytes)))
	if !opts.ShowSystem {
		fmt.Printf("                %14s bytes free\n", 
			formatWithCommas(180*1024-int(totalBytes)))
	}

	return nil
}

func formatWithCommas(n int) string {
	// Handle negative numbers
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}

	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return sign + str
	}

	var result []byte
	pos := len(str) - 1
	count := 0

	for pos >= 0 {
		if count > 0 && count%3 == 0 {
			result = append([]byte{','}, result...)
		}
		result = append([]byte{str[pos]}, result...)
		pos--
		count++
	}

	return sign + string(result)
}

func humanSize(size int) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	sizef := float64(size)
	for _, unit := range []string{"K", "M"} {
		if sizef < 1024 {
			return fmt.Sprintf("%.1f%s", sizef, unit)
		}
		sizef /= 1024
	}
	return fmt.Sprintf("%.1fM", sizef)
}