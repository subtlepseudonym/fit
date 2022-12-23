package fit

import (
	"fmt"
	"math"

	"github.com/jftuga/geodist"
	"github.com/subtlepseudonym/fit-go"
)

type measure struct {
	Unit  string
	Unset uint
}

var DefaultMeasurements = map[string]measure{
	"altitude":    {"meter", 0xFFFFFFFF},
	"heart_rate":  {"1 / minute", 0xFF},
	"temperature": {"degrees Celsius", 0x7F},
}

var DefaultSportMeasurements = map[string]measure{
	"distance":         {"centimeter", 0xFFFFFFFF},
	"latitude":         {"degrees", 0xFF},
	"longitude":        {"degrees", 0xFF},
	"moving_speed":     {"millimeter / second", 0xFFFFFFFF},
	"speed":            {"millimeter / second", 0xFFFFFFFF},
	"vicenty_distance": {"centimeter", 0xFFFFFFFF},
}

var DefaultCyclingMeasurements = map[string]measure{
	"cadence": {"1 / minute", 0xFF},
}

const DefaultMovingThreshold = 112 // 112 mm/s ~= 0.25 mph

// Accumulator is used to calculate generated measurements that require
// multi-record context
type Accumulator struct {
	index         int
	startPosition *geodist.Coord
}

type AddFunc func(key string, value interface{})

func IsUnset(key string, value float64) bool {
	if math.IsNaN(value) {
		return true
	}

	if m, ok := DefaultMeasurements[key]; ok {
		return value >= float64(m.Unset)
	}
	if m, ok := DefaultSportMeasurements[key]; ok {
		return value >= float64(m.Unset)
	}
	if m, ok := DefaultCyclingMeasurements[key]; ok {
		return value >= float64(m.Unset)
	}

	return true
}

func ReadRecord(accumulator *Accumulator, record *fit.RecordMsg, add AddFunc) (*Accumulator, error) {
	accumulator.index += 1

	add("altitude", record.GetEnhancedAltitudeScaled())
	add("cadence", record.Cadence)
	add("distance", record.Distance)
	add("heart_rate", record.HeartRate)
	add("latitude", record.PositionLat.Degrees())
	add("longitude", record.PositionLong.Degrees())
	add("speed", record.EnhancedSpeed)
	add("temperature", record.Temperature)

	if record.EnhancedSpeed > DefaultMovingThreshold {
		add("moving_speed", float64(record.EnhancedSpeed))
	} else {
		add("moving_speed", math.NaN())
	}

	// don't calculate vicenty_distance from start if no positions recorded
	if accumulator.index > 60 && accumulator.startPosition == nil {
		return accumulator, nil
	}

	pos := geodist.Coord{
		Lat: record.PositionLat.Degrees(),
		Lon: record.PositionLong.Degrees(),
	}
	if math.IsNaN(pos.Lat) || math.IsNaN(pos.Lon) {
		return accumulator, nil
	}
	if accumulator.startPosition == nil {
		accumulator.startPosition = &pos
		return accumulator, nil
	}

	// calculate Vicenty distance from first recorded position
	_, dist, err := geodist.VincentyDistance(*accumulator.startPosition, pos)
	if err != nil {
		return accumulator, fmt.Errorf("vicenty distance: %w", err)
	}
	// convert v from kilometers to centimeters
	add("vicenty_distance", dist*100000)

	return accumulator, nil
}
