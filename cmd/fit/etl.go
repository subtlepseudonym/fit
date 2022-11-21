package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	fitcmd "github.com/subtlepseudonym/fit"

	"github.com/influxdata/influxdb-client-go/v2"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	fit "github.com/subtlepseudonym/fit-go"
)

func NewETLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "etl",
		Short: "ETL the given file into downstream storage",
		RunE:  etl,
	}

	// non-persistent
	cmd.Flags().String("device", DefaultDevice, "Telemetry device name")

	flags := cmd.PersistentFlags()
	flags.String("postgres", "", "Postgres DSN")
	flags.String("postgres_activity_table", "activity", "Postgres table")
	flags.String("postgres_measurement_table", "measurement", "Postgres table")
	flags.String("postgres_correlation_table", "correlation", "Postgres table")
	flags.String("influx_host", "", "InfluxDB DSN")
	flags.String("influx_token", "", "InfluxDB API token")
	flags.String("influx_org", "default", "InfluxDB organization")
	flags.String("influx_bucket", "fit", "InfluxDB bucket")

	cobra.MarkFlagRequired(flags, "postgres")
	cobra.MarkFlagRequired(flags, "influx_host")
	cobra.MarkFlagRequired(flags, "influx_token")

	cmd.AddCommand(NewETLSetupCommand())

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

		summary, err := fitcmd.Summarize(data, DefaultCorrelates, tags)
		if err != nil {
			return fmt.Errorf("summarize: %w", err)
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

		activityQuery, err := buildActivityQuery(activityTable, summary)
		if err != nil {
			return fmt.Errorf("build activity query: %w", err)
		}

		var activityID string
		err = tx.QueryRow(activityQuery).Scan(&activityID)
		if err != nil {
			return fmt.Errorf("insert activity: %w", err)
		}

		queries, err := buildQueries(measurementTable, correlationTable, activityID, summary)
		if err != nil {
			return fmt.Errorf("build measurement and correlation queries: %w", err)
		}

		for _, query := range queries {
			_, err = tx.Exec(query)
			if err != nil {
				return fmt.Errorf("insert query: %w", err)
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
