package main

import (
	"encoding/json"
	"fmt"
	"os"

	fitcmd "github.com/subtlepseudonym/fit"

	"github.com/spf13/cobra"
	fit "github.com/subtlepseudonym/fit-go"
)

var DefaultCorrelates = [][2]string{
	[2]string{"heart_rate", "cadence"},
	[2]string{"heart_rate", "speed"},
	[2]string{"cadence", "speed"},
}

func NewSummarizeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summarize",
		Short: "Generate an aggregated summary of given file",
		RunE:  summarize,
	}

	cmd.Flags().String("device", DefaultDevice, "Telemetry device name")

	return cmd
}

func summarize(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		file, err := os.Open(arg)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer file.Close()

		data, err := fit.Decode(file)
		if err != nil {
			return fmt.Errorf("decode: %w", err)
		}

		device, err := cmd.Flags().GetString("device")
		if err != nil {
			return fmt.Errorf("device flag: %w", err)
		}

		tags := map[string]string{
			"device": device,
		}

		summary, err := fitcmd.Summarize(data, DefaultCorrelates, tags)
		if err != nil {
			return fmt.Errorf("summarize: %w", err)
		}

		b, err := json.Marshal(summary)
		if err != nil {
			return fmt.Errorf("json marshal: %w", err)
		}
		fmt.Println(string(b))
	}

	return nil
}
