import argparse
import heart_rate
import os
import sys
from fitparse import FitFile


parser = argparse.ArgumentParser(
        prog='fit_type',
        description='Get fit type data'
)
parser.add_argument('--file', action='store_true', help='print fit file type')
parser.add_argument('--activity', action='store_true', help='print fit activity type')
parser.add_argument('--data', action='store_true', help='print fit data type')
parser.add_argument('file_path', nargs=1, type=str, help='path to file')
args = parser.parse_args()

if len(args.file_path) < 1:
    print('file path required')
    os.exit(1)

f = FitFile(args.file_path[0])

file_type = heart_rate.get_file_type(f)
activity_type = heart_rate.get_activity_type(f)
data_type = heart_rate.get_data_type(f)

if args.file:
    print(file_type)
if args.activity:
    print(activity_type)
if args.data:
    print(data_type)
