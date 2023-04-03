package main

import (
	"fmt"
	"os"

	fitcmd "github.com/subtlepseudonym/fit"

	"github.com/spf13/cobra"
	fit "github.com/subtlepseudonym/fit-go"
)

func NewTypeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "type",
		Short: "Display fit file type information",
		RunE:  fitType,
	}
}

func fitType(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		f, err := os.Open(arg)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}

		data, err := fit.Decode(f)
		if err != nil {
			ignore, _ := cmd.Flags().GetBool("ignore-file-checksum")
			_, ok := err.(fit.IntegrityError)
			if !ignore || !ok {
				return fmt.Errorf("decode: %w", err)
			}
		}

		t, err := fitcmd.Type(data)
		if err != nil {
			return fmt.Errorf("type: %w", err)
		}

		fmt.Println(t)
		return nil
	}

	return nil
}
