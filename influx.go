package fit

import (
	"fmt"
	"io"
	"sort"

	lp "github.com/influxdata/line-protocol/v2/lineprotocol"
	"github.com/subtlepseudonym/fit-go"
)

func EncodeFunc(encoder *lp.Encoder, measurements map[string]struct{}) AddFunc {
	return func(key string, value interface{}) {
		if _, ok := measurements[key]; !ok {
			return
		}

		var val lp.Value
		switch v := value.(type) {
		case uint:
			if IsUnset(key, float64(v)) {
				return
			}
			val = lp.UintValue(uint64(v))
		case uint8:
			if IsUnset(key, float64(v)) {
				return
			}
			val = lp.UintValue(uint64(v))
		case uint16:
			if IsUnset(key, float64(v)) {
				return
			}
			val = lp.UintValue(uint64(v))
		case uint32:
			if IsUnset(key, float64(v)) {
				return
			}
			val = lp.UintValue(uint64(v))
		case uint64:
			if IsUnset(key, float64(v)) {
				return
			}
			val = lp.UintValue(v)
		case int:
			if IsUnset(key, float64(v)) {
				return
			}
			val = lp.IntValue(int64(v))
		case int8:
			if IsUnset(key, float64(v)) {
				return
			}
			val = lp.IntValue(int64(v))
		case int16:
			if IsUnset(key, float64(v)) {
				return
			}
			val = lp.IntValue(int64(v))
		case int32:
			if IsUnset(key, float64(v)) {
				return
			}
			val = lp.IntValue(int64(v))
		case int64:
			if IsUnset(key, float64(v)) {
				return
			}
			val = lp.IntValue(v)
		case float32:
			if IsUnset(key, float64(v)) {
				return
			}
			if v, ok := lp.FloatValue(float64(v)); ok {
				val = v
			} else {
				return
			}
		case float64:
			if IsUnset(key, v) {
				return
			}
			if v, ok := lp.FloatValue(v); ok {
				val = v
			} else {
				return
			}
		default:
			return
		}

		encoder.AddField(key, val)
	}
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

		measurements := make(map[string]struct{})
		for m := range DefaultMeasurements {
			measurements[m] = struct{}{}
		}
		if fitType != TypeMonitoring && activity.Sport.Name != SportTracking {
			for m := range DefaultSportMeasurements {
				measurements[m] = struct{}{}
			}
		}
		if activity.Sport.Sport == fit.SportCycling {
			for m := range DefaultCyclingMeasurements {
				measurements[m] = struct{}{}
			}
		}

		// Line protocol requires tags to be added in lexical order
		tagKeys := make([]string, 0, len(tags))
		for key, _ := range tags {
			tagKeys = append(tagKeys, key)
		}
		sort.Strings(tagKeys)

		var encoder lp.Encoder
		encoder.SetPrecision(lp.Second)

		encode := EncodeFunc(&encoder, measurements)
		acc := new(Accumulator)
		for _, record := range activity.Records {
			encoder.StartLine(fitType)

			for _, key := range tagKeys {
				encoder.AddTag(key, tags[key])
			}

			acc, err = ReadRecord(acc, record, encode)
			if err != nil {
				return fmt.Errorf("read record: %w", err)
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
