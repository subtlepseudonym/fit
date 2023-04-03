package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	fitcmd "github.com/subtlepseudonym/fit"

	"github.com/spf13/cobra"
	fit "github.com/subtlepseudonym/fit-go"
)

const (
	DefaultDevice = "unknown"
)

var (
	output string
)

func NewLineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "line",
		Short: "Convert fit file to influx line protocol",
		RunE:  line,
	}

	cmd.Flags().String("device", DefaultDevice, "Telemetry device name")

	return cmd
}

func line(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		file, err := os.Open(arg)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer file.Close()

		data, err := fit.Decode(file)
		if err != nil {
			ignore, _ := cmd.Flags().GetBool("ignore-file-checksum")
			_, ok := err.(fit.IntegrityError)
			if !ignore || !ok {
				return fmt.Errorf("decode: %w", err)
			}
		}

		lineFile := fmt.Sprintf("%s.line", strings.TrimSuffix(path.Base(file.Name()), path.Ext(file.Name())))
		output, err := os.Create(lineFile)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer output.Close()

		device, err := cmd.Flags().GetString("device")
		if err != nil {
			return fmt.Errorf("device flag: %w", err)
		}

		tags := map[string]string{
			"device": device,
		}

		err = fitcmd.WriteLineProtocol(output, data, tags)
		if err != nil {
			return fmt.Errorf("write line protocol: %w", err)
		}
	}

	return nil
}
