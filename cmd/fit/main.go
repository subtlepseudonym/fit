package main

import (
	"os"

	"github.com/spf13/cobra"
)

var Version string

func main() {
	root := &cobra.Command{
		Use:          "fit",
		Short:        "Interrogate and manipulate fit files",
		Version:      Version,
		SilenceUsage: true,
	}

	root.PersistentFlags().Bool("ignore-file-checksum", false, "Ignore file integrity checksum")

	root.AddCommand(NewDumpCommand())
	root.AddCommand(NewETLCommand())
	root.AddCommand(NewInspectCommand())
	root.AddCommand(NewLineCommand())
	root.AddCommand(NewSummarizeCommand())
	root.AddCommand(NewTypeCommand())

	err := root.Execute()
	if err != nil {
		os.Exit(1)
	}
}
