import heart_rate
import sys
import numpy as np

import matplotlib.pyplot as plt
import matplotlib.dates as mdates

from fitparse import FitFile


def clean_data(data) -> list:
    data = np.array(data)
    ts = data[:,0]
    bpm = data[:,1]

    # Filter large breaks in data continuity
    dx = ts[1:] - ts[:-1]
    x = np.insert(ts, np.where(dx>60)[0]+1, np.nan)
    x = np.asarray(x, dtype='datetime64[s]')
    x = list(map(lambda a: a - np.timedelta64(9, 'h'), x))  # UTC to CST
    y = np.insert(bpm, np.where(dx>60)[0]+1, np.nan)

    return [x, y]


records_map = {}
for name in sys.argv[1:]:
    #type = name.split('/')[0]
    f = FitFile(name)
    type = heart_rate.get_file_type(f)
    if type == heart_rate.file_type_activity and heart_rate.is_tracking_activity(f):
        type = heart_rate.file_type_tracking

    record = heart_rate.get_heart_rate(f)
    print(type + " - " + name)

    if type not in records_map:
        records_map[type] = []
    records_map[type].extend(record)

plot_data = {}
for type, records in records_map.items():
    records.sort(key=lambda x: x['timestamp'])
    if type not in plot_data:
        plot_data[type] = []
    for record in records:
        plot_data[type].append([record['timestamp'], record['heart_rate']])

fig, ax = plt.subplots()
for type, data in plot_data.items():
    plot_data[type] = clean_data(data)
    ax.plot(plot_data[type][0], plot_data[type][1], 'o-', markersize=2)

ax.xaxis.set_major_locator(mdates.HourLocator(interval=2))
ax.xaxis.set_minor_locator(mdates.MinuteLocator(interval=30))
ax.xaxis.set_major_formatter(mdates.DateFormatter('%H:%m'))
ax.grid(True)

#plt.plot(x, y)
plt.show()
