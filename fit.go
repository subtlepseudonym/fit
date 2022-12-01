package fit

import (
	"fmt"
	"io"
	"math"
	"math/big"
	"time"

	lp "github.com/influxdata/line-protocol/v2/lineprotocol"
	"github.com/jftuga/geodist"
	"github.com/subtlepseudonym/fit-go"
	"gonum.org/v1/gonum/stat"
)

const (
	TypeUnknown    = "unknown"
	TypeMonitoring = "monitor"          // only non-sport type
	SportTracking  = "All-Day Tracking" // Sport value for tracking activity
)

// Use Sport.Name mapping to capture custom activities
var sportToType map[string]string = map[string]string{
	SportTracking:       "track",
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

type Summary struct {
	Type         string            `json:"type"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Measurements []*Measurement    `json:"measurements" hash:"ignore"`
	Correlations []*Correlation    `json:"correlations" hash:"ignore"`
	Tags         map[string]string `json:"tags" hash:"ignore"`
}

type Measurement struct {
	Name   string    `json:"name"`
	Unit   string    `json:"unit"`
	Values []float64 `json:"-"`

	Maximum           float64 `json:"maximum"`
	Minimum           float64 `json:"minimum"`
	Median            float64 `json:"median"`
	Mean              float64 `json:"mean"`
	Variance          float64 `json:"variance"`
	StandardDeviation float64 `json:"standard_deviation"`
}

func (m *Measurement) CalculateStats() *Measurement {
	if len(m.Values) < 1 {
		return m
	}

	var sum float64
	m.Maximum = m.Values[0]
	m.Minimum = m.Values[0]

	for _, v := range m.Values {
		sum += v
		if v > m.Maximum {
			m.Maximum = v
		}
		if v < m.Minimum {
			m.Minimum = v
		}
	}

	numValues := len(m.Values)
	m.Mean = sum / float64(numValues)
	if numValues%2 == 0 {
		// mean of middle two values
		m.Median = (m.Values[numValues/2-1] + m.Values[numValues/2]) / 2
	} else {
		m.Median = m.Values[int(math.Floor(float64(numValues)/2))]
	}

	ss, compensation := 0.0, 0.0
	for _, v := range m.Values {
		deviation := v - m.Mean
		ss += deviation * deviation
		compensation += deviation
	}
	m.Variance = (ss - (compensation * compensation / float64(numValues))) / float64(numValues-1)
	m.StandardDeviation = math.Sqrt(m.Variance)

	return m
}

type Correlation struct {
	MeasurementA string  `json:"measurement_a"`
	MeasurementB string  `json:"measurement_b"`
	Correlation  float64 `json:"correlation"`
}

func Type(data *fit.File) (string, error) {
	switch data.Type() {
	case fit.FileTypeActivity:
		activity, err := data.Activity()
		if err != nil {
			return "", fmt.Errorf("activity: %w", err)
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
		if t, ok := sportToType[sport.Sport.Name]; ok {
			return t, nil
		} else {
			return TypeUnknown, nil
		}
	case fit.FileTypeMonitoringA, fit.FileTypeMonitoringB, fit.FileTypeMonitoringDaily:
		return TypeMonitoring, nil
	}

	return "", fmt.Errorf("file type unknown")
}

func Summarize(data *fit.File, correlates [][2]string, tags map[string]string) (*Summary, error) {
	switch data.Type() {
	case fit.FileTypeActivity:
		activity, err := data.Activity()
		if err != nil {
			return nil, fmt.Errorf("activity: %w", err)
		}

		fitType := TypeUnknown
		if t, ok := sportToType[activity.Sport.Name]; ok {
			fitType = t
		}

		numRecords := len(activity.Records)
		if numRecords < 1 {
			return nil, fmt.Errorf("records length: %d", numRecords)
		}

		summary := &Summary{
			Type:         fitType,
			StartTime:    activity.Records[0].Timestamp,
			EndTime:      activity.Records[numRecords-1].Timestamp,
			Measurements: make([]*Measurement, 0, 8),
			Correlations: make([]*Correlation, 0, len(correlates)),
			Tags:         tags,
		}

		m := map[string]*Measurement{
			"heart_rate":  &Measurement{Unit: "1 / minute"},
			"altitude":    &Measurement{Unit: "meter"},
			"temperature": &Measurement{Unit: "degrees Celsius"},
		}

		if activity.Sport.Name != SportTracking {
			m["distance"] = &Measurement{Unit: "centimeter"}
			m["vicenty_distance"] = &Measurement{Unit: "centimeter"}
		}

		if activity.Sport.Sport == fit.SportCycling {
			m["speed"] = &Measurement{Unit: "centimeter / second"}

			// indicates erroneous mean cadence value of 255
			if !(len(activity.Sessions) > 0 && activity.Sessions[0].AvgCadence == 255) {
				m["cadence"] = &Measurement{Unit: "1 / minute"}
			}
		}

		start := geodist.Coord{}
		for i, record := range activity.Records {
			if v, ok := m["heart_rate"]; ok {
				v.Values = append(v.Values, float64(record.HeartRate))
			}
			if v, ok := m["temperature"]; ok {
				v.Values = append(v.Values, float64(record.Temperature))
			}
			if v, ok := m["altitude"]; ok {
				// altitude value requires scaling
				altitude := record.GetEnhancedAltitudeScaled()
				if !math.IsNaN(altitude) {
					v.Values = append(v.Values, altitude)
				}
			}
			if v, ok := m["distance"]; ok {
				v.Values = append(v.Values, float64(record.Distance))
			}
			if v, ok := m["cadence"]; ok {
				v.Values = append(v.Values, float64(record.Cadence))
			}
			if v, ok := m["speed"]; ok {
				v.Values = append(v.Values, float64(record.EnhancedSpeed))
			}

			// don't calculate max distance from start if no distance recorded
			if i > 60 && start.Lat == 0.0 && start.Lon == 0.0 {
				continue
			}

			pos := geodist.Coord{
				Lat: record.PositionLat.Degrees(),
				Lon: record.PositionLong.Degrees(),
			}
			if math.IsNaN(pos.Lat) || math.IsNaN(pos.Lon) {
				continue
			}
			if start.Lat == 0.0 && start.Lon == 0.0 {
				start = pos
				continue
			}

			// calculate Vicenty distance from first recorded position
			_, dist, err := geodist.VincentyDistance(start, pos)
			if err != nil {
				continue
			}
			if v, ok := m["vicenty_distance"]; ok {
				// convert v to centimeters
				v.Values = append(v.Values, dist*1000)
			}
		}

		for key, measurement := range m {
			measurement.Name = key
			measurement.CalculateStats()
			summary.Measurements = append(summary.Measurements, measurement)
		}

		for _, measurements := range correlates {
			a, aok := m[measurements[0]]
			b, bok := m[measurements[1]]
			if !(aok && bok) || len(a.Values) == 0 || len(b.Values) == 0 {
				continue
			}

			correlation := stat.Correlation(a.Values, b.Values, nil)
			if math.IsNaN(correlation) {
				continue
			}

			summary.Correlations = append(summary.Correlations, &Correlation{
				MeasurementA: measurements[0],
				MeasurementB: measurements[1],
				Correlation:  correlation,
			})
		}

		return summary, nil
	}

	return nil, fmt.Errorf("unknown file type: %d", data.Type())
}

func degToRad(degrees float64) float64 {
	ret := big.NewFloat(degrees)
	ret = ret.Mul(ret, big.NewFloat(math.Pi))
	ret = ret.Quo(ret, big.NewFloat(180))
	val, _ := ret.Float64()
	return val
}

func WriteLineProtocol(out io.Writer, data *fit.File, tags map[string]string) error {
	switch data.Type() {
	case fit.FileTypeActivity:
		activity, err := data.Activity()
		if err != nil {
			return fmt.Errorf("activity: %w", err)
		}

		fitType := TypeUnknown
		if t, ok := sportToType[activity.Sport.Name]; ok {
			fitType = t
		}

		var encoder lp.Encoder
		encoder.SetPrecision(lp.Second)

		for _, record := range activity.Records {
			encoder.StartLine(fitType)

			for tag, value := range tags {
				encoder.AddTag(tag, value)
			}

			encoder.AddField("heart_rate", lp.UintValue(uint64(record.HeartRate)))
			encoder.AddField("enhanced_altitude", lp.UintValue(uint64(record.EnhancedAltitude)))
			encoder.AddField("temperature", lp.IntValue(int64(record.Temperature)))

			if activity.Sport.Name != SportTracking {
				if latitude, ok := lp.FloatValue(record.PositionLat.Degrees()); ok {
					encoder.AddField("latitude", latitude)
				}
				if longitude, ok := lp.FloatValue(record.PositionLong.Degrees()); ok {
					encoder.AddField("longitude", longitude)
				}
				encoder.AddField("distance", lp.UintValue(uint64(record.Distance)))
			}

			if activity.Sport.Sport == fit.SportCycling {
				encoder.AddField("cadence", lp.UintValue(uint64(record.Cadence)))
				encoder.AddField("enhanced_speed", lp.UintValue(uint64(record.EnhancedSpeed)))
			}

			encoder.EndLine(record.Timestamp)
			if err = encoder.Err(); err != nil {
				return fmt.Errorf("encoder: %w", err)
			}
		}

		_, err = out.Write(encoder.Bytes())
		if err != nil {
			return fmt.Errorf("write: %w", err)
		}
	}

	return nil
}
