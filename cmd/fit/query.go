package main

import (
	"encoding/json"
	"fmt"
	"time"

	fitcmd "github.com/subtlepseudonym/fit"

	"github.com/mitchellh/hashstructure"
	"github.com/scru128/go-scru128"
)

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
	hash bigint UNIQUE NOT NULL,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	type varchar(64),
	start_time timestamptz,
	end_time timestamptz,
	max_distance_from_start numeric(64, 32),
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
	greatest(measurement_a, measurement_b),
	least(measurement_a, measurement_b)
);
`

func buildSetupQuery(activityTable, measurementTable, correlationTable string) string {
	return fmt.Sprintf(
		setupQueryFormat,
		activityTable,
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
) ON CONFLICT (hash)
DO UPDATE SET
	type = EXCLUDED.type,
	start_time = EXCLUDED.start_time,
	end_time = EXCLUDED.end_time,
	max_distance_from_start = EXCLUDED.max_distance_from_start,
	tags = EXCLUDED.tags
RETURNING id;
`

func buildActivityQuery(table string, summary *fitcmd.Summary) (string, error) {
	scruGenerator := scru128.NewGenerator()
	activityID, err := scruGenerator.Generate()
	if err != nil {
		return "", fmt.Errorf("generate activity ID: %w", err)
	}

	summaryHash, err := hashstructure.Hash(summary, nil)
	if err != nil {
		return "", fmt.Errorf("hash summary: %w", err)
	}

	tags, err := json.Marshal(summary.Tags)
	if err != nil {
		return "", fmt.Errorf("marshal json tags: %w", err)
	}

	return fmt.Sprintf(
		insertActivityFormat,
		table,
		activityID,
		summaryHash,
		summary.Type,
		summary.StartTime.Format(time.RFC3339),
		summary.EndTime.Format(time.RFC3339),
		summary.MaxDistanceFromStart,
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
) ON CONFLICT (activity_id, correlation_measurement_combination_idx)
DO UPDATE SET
	correlation = EXCLUDED.correlation;
`

func buildQueries(measurementTable, correlationTable, activityID string, summary *fitcmd.Summary) ([]string, error) {
	scruGenerator := scru128.NewGenerator()
	queries := make([]string, 0, len(summary.Measurements)+len(summary.Correlations))

	for _, m := range summary.Measurements {
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

	for _, c := range summary.Correlations {
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