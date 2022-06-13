import json
import sys
import numpy as np

from fitparse import FitFile
from datetime import datetime
from time import mktime


def get_file_type(messages) -> str:
    for msg in messages:
        if msg.name == 'file_id' and msg.get('type'):
            return msg.get_value('type')


def get_base_timestamp(messages) -> datetime:
    for msg in messages:
        if msg.name == 'device_info' and msg.get('timestamp'):
            return msg.get_value('timestamp')


def activity_timestamp(base_ts, msg) -> datetime:
    return msg.get_value('timestamp')


def monitoring_timestamp(base_ts, msg) -> datetime:
    base = np.uintc(mktime(base_ts.timetuple()))
    ts16 = np.ushort(msg.get_value('timestamp_16'))
    ts = base + ((ts16 - (base & 0xFFFF)) & 0xFFFF) + 19800
    return datetime.utcfromtimestamp(ts)
    #sum = mktime(base_ts.timetuple()) + msg.get_value('timestamp_16')
    #mesgTimestamp += ( timestamp_16 - ( mesgTimestamp & 0xFFFF ) ) & 0xFFFF
    #return datetime.utcfromtimestamp(sum)


def get_heart_rate(messages) -> list:
    file_type = get_file_type(messages)
    base_ts = get_base_timestamp(messages)

    records = []
    for msg in messages:
        if msg.get('heart_rate') is None:
            continue

        heart_rate = msg.get_value('heart_rate')
        if heart_rate <= 0:
            continue

        if file_type == 'activity':
            ts = activity_timestamp(base_ts, msg)
        elif file_type == 'monitoring_b':
            ts = monitoring_timestamp(base_ts, msg)

        #timestamp = ts  # datetime
        timestamp = mktime(ts.timetuple())  # unix int
        #timestamp = ts.strftime('%Y-%m-%dT%H:%M:%SZ')
        records.append({'timestamp': timestamp, 'heart_rate': msg.get_value('heart_rate')})

    return records
