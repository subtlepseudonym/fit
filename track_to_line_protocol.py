import argparse
import csv
import datetime
import json
import os
import sys

from fitparse import FitFile
from influx_line_protocol import Metric, MetricCollection

record_field_name = 'record'
metric_name = 'fit_track'
fields = [
    "heart_rate",
    "enhanced_altitude",
    "temperature",
    "timestamp",
]

parser = argparse.ArgumentParser(
        description='Convert tracking fit file to influxdb line protocol'
)
parser.add_argument('--device', dest='device', nargs=1, type=str, help='tracking device', required=True)
parser.add_argument('file_path', nargs=1, type=str, help='path to file')
args = parser.parse_args()

if len(args.file_path) < 1:
    print('file path required')
    os.exit(1)

if len(args.device) < 1:
    print('device flag required')
    os.exit(1)

f = FitFile(args.file_path[0])

metrics = MetricCollection()
for msg in f.messages:
    if msg.name != record_field_name:
        continue

    metric = Metric(metric_name)
    metric.add_tag('device', args.device[0])
    for field in fields:
        if field == 'timestamp':
            field_data = msg.get(field)
            metric.with_timestamp(field_data.raw_value)
        else:
            metric.add_value(field, msg.get_value(field))

    metrics.append(metric)

with open('out.line', 'w') as out:
    out.truncate()
    out.write(metrics.__str__())
