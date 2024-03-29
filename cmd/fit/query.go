package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	fitcmd "github.com/subtlepseudonym/fit"

	"github.com/lib/pq"
	"github.com/mitchellh/hashstructure"
	"github.com/scru128/go-scru128"
)

// scruGenerator ensures that IDs generated by these queries
// are time-sortable
var scruGenerator = scru128.NewGenerator()

const setupQueryFormat = `
CREATE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
	NEW.updated_at = NOW();
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE %s
(
	id varchar(64) PRIMARY KEY,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	start_time timestamptz,
	end_time timestamptz,
	device varchar(64),
	files varchar(64)[],
	removed varchar(64)[],
	errors varchar(128)[],
	log text
);

CREATE TRIGGER set_updated_at
BEFORE UPDATE ON %s
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();

CREATE INDEX ON %s (start_time);

CREATE TABLE %s
(
	id varchar(64) PRIMARY KEY,
	hash bigint UNIQUE NOT NULL,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	import_id varchar(64) NOT NULL REFERENCES %s(id)
		ON DELETE RESTRICT
		ON UPDATE RESTRICT,
	type varchar(64),
	start_time timestamptz,
	end_time timestamptz,
	tags jsonb
);

CREATE TRIGGER set_updated_at
BEFORE UPDATE ON %s
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();

CREATE INDEX ON %s (start_time);

CREATE TABLE %s
(
	id varchar(64) PRIMARY KEY,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	activity_id varchar(64) NOT NULL REFERENCES %s(id)
		ON DELETE RESTRICT
		ON UPDATE RESTRICT,
	name varchar(64) NOT NULL,
	unit varchar(64),
	maximum numeric(64, 32),
	minimum numeric(64, 32),
	median numeric(64, 32),
	mean numeric(64, 32),
	variance numeric(64, 32),
	standard_deviation numeric(64, 32),
	UNIQUE (activity_id, name)
);

CREATE TRIGGER set_updated_at
BEFORE UPDATE ON %s
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();

CREATE TABLE %s
(
	id varchar(64) PRIMARY KEY,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	activity_id varchar(64) NOT NULL REFERENCES %s(id)
		ON DELETE RESTRICT
		ON UPDATE RESTRICT,
	measurement_a varchar(64) NOT NULL,
	measurement_b varchar(64) NOT NULL,
	correlation numeric(32, 30),
	FOREIGN KEY (activity_id, measurement_a)
		REFERENCES %s(activity_id, name),
	FOREIGN KEY (activity_id, measurement_b)
		REFERENCES %s(activity_id, name)
);

CREATE TRIGGER set_updated_at
BEFORE UPDATE ON %s
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();

CREATE UNIQUE INDEX %s_measurement_combination_idx ON %s(
	activity_id,
	GREATEST(measurement_a, measurement_b),
	LEAST(measurement_a, measurement_b)
);
`

func buildSetupQuery(importTable, activityTable, measurementTable, correlationTable string) string {
	return fmt.Sprintf(
		setupQueryFormat,
		importTable,
		importTable,
		importTable,
		activityTable,
		importTable,
		activityTable,
		activityTable,
		measurementTable,
		activityTable,
		measurementTable,
		correlationTable,
		activityTable,
		measurementTable,
		measurementTable,
		correlationTable,
		correlationTable,
		correlationTable,
	)
}

const insertImportFormat = `
INSERT INTO %s
(
	id,
	start_time,
	device
) VALUES (
	$1, $2, $3
);
`

func insertImport(db *sql.DB, table string, start time.Time, device string) (string, error) {
	query := fmt.Sprintf(insertImportFormat, table)
	importID, err := scruGenerator.Generate()
	_, err = db.Exec(query, importID.String(), start.Format(time.RFC3339), device)
	return importID.String(), err
}

const updateImportFormat = `
UPDATE %s SET
	end_time = $1,
	files = $2,
	errors = $3
WHERE id = $4;
`

func updateImport(db *sql.DB, table, importID string, end time.Time, files, errors []string) error {
	query := fmt.Sprintf(updateImportFormat, table)
	_, err := db.Exec(query, end.Format(time.RFC3339), pq.Array(files), pq.Array(errors), importID)
	return err
}

const insertActivityFormat = `
INSERT INTO %s
(
	id,
	hash,
	import_id,
	type,
	start_time,
	end_time,
	tags
) VALUES (
	'%s', %d, '%s', '%s', '%s', '%s', '%s'
) ON CONFLICT (hash)
DO UPDATE SET
	import_id = EXCLUDED.import_id,
	type = EXCLUDED.type,
	start_time = EXCLUDED.start_time,
	end_time = EXCLUDED.end_time,
	tags = EXCLUDED.tags
RETURNING id;
`

func buildActivityQuery(table string, activity *fitcmd.Activity, importID string) (string, error) {
	activityID, err := scruGenerator.Generate()
	if err != nil {
		return "", fmt.Errorf("generate activity ID: %w", err)
	}

	hash, err := hashstructure.Hash(activity, nil)
	if err != nil {
		return "", fmt.Errorf("hash activity: %w", err)
	}

	tags, err := json.Marshal(activity.Tags)
	if err != nil {
		return "", fmt.Errorf("marshal json tags: %w", err)
	}

	return fmt.Sprintf(
		insertActivityFormat,
		table,
		activityID,
		int64(hash),
		importID,
		activity.Type,
		activity.StartTime.Format(time.RFC3339),
		activity.EndTime.Format(time.RFC3339),
		tags,
	), nil
}

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
) ON CONFLICT (activity_id, name)
DO UPDATE SET
	unit = EXCLUDED.unit,
	maximum = EXCLUDED.maximum,
	minimum = EXCLUDED.minimum,
	median = EXCLUDED.median,
	mean = EXCLUDED.mean,
	variance = EXCLUDED.variance,
	standard_deviation = EXCLUDED.standard_deviation;
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
) ON CONFLICT (
	activity_id,
	GREATEST(measurement_a, measurement_b),
	LEAST(measurement_a, measurement_b)
)
DO UPDATE SET
	correlation = EXCLUDED.correlation;
`

func buildQueries(measurementTable, correlationTable, activityID string, activity *fitcmd.Activity) ([]string, error) {
	queries := make([]string, 0, len(activity.Measurements)+len(activity.Correlations))

	for _, m := range activity.Measurements {
		id, err := scruGenerator.Generate()
		if err != nil {
			return nil, fmt.Errorf("generate scru ID: %w", err)
		}

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

	for _, c := range activity.Correlations {
		id, err := scruGenerator.Generate()
		if err != nil {
			return nil, fmt.Errorf("generate scru ID: %w", err)
		}

		queries = append(queries, fmt.Sprintf(
			insertCorrelationFormat,
			correlationTable,
			id,
			activityID,
			c.MeasurementA,
			c.MeasurementB,
			c.Correlation,
		))
	}

	return queries, nil
}
