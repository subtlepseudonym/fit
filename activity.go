package fit

import (
	"fmt"
	"math"
	"time"

	"github.com/jftuga/geodist"
	"github.com/subtlepseudonym/fit-go"
	"gonum.org/v1/gonum/stat"
)

type Activity struct {
	Type         string            `json:"type"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Measurements []*Measurement    `json:"measurements" hash:"ignore"`
	Correlations []*Correlation    `json:"correlations" hash:"ignore"`
	Tags         map[string]string `json:"tags" hash:"ignore"`

	mmap     map[string]*Measurement `json:"-"`
	startPos *geodist.Coord          `json:"-"`
}

func (a *Activity) FinalizeMeasurements(measurements []string) []*Measurement {
	for _, measurement := range measurements {
		if v, ok := a.mmap[measurement]; ok {
			if m, ok := v.Finalize(); ok {
				a.Measurements = append(a.Measurements, m)
			}
		}
	}

	return a.Measurements
}

func (s *Activity) CalculateCorrelations(correlates [][2]string) []*Correlation {
	correlations := make([]*Correlation, 0, len(correlates))
	for _, measurements := range correlates {
		a, aok := s.mmap[measurements[0]]
		b, bok := s.mmap[measurements[1]]
		if !(aok && bok) || len(a.values) == 0 || len(b.values) == 0 {
			continue
		}

		// remove unset values
		correlateA := make([]float64, 0, len(a.values))
		correlateB := make([]float64, 0, len(b.values))
		for i := range a.values {
			if uint(a.values[i]) == a.unset || uint(b.values[i]) == b.unset {
				continue
			}
			correlateA = append(correlateA, a.values[i])
			correlateB = append(correlateB, b.values[i])
		}

		correlation := stat.Correlation(correlateA, correlateB, nil)
		if math.IsNaN(correlation) {
			continue
		}

		correlations = append(correlations, &Correlation{
			MeasurementA: measurements[0],
			MeasurementB: measurements[1],
			Correlation:  correlation,
		})
	}

	return correlations
}

func (a *Activity) AddValue(key string, value interface{}) {
	var val float64
	switch v := value.(type) {
	case uint:
		val = float64(v)
	case uint8:
		val = float64(v)
	case uint16:
		val = float64(v)
	case uint32:
		val = float64(v)
	case uint64:
		val = float64(v)
	case int:
		val = float64(v)
	case int8:
		val = float64(v)
	case int16:
		val = float64(v)
	case int32:
		val = float64(v)
	case int64:
		val = float64(v)
	case float32:
		val = float64(v)
	case float64:
		val = v
	}

	m, ok := a.mmap[key]
	if !ok {
		return
	}

	// don't keep nan values
	if math.IsNaN(val) {
		val = float64(m.unset)
	}

	m.values = append(m.values, val)
	if val >= float64(m.unset) {
		return
	}

	m.count += 1
	m.sum += val
	if val > m.Maximum {
		m.Maximum = val
	}
	if val < m.Minimum {
		m.Minimum = val
	}
}

func Summarize(data *fit.File, measures []string, correlates [][2]string, tags map[string]string) (*Activity, error) {
	switch data.Type() {
	case fit.FileTypeActivity:
		fitType, err := Type(data)
		if err != nil {
			return nil, fmt.Errorf("type: %w", err)
		}

		activityData, err := data.Activity()
		if err != nil {
			return nil, fmt.Errorf("activity: %w", err)
		}

		lastIdx := len(activityData.Records) - 1
		if lastIdx < 0 {
			return nil, fmt.Errorf("file contains no records")
		}

		activity := &Activity{
			Type:         fitType,
			StartTime:    activityData.Records[0].Timestamp,
			EndTime:      activityData.Records[lastIdx].Timestamp,
			Measurements: make([]*Measurement, 0, 8),
			Correlations: make([]*Correlation, 0, len(correlates)),
			Tags:         tags,
			mmap:         make(map[string]*Measurement),
		}

		for name, m := range DefaultMeasurements {
			activity.mmap[name] = NewMeasurement(name, m.Unit, m.Unset)
		}

		if activity.Type != TypeMonitoring && activity.Type != TypeTracking {
			for name, m := range DefaultSportMeasurements {
				activity.mmap[name] = NewMeasurement(name, m.Unit, m.Unset)
			}
		}

		if activity.Type == TypeCycling {
			for name, m := range DefaultCyclingMeasurements {
				activity.mmap[name] = NewMeasurement(name, m.Unit, m.Unset)
			}
		}

		acc := new(Accumulator)
		for _, record := range activityData.Records {
			acc, err = ReadRecord(acc, record, activity.AddValue)
			if err != nil {
				return nil, fmt.Errorf("read record: %w", err)
			}
		}
		activity.Measurements = activity.FinalizeMeasurements(measures)
		activity.Correlations = activity.CalculateCorrelations(correlates)

		return activity, nil
	}

	return nil, fmt.Errorf("unknown file type: %d", data.Type())
}
