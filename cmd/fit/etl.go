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
	persistent.String("postgres-import-table", "import", "Table name for import run information")
	persistent.String("postgres-activity-table", "activity", "Table name for activity records")
	persistent.String("postgres-measurement-table", "measurement", "Table name for per-activity measurement records")
	persistent.String("postgres-correlation-table", "correlation", "Table for measurement correlation records")
	persistent.String("influx-host", "", "InfluxDB DSN")
	persistent.String("influx-token", "", "InfluxDB API token")
	persistent.String("influx-org", "default", "InfluxDB organization")
	persistent.String("influx-bucket", "fit", "InfluxDB bucket")

	cmd.MarkPersistentFlagRequired("postgres")
	cmd.MarkFlagRequired("influx-host")
	cmd.MarkFlagRequired("influx-token")

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
	influxHost, _ := flags.GetString("influx-host")
	influxToken, _ := flags.GetString("influx-token")
	influxOrg, _ := flags.GetString("influx-org")
	influxBucket, _ := flags.GetString("influx-bucket")

	options := influxdb2.DefaultOptions()
	options.SetPrecision(time.Second)

	client := influxdb2.NewClientWithOptions(influxHost, influxToken, options)
	defer client.Close()
	influxAPI := client.WriteAPIBlocking(influxOrg, influxBucket)

	tags := map[string]string{
		"device": device,
	}

	if ignore, _ := cmd.Flags().GetBool("ignore-file-checksum"); ignore {
		tags["ignore-file-checksum"] = "true"
	}

	importTable, _ := flags.GetString("postgres-import-table")
	importID, err := insertImport(db, importTable, time.Now(), device)
	if err != nil {
		return fmt.Errorf("insert import record: %w", err)
	}

	var files []string
	var errors []string
	verbose, _ := flags.GetBool("verbose")
	for _, arg := range args {
		files = append(files, path.Base(arg))
		err = etl(cmd, db, influxAPI, arg, importID, tags)
		if err != nil {
			errors = append(errors, fmt.Sprintf("etl: %s: %s", arg, err))
		}
		if verbose {
			fmt.Println(path.Base(arg))
		}
	}

	err = updateImport(db, importTable, importID, time.Now(), files, errors)
	if err != nil {
		return fmt.Errorf("update import record: %s: %w", importID, err)
	}
	if verbose {
		fmt.Println("import ID:", importID)
	}

	return nil
}

func etl(cmd *cobra.Command, db *sql.DB, influxAPI api.WriteAPIBlocking, filename, importID string, tags map[string]string) (ret error) {
	file, err := os.Open(filename)
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

	activity, err := fitcmd.Summarize(data, DefaultMeasurements, DefaultCorrelates, tags)
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
	activityTable, _ := flags.GetString("postgres-activity-table")
	measurementTable, _ := flags.GetString("postgres-measurement-table")
	correlationTable, _ := flags.GetString("postgres-correlation-table")

	activityQuery, err := buildActivityQuery(activityTable, activity, importID)
	if err != nil {
		return fmt.Errorf("build activity query: %w", err)
	}

	var activityID string
	err = tx.QueryRow(activityQuery).Scan(&activityID)
	if err != nil {
		return fmt.Errorf("insert activity: %w", err)
	}

	queries, err := buildQueries(measurementTable, correlationTable, activityID, activity)
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
