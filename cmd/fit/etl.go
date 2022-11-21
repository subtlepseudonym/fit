package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"time"

	fitcmd "github.com/subtlepseudonym/fit"

	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	fit "github.com/subtlepseudonym/fit-go"
)

func NewETLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "etl",
		Short: "ETL the given file into downstream storage",
		RunE:  etlAll,
	}

	// non-persistent
	flags := cmd.Flags()
	flags.Bool("verbose", false, "Print additional information")
	flags.String("device", DefaultDevice, "Telemetry device name")

	persistent := cmd.PersistentFlags()
	persistent.String("postgres", "", "Postgres DSN")
	persistent.String("postgres_activity_table", "activity", "Postgres table")
	persistent.String("postgres_measurement_table", "measurement", "Postgres table")
	persistent.String("postgres_correlation_table", "correlation", "Postgres table")
	persistent.String("influx_host", "", "InfluxDB DSN")
	persistent.String("influx_token", "", "InfluxDB API token")
	persistent.String("influx_org", "default", "InfluxDB organization")
	persistent.String("influx_bucket", "fit", "InfluxDB bucket")

	cobra.MarkFlagRequired(persistent, "postgres")
	cobra.MarkFlagRequired(persistent, "influx_host")
	cobra.MarkFlagRequired(persistent, "influx_token")

	cmd.AddCommand(NewETLSetupCommand())

	return cmd
}

func etlAll(cmd *cobra.Command, args []string) error {
	// set up postgres client
	flags := cmd.Flags()
	postgresDSN, _ := flags.GetString("postgres")
	db, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		return fmt.Errorf("sql open: %w", err)
	}
	defer db.Close()

	device, err := cmd.Flags().GetString("device")
	if err != nil {
		return fmt.Errorf("device flag: %w", err)
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

	tags := map[string]string{
		"device": device,
	}

	for _, arg := range args {
		err = etl(cmd, db, influxAPI, arg, tags)
		if err != nil {
			return fmt.Errorf("etl: %q: %w", arg, err)
		}
		if verbose, _ := flags.GetBool("verbose"); verbose {
			fmt.Println(path.Base(arg))
		}
	}

	return nil
}

func etl(cmd *cobra.Command, db *sql.DB, influxAPI api.WriteAPIBlocking, filename string, tags map[string]string) (ret error) {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer file.Close()

	data, err := fit.Decode(file)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

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

	flags := cmd.Flags()
	activityTable, _ := flags.GetString("postgres_activity_table")
	measurementTable, _ := flags.GetString("postgres_measurement_table")
	correlationTable, _ := flags.GetString("postgres_correlation_table")

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

	return nil
}
