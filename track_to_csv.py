import argparse
import csv
import json
import os
import sys
from fitparse import FitFile

record_field_name = 'record'
fields = [
    "heart_rate",
    "enhanced_altitude",
    "temperature",
    "timestamp",
]

parser = argparse.ArgumentParser(
        prog='track_to_csv',
        description='Convert tracking fit file to csv'
)
parser.add_argument('file_path', nargs=1, type=str, help='path to file')
args = parser.parse_args()

if len(args.file_path) < 1:
    print('file path required')
    os.exit(1)

f = FitFile(args.file_path[0])

records = []
for msg in f.messages:
    if msg.name != record_field_name:
        continue
    record = []
    for field in fields:
        record.append(msg.get_value(field))
    records.append(record)

with open('out.csv', 'w', newline='') as out:
    out.truncate()
    writer = csv.writer(out, delimiter=',')
    writer.writerow(fields)
    for record in records:
        writer.writerow(record)
