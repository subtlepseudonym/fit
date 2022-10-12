package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/influxdata/line-protocol/v2/lineprotocol"
	"github.com/spf13/cobra"
	"github.com/subtlepseudonym/fit-go"
)

const (
	defaultDeviceValue = "unknown"
	typeUnknown        = "unknown"
	typeMonitoring     = "monitor" // only non-sport type
)

// Use Sport.Name mapping to capture custom activities
var sportToType map[string]string = map[string]string{
	"All-Day Tracking":  "track",
	"American Football": "football",
	"Basketball":        "basketball",
	"Bike":              "cycle",
	"Cooldown":          "cooldown",
	"Hike":              "hike",
	"Ice Skate":         "iceskate",
	"Kayak":             "kayak",
	"MTB":               "mountain",
	"Open Water":        "openwater",
	"Pool Swim":         "swim",
	"Run":               "run",
	"SUP":               "paddleboard",
	"Ski":               "ski",
	"Snowboard":         "snowboard",
	"Soccer":            "soccer",
	"Strength":          "strength",
	"Tennis":            "tennis",
	"Treadmill":         "treadmill",
	"Walk":              "walk",
	"Yoga":              "yoga",
}

var (
	Version string // value set at build time
	device  string
)

func main() {
	root := &cobra.Command{
		Use:          "fit",
		Short:        "Interrogate and manipulate fit files",
		Version:      Version,
		SilenceUsage: true,
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
	root.AddCommand(&cobra.Command{
		Use:   "dump",
		Short: "Dump file header and ID",
		RunE:  dump,
	})

	err := root.Execute()
	if err != nil {
		os.Exit(1)
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
			if t, ok := sportToType[activity.Sport.Name]; ok {
				fmt.Println(t)
			} else {
				fmt.Println(typeUnknown)
			}
		case fit.FileTypeSport:
			sport, err := data.Sport()
			if err != nil {
				return fmt.Errorf("sport: %w", err)
			}
			if t, ok := sportToType[sport.Sport.Name]; ok {
				fmt.Println(t)
			} else {
				fmt.Println(typeUnknown)
			}
		case fit.FileTypeMonitoringA, fit.FileTypeMonitoringB, fit.FileTypeMonitoringDaily:
			fmt.Println(typeMonitoring)
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

			fitType := typeUnknown
			if t, ok := sportToType[activity.Sport.Name]; ok {
				fitType = t
			}

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

func dump(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		fitFile, err := os.Open(arg)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer fitFile.Close()

		header, fileID, err := fit.DecodeHeaderAndFileID(fitFile)
		if err != nil {
			return fmt.Errorf("decode header and file ID: %w", err)
		}

		b, err := json.Marshal(header)
		if err != nil {
			return fmt.Errorf("marshal header: %w", err)
		}
		fmt.Println(string(b))

		fid := struct {
			Type         string
			Manufacturer string
			Product      interface{}
			SerialNumber uint32
			TimeCreated  time.Time
			Number       uint16
			ProductName  string
		}{
			Type:         fileID.Type.String(),
			Manufacturer: fileID.Manufacturer.String(),
			SerialNumber: fileID.SerialNumber,
			TimeCreated:  fileID.TimeCreated,
			Number:       fileID.Number,
			ProductName:  fileID.ProductName,
		}

		product := fileID.GetProduct()
		if p, ok := product.(fit.GarminProduct); ok {
			fid.Product = p.String()
		} else {
			fid.Product = product
		}

		b, err = json.Marshal(fid)
		if err != nil {
			return fmt.Errorf("marshal file ID message: %w", err)
		}
		fmt.Println(string(b))
	}

	return nil
}
