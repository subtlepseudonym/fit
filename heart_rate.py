import json
import sys
import time

from fitparse import FitFile
from datetime import datetime


file_type_monitoring = 'monitoring_b'
file_type_activity = 'activity'
file_type_tracking = 'tracking'
tracking_activity_name = 'All-Day Tracking'


def get_file_type(messages) -> str:
    for msg in messages:
        if msg.name == 'file_id' and msg.get_value('type'):
            return msg.get_value('type')


# is_tracking_activity is used to modify file type in the case
# that the data represents an activity exclusively for tracking
def is_tracking_activity(messages) -> bool:
    for msg in messages:
        if msg.name == 'sport' and msg.get_value('name'):
            return msg.get_value('name') == tracking_activity_name


def get_base_timestamp(file_type, messages) -> datetime:
    if file_type == file_type_activity:
        name = 'device_info'
        field = 'timestamp'
    elif file_type == file_type_monitoring:
        name = 'monitoring_info'
        field = 'local_timestamp'
    else:
        return time.gmtime(0)

    for msg in messages:
        if msg.name == name and msg.get_value(field):
            return msg.get_value('timestamp')


def activity_timestamp(base_ts, msg) -> datetime:
    return msg.get_value('timestamp')


def monitoring_timestamp(base_ts, msg) -> datetime:
    base = np.uintc(time.mktime(base_ts.timetuple()))
    ts16 = np.ushort(msg.get_value('timestamp_16'))
    ts = base + ((ts16 - (base & 0xFFFF)) & 0xFFFF) + 19800
    return datetime.utcfromtimestamp(ts)
    #sum = time.mktime(base_ts.timetuple()) + msg.get_value('timestamp_16')
    #mesgTimestamp += ( timestamp_16 - ( mesgTimestamp & 0xFFFF ) ) & 0xFFFF
    #return datetime.utcfromtimestamp(sum)


def get_heart_rate(messages) -> list:
    file_type = get_file_type(messages)
    base_ts = get_base_timestamp(file_type, messages)

    records = []
    for msg in messages:
        heart_rate = msg.get_value('heart_rate')
        if heart_rate is None or heart_rate <= 0:
            continue

        if file_type == file_type_activity:
            ts = activity_timestamp(base_ts, msg)
        elif file_type == file_type_monitoring:
            ts = monitoring_timestamp(base_ts, msg)

        #timestamp = ts  # datetime
        timestamp = time.mktime(ts.timetuple())  # unix int
        #timestamp = ts.strftime('%Y-%m-%dT%H:%M:%SZ')
        records.append({'timestamp': timestamp, 'heart_rate': msg.get_value('heart_rate')})

    return records
