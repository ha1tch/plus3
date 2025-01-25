package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ha1tch/plus3/cmd/add"
	"github.com/ha1tch/plus3/cmd/create"
	"github.com/ha1tch/plus3/cmd/delete"
	"github.com/ha1tch/plus3/cmd/extract"
	"github.com/ha1tch/plus3/cmd/info"
	"github.com/ha1tch/plus3/cmd/list"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "plus3",
		Short: "A CLI tool for managing +3DOS disk images",
		Long:  `plus3 is a command-line tool for creating, managing, and extracting files from +3DOS disk images.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help() // Show help if no subcommand is provided
		},
	}

	// Add subcommands to the root command
	rootCmd.AddCommand(
		createCommand(),
		addCommand(),
		deleteCommand(),
		extractCommand(),
		listCommand(),
		infoCommand(),
	)

	// Execute the CLI
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// createCommand sets up the 'create' subcommand
func createCommand() *cobra.Command {
	opts := create.DefaultCreateOptions()
	cmd := &cobra.Command{
		Use:   "create [output file]",
		Short: "Create a new +3DOS disk image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return create.Create(args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.Label, "label", opts.Label, "Disk label (max 11 characters)")
	cmd.Flags().BoolVar(&opts.Boot, "boot", opts.Boot, "Create a bootable disk")
	cmd.Flags().BoolVar(&opts.Force, "force", opts.Force, "Overwrite existing files")
	cmd.Flags().BoolVar(&opts.Quiet, "quiet", opts.Quiet, "Suppress non-error output")

	return cmd
}

// addCommand sets up the 'add' subcommand
func addCommand() *cobra.Command {
	opts := add.DefaultAddOptions()
	cmd := &cobra.Command{
		Use:   "add [disk image] [file path]",
		Short: "Add a file to a +3DOS disk image",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return add.Add(args[0], args[1], opts)
		},
	}

	cmd.Flags().StringVarP((*string)(&opts.FileType), "type", "t", "auto", "File type (basic, code, screen, raw, auto)")
	cmd.Flags().Uint16Var(&opts.Line, "line", opts.Line, "Line number for BASIC programs")
	cmd.Flags().Uint16Var(&opts.LoadAddr, "load-addr", opts.LoadAddr, "Load address for CODE files")
	cmd.Flags().BoolVar(&opts.Force, "force", opts.Force, "Overwrite existing files")
	cmd.Flags().BoolVar(&opts.Quiet, "quiet", opts.Quiet, "Suppress non-error output")

	return cmd
}

// deleteCommand sets up the 'delete' subcommand
func deleteCommand() *cobra.Command {
	opts := delete.DefaultDeleteOptions()
	cmd := &cobra.Command{
		Use:   "delete [disk image] [filename]",
		Short: "Delete a file from a +3DOS disk image",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return delete.Delete(args[0], args[1], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Force, "force", opts.Force, "Skip confirmation")
	cmd.Flags().BoolVar(&opts.Quiet, "quiet", opts.Quiet, "Suppress non-error output")
	cmd.Flags().BoolVar(&opts.NoRecycle, "no-recycle", opts.NoRecycle, "Don't preserve deleted file info")

	return cmd
}

// extractCommand sets up the 'extract' subcommand
func extractCommand() *cobra.Command {
	opts := extract.DefaultExtractOptions()
	cmd := &cobra.Command{
		Use:   "extract [disk image] [filename]",
		Short: "Extract a file from a +3DOS disk image",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return extract.Extract(args[0], args[1], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.StripHeader, "strip-header", opts.StripHeader, "Remove +3DOS header if present")
	cmd.Flags().StringVarP(&opts.OutputDir, "output-dir", "o", opts.OutputDir, "Directory to extract files to")
	cmd.Flags().BoolVar(&opts.Overwrite, "overwrite", opts.Overwrite, "Allow overwriting existing files")
	cmd.Flags().BoolVar(&opts.Quiet, "quiet", opts.Quiet, "Suppress non-error output")

	return cmd
}

// listCommand sets up the 'list' subcommand
func listCommand() *cobra.Command {
	opts := list.DefaultListOptions()
	cmd := &cobra.Command{
		Use:   "list [disk image]",
		Short: "List the contents of a +3DOS disk image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return list.List(args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.Sort, "sort", opts.Sort, "Sort order (name, size, type)")
	cmd.Flags().BoolVar(&opts.Reverse, "reverse", opts.Reverse, "Reverse sort order")
	cmd.Flags().BoolVar(&opts.ShowDeleted, "show-deleted", opts.ShowDeleted, "Include deleted files in the listing")
	cmd.Flags().BoolVar(&opts.ShowSystem, "show-system", opts.ShowSystem, "Include system files in the listing")
	cmd.Flags().BoolVar(&opts.JSON, "json", opts.JSON, "Output in JSON format")
	cmd.Flags().BoolVar(&opts.Long, "long", opts.Long, "Show detailed information")
	cmd.Flags().StringVar(&opts.Pattern, "pattern", opts.Pattern, "Filter files by name pattern (e.g., '*.BAS')")
	cmd.Flags().StringVar(&opts.Format, "format", "dos", "Output format (options: 'ls', 'cpm', 'dos')")

	return cmd
}

// infoCommand sets up the 'info' subcommand
func infoCommand() *cobra.Command {
	opts := info.DefaultInfoOptions()
	cmd := &cobra.Command{
		Use:   "info [disk image]",
		Short: "Display information about a +3DOS disk image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return info.Info(args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", opts.JSON, "Output in JSON format")
	cmd.Flags().BoolVar(&opts.Validate, "validate", opts.Validate, "Perform disk validation")
	cmd.Flags().BoolVar(&opts.Verbose, "verbose", opts.Verbose, "Show additional details")
	cmd.Flags().BoolVar(&opts.ShowDeleted, "show-deleted", opts.ShowDeleted, "Include information about deleted files")

	return cmd
}
