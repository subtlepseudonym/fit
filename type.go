package fit

import (
	"fmt"

	"github.com/subtlepseudonym/fit-go"
)

const (
	SportTracking  = "All-Day Tracking" // Sport value for tracking activity
	TypeCycling    = "cycle"
	TypeMonitoring = "monitor" // only non-sport type
	TypeTracking   = "track"
	TypeUnknown    = "unknown"
)

// Use Sport.Name mapping to capture custom activities
var sportToType map[string]string = map[string]string{
	SportTracking:       TypeTracking,
	"American Football": "football",
	"Basketball":        "basketball",
	"Bike":              TypeCycling,
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

func Type(data *fit.File) (string, error) {
	switch data.Type() {
	case fit.FileTypeActivity:
		activity, err := data.Activity()
		if err != nil {
			return "", fmt.Errorf("activity: %w", err)
		}
		if activity.Sport == nil {
			return TypeUnknown, nil
		}
		if t, ok := sportToType[activity.Sport.Name]; ok {
			return t, nil
		} else {
			return TypeUnknown, nil
		}
	case fit.FileTypeSport:
		sport, err := data.Sport()
		if err != nil {
			return "", fmt.Errorf("sport: %w", err)
		}
		if sport.Sport == nil {
			return TypeUnknown, nil
		}
		if t, ok := sportToType[sport.Sport.Name]; ok {
			return t, nil
		} else {
			return TypeUnknown, nil
		}
	case fit.FileTypeMonitoringA, fit.FileTypeMonitoringB, fit.FileTypeMonitoringDaily:
		return TypeMonitoring, nil
	}

	return TypeUnknown, fmt.Errorf("file type unknown")
}
