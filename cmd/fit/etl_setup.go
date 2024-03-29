package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/influxdata/influxdb-client-go/v2"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

func NewETLSetupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup up databases for ETL command",
		RunE:  etlSetup,
	}

	cmd.Flags().Bool("no-influx", false, "Exclude influxdb from setup")

	return cmd
}

func etlSetup(cmd *cobra.Command, args []string) (ret error) {
	flags := cmd.Flags()
	postgresDSN, _ := flags.GetString("postgres")
	importTable, _ := flags.GetString("postgres-import-table")
	activityTable, _ := flags.GetString("postgres-activity-table")
	measurementTable, _ := flags.GetString("postgres-measurement-table")
	correlationTable, _ := flags.GetString("postgres-correlation-table")

	db, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		return fmt.Errorf("sql open: %w", err)
	}
	defer db.Close()

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

	setupQuery := buildSetupQuery(importTable, activityTable, measurementTable, correlationTable)
	_, err = tx.Exec(setupQuery)
	if err != nil {
		return fmt.Errorf("setup query: %w", err)
	}

	if noInflux, _ := flags.GetBool("no-influx"); !noInflux {
		err = setupInflux(cmd, args)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit sql: %w", err)
	}

	return nil
}

func setupInflux(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()
	influxHost, _ := flags.GetString("influx-host")
	influxToken, _ := flags.GetString("influx-token")
	influxOrg, _ := flags.GetString("influx-org")
	influxBucket, _ := flags.GetString("influx-bucket")

	options := influxdb2.DefaultOptions()
	options.SetPrecision(time.Second)

	client := influxdb2.NewClientWithOptions(influxHost, influxToken, options)
	defer client.Close()

	org, err := client.OrganizationsAPI().FindOrganizationByName(context.Background(), influxOrg)
	if err != nil {
		return fmt.Errorf("influx get org: %w", err)
	}

	bucket, err := client.BucketsAPI().FindBucketByName(context.Background(), influxBucket)
	if err != nil {
		return fmt.Errorf("influx find bucket: %w", err)
	}

	if bucket == nil {
		_, err = client.BucketsAPI().CreateBucketWithName(context.Background(), org, influxBucket)
		if err != nil {
			return fmt.Errorf("influx create bucket: %w", err)
		}
	}

	return nil
}
