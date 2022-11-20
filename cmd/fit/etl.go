package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	fitcmd "github.com/subtlepseudonym/fit"

	"github.com/influxdata/influxdb-client-go/v2"
	_ "github.com/lib/pq"
	"github.com/mitchellh/hashstructure"
	"github.com/scru128/go-scru128"
	"github.com/spf13/cobra"
	fit "github.com/subtlepseudonym/fit-go"
)

const insertActivityFormat = `
INSERT INTO %s
(
	id,
	hash,
	type,
	start_time,
	end_time,
	max_distance_from_start,
	tags
) VALUES (
	'%s', %d, '%s', '%s', '%s',
	%f, '%s'
);
`

const insertMeasurementFormat = `
INSERT INTO %s
(
	id,
	activity_id,
	name,
	unit,
	maximum,
	minimum,
	median,
	mean,
	variance,
	standard_deviation
) VALUES (
	'%s', '%s', '%s', '%s',
	%f, %f, %f, %f, %f, %f
);
`

const insertCorrelationFormat = `
INSERT INTO %s
(
	id,
	activity_id,
	measurement_a,
	measurement_b,
	correlation
) VALUES (
	'%s', '%s', '%s', '%s', %f
);
`

func NewETLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "etl",
		Short: "ETL the given file into downstream storage",
		RunE:  etl,
	}

	flags := cmd.Flags()
	flags.String("device", DefaultDevice, "Telemetry device name")

	flags.String("postgres", "", "Postgres DSN")
	flags.String("postgres_activity_table", "activity", "Postgres table")
	flags.String("postgres_measurement_table", "measurement", "Postgres table")
	flags.String("postgres_correlation_table", "correlation", "Postgres table")
	flags.String("influx_host", "", "InfluxDB DSN")
	flags.String("influx_token", "", "InfluxDB API token")
	flags.String("influx_org", "default", "InfluxDB organization")
	flags.String("influx_bucket", "fit", "InfluxDB bucket")

	cmd.MarkFlagRequired("postgres")
	cmd.MarkFlagRequired("influx_host")
	cmd.MarkFlagRequired("influx_token")

	return cmd
}

func etl(cmd *cobra.Command, args []string) (ret error) {
	for _, arg := range args {
		file, err := os.Open(arg)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer file.Close()

		data, err := fit.Decode(file)
		if err != nil {
			return fmt.Errorf("decode: %w", err)
		}

		device, err := cmd.Flags().GetString("device")
		if err != nil {
			return fmt.Errorf("device flag: %w", err)
		}

		tags := map[string]string{
			"device": device,
		}

		// set up postgres client
		flags := cmd.Flags()
		postgresDSN, _ := flags.GetString("postgres")
		activityTable, _ := flags.GetString("postgres_activity_table")
		measurementTable, _ := flags.GetString("postgres_measurement_table")
		correlationTable, _ := flags.GetString("postgres_correlation_table")
		db, err := sql.Open("postgres", postgresDSN)
		if err != nil {
			return fmt.Errorf("sql open: %w", err)
		}
		defer db.Close()

		summary, err := fitcmd.Summarize(data, tags)
		if err != nil {
			return fmt.Errorf("summarize: %w", err)
		}

		summaryHash, err := hashstructure.Hash(summary, nil)
		if err != nil {
			return fmt.Errorf("hash summary: %w", err)
		}

		queries, err := buildInsertQueries(activityTable, measurementTable, correlationTable, summary, int64(summaryHash))
		if err != nil {
			return fmt.Errorf("build insert query: %w", err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin sql transaction: %w", err)
		}
		defer func() {
			if ret != nil {
				err := tx.Rollback()
				if err != nil {
					fmt.Println("ERR: failed to rollback transaction:", err)
				}
			}
		}()

		var count int
		hashQuery := fmt.Sprintf("SELECT count(*) FROM %s WHERE hash = %d", activityTable, int64(summaryHash))
		err = tx.QueryRow(hashQuery).Scan(&count)
		if err != nil {
			return fmt.Errorf("hash existence query: %w", err)
		}

		if count != 0 {
			return fmt.Errorf("summary hash should be unique: found %d existing records", count)
		}

		for _, query := range queries {
			_, err = tx.Exec(query)
			if err != nil {
				return fmt.Errorf("sql insert: %w", err)
			}
		}

		// set up influx client
		influxHost, _ := flags.GetString("influx_host")
		influxToken, _ := flags.GetString("influx_token")
		influxOrg, _ := flags.GetString("influx_org")
		influxBucket, _ := flags.GetString("influx_bucket")

		options := influxdb2.DefaultOptions()
		options.SetPrecision(time.Second)

		client := influxdb2.NewClientWithOptions(influxHost, influxToken, options)
		defer client.Close()
		influxAPI := client.WriteAPIBlocking(influxOrg, influxBucket)

		buf := new(bytes.Buffer)
		err = fitcmd.WriteLineProtocol(buf, data, tags)
		if err != nil {
			return fmt.Errorf("write line protocol: %w", err)
		}

		err = influxAPI.WriteRecord(context.Background(), buf.String())
		if err != nil {
			return fmt.Errorf("write influx records: %w", err)
		}

		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("commit sql: %w", err)
		}
	}

	return nil
}

func buildInsertQueries(activityTable, measurementTable, correlationTable string, summary *fitcmd.Summary, summaryHash int64) ([]string, error) {
	scruGenerator := scru128.NewGenerator()
	activityID, err := scruGenerator.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate activity ID: %w", err)
	}

	tags, err := json.Marshal(summary.Tags)
	if err != nil {
		return nil, fmt.Errorf("marshal json tags: %w", err)
	}

	queries := make([]string, 0, 1+len(summary.Measurements)+len(summary.Correlations))
	queries = append(queries, fmt.Sprintf(
		insertActivityFormat,
		activityTable,
		activityID,
		summaryHash,
		summary.Type,
		summary.StartTime.Format(time.RFC3339),
		summary.EndTime.Format(time.RFC3339),
		summary.MaxDistanceFromStart,
		tags,
	))

	measurementIDs := make(map[string]string)
	for _, m := range summary.Measurements {
		id, err := scruGenerator.Generate()
		if err != nil {
			return nil, fmt.Errorf("measurement generate scru ID: %w", err)
		}

		measurementIDs[m.Name] = id.String()
		queries = append(queries, fmt.Sprintf(
			insertMeasurementFormat,
			measurementTable,
			id,
			activityID,
			m.Name,
			m.Unit,
			m.Maximum,
			m.Minimum,
			m.Median,
			m.Mean,
			m.Variance,
			m.StandardDeviation,
		))
	}

	for _, c := range summary.Correlations {
		measurementA, ok := measurementIDs[c.MeasurementA]
		if !ok {
			return nil, fmt.Errorf("get measurement ID for name %q", c.MeasurementA)
		}
		measurementB, ok := measurementIDs[c.MeasurementB]
		if !ok {
			return nil, fmt.Errorf("get measurement ID for name %q", c.MeasurementA)
		}

		id, err := scruGenerator.Generate()
		if err != nil {
			return nil, fmt.Errorf("measurement generate scru ID: %w", err)
		}

		queries = append(queries, fmt.Sprintf(
			insertCorrelationFormat,
			correlationTable,
			id,
			activityID,
			measurementA,
			measurementB,
			c.Correlation,
		))
	}

	return queries, nil
}
