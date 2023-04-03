package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/spf13/cobra"
	fit "github.com/subtlepseudonym/fit-go"
)

var defaultNum = 10

func NewInspectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect available measurements in fit file",
		RunE:  inspect,
	}

	cmd.Flags().Int("n", defaultNum, "Number of records to output")
	return cmd
}

func inspect(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		file, err := os.Open(arg)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer file.Close()

		data, err := fit.Decode(file)
		if err != nil {
			ignore, _ := cmd.Flags().GetBool("ignore-file-checksum")
			_, ok := err.(fit.IntegrityError)
			if !ignore || !ok {
				return fmt.Errorf("decode: %w", err)
			}
		}

		switch data.Type() {
		case fit.FileTypeActivity:
			activity, err := data.Activity()
			if err != nil {
				return fmt.Errorf("activity: %w", err)
			}

			encoder := json.NewEncoder(os.Stdout)
			n, err := cmd.Flags().GetInt("n")
			if err != nil {
				return fmt.Errorf("get n flag value: %w", err)
			}
			if n > len(activity.Records) {
				n = len(activity.Records)
			}

			for i := 0; i < n; i++ {
				if activity.Records[i] == nil {
					continue
				}

				obj := make(map[string]interface{})
				recordValue := reflect.ValueOf(*activity.Records[i])
				for i := 0; i < recordValue.NumField(); i++ {
					field := recordValue.Field(i)
					if err != nil {
						return fmt.Errorf("get field by index %d: %w", i, err)
					}
					if !field.IsValid() || field.IsZero() {
						continue
					}

					if field.CanUint() {
						u := field.Uint()
						bits := field.Type().Bits()
						if u == uint64(1<<bits)-1 {
							continue
						}
					} else if field.CanInt() {
						i := field.Int()
						bits := field.Type().Bits()
						if i == int64(1<<bits)/2-1 {
							continue
						}
					}

					stringFunc := field.MethodByName("String")
					if stringFunc.IsValid() {
						str := stringFunc.Call(nil)[0].String()
						if str != "" && str != "Invalid" {
							obj[recordValue.Type().Field(i).Name] = str
						}
					} else {
						obj[recordValue.Type().Field(i).Name] = field.Interface()
					}
				}

				err = encoder.Encode(obj)
				if err != nil {
					return fmt.Errorf("encode activity record: %w", err)
				}
			}
		}
	}

	return nil
}
