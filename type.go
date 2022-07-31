package main

import (
	"fmt"
	"os"

	"github.com/tormoder/fit"
)

const (
	TypeMonitoring = "monitor"
	TypeCycling    = "cycle"
	TypeCooldown   = "cooldown"
	TypeTracking   = "track"
	TypeStrength   = "strength"
	TypeWalk       = "walk"
)

func main() {
	var files []*os.File
	for _, arg := range os.Args[1:] {
		f, err := os.Open(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open: %s", err)
			os.Exit(1)
		}
		files = append(files, f)
	}

	for _, f := range files {
		data, err := fit.Decode(f, fit.WithUnknownFields())
		if err != nil {
			fmt.Fprintf(os.Stderr, "fit decode: %s: %s", f.Name(), err)
			continue
		}

		switch data.Type() {
		case fit.FileTypeActivity:
			activity, err := data.Activity()
			if err != nil {
				fmt.Fprintf(os.Stderr, "fit activity: %s: %s", f.Name(), err)
				continue
			}

			fmt.Fprintf(os.Stdout, "activity: %v\n", activity.Activity)
			session := activity.Sessions[0]
			if session.Sport != fit.SportGeneric {
				fmt.Fprintf(os.Stdout, "%#v\n", data.UnknownFields)
			}
		case fit.FileTypeMonitoringA:
			monitor, err := data.MonitoringA()
			if err != nil {
				fmt.Fprintf(os.Stderr, "fit monitoring a: %s: %s", f.Name(), err)
				continue
			}

			fmt.Fprintf(os.Stdout, "monitor a: %v\n", monitor.MonitoringInfo)
		case fit.FileTypeMonitoringB:
			monitor, err := data.MonitoringB()
			if err != nil {
				fmt.Fprintf(os.Stderr, "fit monitoring b: %s: %s", f.Name(), err)
				continue
			}

			fmt.Fprintf(os.Stdout, "monitor b: %v\n", monitor.MonitoringInfo)
		case fit.FileTypeMonitoringDaily:
			monitor, err := data.MonitoringDaily()
			if err != nil {
				fmt.Fprintf(os.Stderr, "fit monitoring daily: %s: %s", f.Name(), err)
				continue
			}

			fmt.Fprintf(os.Stdout, "monitor daily: %v\n", monitor.MonitoringInfo)
		case fit.FileTypeSport:
			sport, err := data.Sport()
			if err != nil {
				fmt.Fprintf(os.Stderr, "fit sport: %s: %s", f.Name(), err)
				continue
			}

			fmt.Fprintf(os.Stdout, "sport: %v", sport.Sport)
		}
	}
}
