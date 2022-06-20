import json
import sys
import time

from fitparse import FitFile
from datetime import datetime


file_type_unknown = 'unknown'
file_type_monitoring = 'monitoring_b'
file_type_activity = 'activity'

activity_type_bike = 'Bike'
activity_type_cooldown = 'Cooldown'
activity_type_strength = 'Strength'
activity_type_tracking = 'All-Day Tracking'
activity_type_walk = 'Walk'


data_type_dict = {
    file_type_monitoring:   'monitor',
    activity_type_bike:     'cycle',
    activity_type_cooldown: 'cooldown',
    activity_type_tracking: 'track',
    activity_type_strength: 'lift',
    activity_type_walk:     'walk'
}


def get_file_type(messages) -> str:
    for msg in messages:
        if msg.name == 'file_id' and msg.get_value('type'):
            return msg.get_value('type')

    return file_type_unknown


def get_activity_type(messages) -> str:
    for msg in messages:
        if msg.name == 'sport' and msg.get_value('name'):
            return msg.get_value('name')

    return file_type_unknown


# get_data_type provides a friendly name for the type of fit data
def get_data_type(messages) -> str:
    file_type = get_file_type(messages)
    if file_type in data_type_dict:
        return data_type_dict[file_type]

    activity_type = get_activity_type(messages)
    if activity_type in data_type_dict:
        return data_type_dict[activity_type]

    return file_type_unknown


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
