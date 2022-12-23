package fit

import (
	"math"
	"sort"
)

type Measurement struct {
	Name string `json:"name"`
	Unit string `json:"unit"`

	Maximum           float64 `json:"maximum"`
	Minimum           float64 `json:"minimum"`
	Median            float64 `json:"median"`
	Mean              float64 `json:"mean"`
	Variance          float64 `json:"variance"`
	StandardDeviation float64 `json:"standard_deviation"`

	unset  uint      `json:"-"`
	count  uint      `json:"-"`
	sum    float64   `json:"-"`
	values []float64 `json:"-"`
}

func NewMeasurement(name, unit string, unset uint) *Measurement {
	return &Measurement{
		Name:    name,
		Unit:    unit,
		Minimum: float64(unset),
		unset:   unset,
	}
}

func (m *Measurement) Valid() bool {
	return len(m.values) > 1 && m.count > 0
}

func (m *Measurement) Finalize() (*Measurement, bool) {
	if !m.Valid() {
		return m, false
	}

	m.Mean = m.sum / float64(m.count)

	ss, compensation := 0.0, 0.0
	valid := make([]float64, 0, len(m.values))
	for _, v := range m.values {
		if v >= float64(m.unset) {
			continue
		}

		valid = append(valid, v)
		deviation := v - m.Mean
		ss += deviation * deviation
		compensation += deviation
	}

	m.Variance = (ss - (compensation * compensation / float64(m.count))) / (float64(m.count) - 1)
	m.StandardDeviation = math.Sqrt(m.Variance)

	sort.Float64s(valid)
	if m.count%2 == 0 {
		// mean of middle two values
		m.Median = (valid[m.count/2-1] + valid[m.count/2]) / 2
	} else {
		m.Median = valid[m.count/2]
	}

	return m, len(valid) > 0
}

type Correlation struct {
	MeasurementA string  `json:"measurement_a"`
	MeasurementB string  `json:"measurement_b"`
	Correlation  float64 `json:"correlation"`
}
