package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/influxdata/line-protocol/v2/lineprotocol"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/subtlepseudonym/fit-go"
)

const (
	defaultDeviceValue = "unknown"

	TypeCooldown   = "cooldown"
	TypeCycling    = "cycle"
	TypeMonitoring = "monitor"
	TypeStrength   = "strength"
	TypeTracking   = "track"
	TypeWalk       = "walk"

	SportBike     = "Bike"
	SportCooldown = "Cooldown"
	SportStrength = "Strength"
	SportTracking = "All-Day Tracking"
	SportWalk     = "Walk"
)

var device string

var sportToType map[string]string = map[string]string{
	SportBike:     TypeCycling,
	SportCooldown: TypeCooldown,
	SportStrength: TypeStrength,
	SportTracking: TypeTracking,
	SportWalk:     TypeWalk,
}

func main() {
	root := &cobra.Command{
		Use:   "fit",
		Short: "Interrogate and manipulate fit files",
	}

	lineCmd := &cobra.Command{
		Use:   "line",
		Short: "Convert fit file to influx line protocol",
		RunE:  line,
	}
	lineCmd.Flags().StringVar(&device, "device", defaultDeviceValue, "Telemetry device name")

	root.AddCommand(lineCmd)
	root.AddCommand(&cobra.Command{
		Use:   "type",
		Short: "Display fit file type information",
		RunE:  fitType,
	})

	pflag.Parse()
	if err := root.Execute(); err != nil {
		fmt.Printf("ERR: %s\n", err)
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
			return fmt.Errorf("decode: %w", err)
		}

		switch data.Type() {
		case fit.FileTypeActivity:
			activity, err := data.Activity()
			if err != nil {
				return fmt.Errorf("activity: %w", err)
			}
			fmt.Println(sportToType[activity.Sport.Name])
		case fit.FileTypeSport:
			sport, err := data.Sport()
			if err != nil {
				return fmt.Errorf("sport: %w", err)
			}
			fmt.Println(sportToType[sport.Sport.Name])
		case fit.FileTypeMonitoringA, fit.FileTypeMonitoringB, fit.FileTypeMonitoringDaily:
			fmt.Println(TypeMonitoring)
		}
	}

	return nil
}

func line(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		fitFile, err := os.Open(arg)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer fitFile.Close()

		data, err := fit.Decode(fitFile)
		if err != nil {
			return fmt.Errorf("decode: %w", err)
		}

		switch data.Type() {
		case fit.FileTypeActivity:
			activity, err := data.Activity()
			if err != nil {
				return fmt.Errorf("activity: %w", err)
			}

			lineFile := fmt.Sprintf("%s.line", strings.TrimSuffix(path.Base(fitFile.Name()), path.Ext(fitFile.Name())))
			output, err := os.Create(lineFile)
			if err != nil {
				return fmt.Errorf("open: %w", err)
			}
			defer output.Close()

			fitType := sportToType[activity.Sport.Name]

			var encoder lineprotocol.Encoder
			for _, record := range activity.Records {
				encoder.SetPrecision(lineprotocol.Second)
				encoder.StartLine(fitType)
				encoder.AddTag("device", device)

				encoder.AddField("heart_rate", lineprotocol.UintValue(uint64(record.HeartRate)))
				encoder.AddField("enhanced_altitude", lineprotocol.UintValue(uint64(record.EnhancedAltitude)))
				encoder.AddField("temperature", lineprotocol.IntValue(int64(record.Temperature)))

				if activity.Sport.Sport == fit.SportCycling {
					if latitude, ok := lineprotocol.FloatValue(record.PositionLat.Degrees()); ok {
						encoder.AddField("latitude", latitude)
					}
					if longitude, ok := lineprotocol.FloatValue(record.PositionLong.Degrees()); ok {
						encoder.AddField("longitude", longitude)
					}
					encoder.AddField("distance", lineprotocol.UintValue(uint64(record.Distance)))
					encoder.AddField("cadence", lineprotocol.UintValue(uint64(record.Cadence)))
					encoder.AddField("enhanced_speed", lineprotocol.UintValue(uint64(record.EnhancedSpeed)))
				}

				encoder.EndLine(record.Timestamp)
				if err = encoder.Err(); err != nil {
					return fmt.Errorf("encoder: %w", err)
				}
			}

			_, err = output.Write(encoder.Bytes())
			if err != nil {
				return fmt.Errorf("write: %w", err)
			}
		}
	}

	return nil
}