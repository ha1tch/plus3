// Command plus3 is a CLI for creating, managing, and extracting files from
// +3DOS disk images. It uses only the standard library (no external CLI
// framework) so the project has no third-party dependencies.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ha1tch/plus3/cmd/add"
	"github.com/ha1tch/plus3/cmd/create"
	"github.com/ha1tch/plus3/cmd/delete"
	"github.com/ha1tch/plus3/cmd/extract"
	"github.com/ha1tch/plus3/cmd/info"
	"github.com/ha1tch/plus3/cmd/list"
	"github.com/ha1tch/plus3/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "-h", "--help", "help":
		usage()
		return
	case "-v", "--version", "version":
		fmt.Printf("plus3 version %s\n", version.Version)
		return
	}

	var err error
	switch cmd {
	case "create":
		err = runCreate(args)
	case "add":
		err = runAdd(args)
	case "delete":
		err = runDelete(args)
	case "extract":
		err = runExtract(args)
	case "list":
		err = runList(args)
	case "info":
		err = runInfo(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Printf(`plus3 %s - manage +3DOS disk images

Usage:
  plus3 <command> [flags] [arguments]

Commands:
  create   [flags] <disk.dsk>            Create a new +3DOS disk image
  add      [flags] <disk.dsk> <file>     Add a file to a disk image
  list     [flags] <disk.dsk>            List the contents of a disk image
  info     [flags] <disk.dsk>            Display information about a disk image
  extract  [flags] <disk.dsk> <name>     Extract a file from a disk image
  delete   [flags] <disk.dsk> <name>     Delete a file from a disk image

Other:
  plus3 --version                        Show the version
  plus3 <command> -h                     Show flags for a command

Run "plus3 <command> -h" for the flags accepted by each command.
`, version.Version)
}

// newFlagSet builds a FlagSet that, on -h or a parse error, prints a one-line
// usage for the subcommand followed by its flag defaults.
func newFlagSet(name, argSpec string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: plus3 %s [flags] %s\n\nFlags:\n", name, argSpec)
		fs.PrintDefaults()
	}
	return fs
}

// parseInterleaved parses flags that may be interleaved with positional
// arguments. The standard flag package stops at the first non-flag argument;
// this keeps going, collecting positionals and re-parsing the remaining flags,
// so "plus3 create disk.dsk --force" works the same as "--force disk.dsk".
func parseInterleaved(fs *flag.FlagSet, args []string) error {
	var positionals []string
	for {
		if err := fs.Parse(args); err != nil {
			return err
		}
		// fs.Args() are the leftover args starting at the first non-flag.
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		// Take the first leftover as a positional, then continue parsing the
		// remainder (which may contain more flags).
		positionals = append(positionals, rest[0])
		args = rest[1:]
	}
	// Re-expose the collected positionals through the FlagSet by parsing a
	// flag-free arg list, so fs.Arg(n)/fs.NArg() report the positionals.
	return fs.Parse(positionals)
}

// requireArgs checks the positional argument count after flag parsing.
func requireArgs(fs *flag.FlagSet, n int) error {
	if fs.NArg() != n {
		fs.Usage()
		return fmt.Errorf("expected %d argument(s), got %d", n, fs.NArg())
	}
	return nil
}

func runCreate(args []string) error {
	opts := create.DefaultCreateOptions()
	fs := newFlagSet("create", "<disk.dsk>")
	fs.StringVar(&opts.Label, "label", opts.Label, "Disk label (max 11 characters)")
	fs.BoolVar(&opts.Boot, "boot", opts.Boot, "Create a bootable disk")
	fs.BoolVar(&opts.Force, "force", opts.Force, "Overwrite existing files")
	fs.BoolVar(&opts.Quiet, "quiet", opts.Quiet, "Suppress non-error output")
	if err := parseInterleaved(fs, args); err != nil {
		return err
	}
	if err := requireArgs(fs, 1); err != nil {
		return err
	}
	return create.Create(fs.Arg(0), opts)
}

func runAdd(args []string) error {
	opts := add.DefaultAddOptions()
	var ftype string
	fs := newFlagSet("add", "<disk.dsk> <file>")
	// -t and --type are equivalent.
	fs.StringVar(&ftype, "type", "auto", "File type (basic, basictext, code, screen, raw, auto)")
	fs.StringVar(&ftype, "t", "auto", "File type (shorthand for --type)")
	fs.Func("line", "Line number for BASIC programs", uint16Flag(&opts.Line))
	fs.Func("load-addr", "Load address for CODE files", uint16Flag(&opts.LoadAddr))
	fs.BoolVar(&opts.Force, "force", opts.Force, "Overwrite existing files")
	fs.BoolVar(&opts.Quiet, "quiet", opts.Quiet, "Suppress non-error output")
	if err := parseInterleaved(fs, args); err != nil {
		return err
	}
	if err := requireArgs(fs, 2); err != nil {
		return err
	}
	switch ftype {
	case "basic":
		opts.FileType = add.TypeBasic
	case "basictext", "basic-text":
		opts.FileType = add.TypeBasicText
	case "code":
		opts.FileType = add.TypeCode
	case "screen":
		opts.FileType = add.TypeScreen
	case "raw":
		opts.FileType = add.TypeRaw
	default:
		opts.FileType = add.TypeAuto
	}
	return add.Add(fs.Arg(0), fs.Arg(1), opts)
}

func runDelete(args []string) error {
	opts := delete.DefaultDeleteOptions()
	fs := newFlagSet("delete", "<disk.dsk> <name>")
	fs.BoolVar(&opts.Force, "force", opts.Force, "Skip confirmation")
	fs.BoolVar(&opts.Quiet, "quiet", opts.Quiet, "Suppress non-error output")
	fs.BoolVar(&opts.NoRecycle, "no-recycle", opts.NoRecycle, "Don't preserve deleted file info")
	if err := parseInterleaved(fs, args); err != nil {
		return err
	}
	if err := requireArgs(fs, 2); err != nil {
		return err
	}
	return delete.Delete(fs.Arg(0), fs.Arg(1), opts)
}

func runExtract(args []string) error {
	opts := extract.DefaultExtractOptions()
	fs := newFlagSet("extract", "<disk.dsk> <name>")
	fs.BoolVar(&opts.StripHeader, "strip-header", opts.StripHeader, "Remove +3DOS header if present")
	// -o and --output-dir are equivalent.
	fs.StringVar(&opts.OutputDir, "output-dir", opts.OutputDir, "Directory to extract files to")
	fs.StringVar(&opts.OutputDir, "o", opts.OutputDir, "Directory to extract files to (shorthand for --output-dir)")
	fs.BoolVar(&opts.Overwrite, "overwrite", opts.Overwrite, "Allow overwriting existing files")
	fs.BoolVar(&opts.Quiet, "quiet", opts.Quiet, "Suppress non-error output")
	fs.BoolVar(&opts.Basic, "basic", opts.Basic, "Detokenise a BASIC program to readable text (stdout, or <name>.txt with -o)")
	if err := parseInterleaved(fs, args); err != nil {
		return err
	}
	if err := requireArgs(fs, 2); err != nil {
		return err
	}
	return extract.Extract(fs.Arg(0), fs.Arg(1), opts)
}

func runList(args []string) error {
	opts := list.DefaultListOptions()
	var format string
	fs := newFlagSet("list", "<disk.dsk>")
	fs.StringVar(&opts.Sort, "sort", opts.Sort, "Sort order (name, size, type)")
	fs.BoolVar(&opts.Reverse, "reverse", opts.Reverse, "Reverse sort order")
	fs.BoolVar(&opts.ShowDeleted, "show-deleted", opts.ShowDeleted, "Include deleted files in the listing")
	fs.BoolVar(&opts.ShowSystem, "show-system", opts.ShowSystem, "Include system files in the listing")
	fs.BoolVar(&opts.JSON, "json", opts.JSON, "Output in JSON format")
	fs.BoolVar(&opts.Long, "long", opts.Long, "Show detailed information")
	fs.StringVar(&opts.Pattern, "pattern", opts.Pattern, "Filter files by name pattern (e.g., '*.BAS')")
	fs.StringVar(&format, "format", "dos", "Output format (options: 'ls', 'cpm', 'dos')")
	if err := parseInterleaved(fs, args); err != nil {
		return err
	}
	if err := requireArgs(fs, 1); err != nil {
		return err
	}
	switch format {
	case "ls":
		opts.Format = list.FormatLS
	case "cpm":
		opts.Format = list.FormatCPM
	default:
		opts.Format = list.FormatDOS
	}
	return list.List(fs.Arg(0), opts)
}

func runInfo(args []string) error {
	opts := info.DefaultInfoOptions()
	fs := newFlagSet("info", "<disk.dsk>")
	fs.BoolVar(&opts.JSON, "json", opts.JSON, "Output in JSON format")
	fs.BoolVar(&opts.Validate, "validate", opts.Validate, "Perform disk validation")
	fs.BoolVar(&opts.Verbose, "verbose", opts.Verbose, "Show additional details")
	fs.BoolVar(&opts.ShowDeleted, "show-deleted", opts.ShowDeleted, "Include information about deleted files")
	if err := parseInterleaved(fs, args); err != nil {
		return err
	}
	if err := requireArgs(fs, 1); err != nil {
		return err
	}
	return info.Info(fs.Arg(0), opts)
}

// uint16Flag returns a flag.Func handler that parses a uint16 (decimal, or 0x
// hex) into the target.
func uint16Flag(target *uint16) func(string) error {
	return func(s string) error {
		var v uint64
		var err error
		if len(s) > 2 && (s[0:2] == "0x" || s[0:2] == "0X") {
			_, err = fmt.Sscanf(s[2:], "%x", &v)
		} else {
			_, err = fmt.Sscanf(s, "%d", &v)
		}
		if err != nil {
			return fmt.Errorf("invalid number %q: %v", s, err)
		}
		if v > 0xFFFF {
			return fmt.Errorf("value %s out of range for uint16", s)
		}
		*target = uint16(v)
		return nil
	}
}
