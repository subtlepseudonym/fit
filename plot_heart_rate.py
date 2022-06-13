import heart_rate
import sys
import numpy as np

import matplotlib.pyplot as plt
import matplotlib.dates as mdates

from fitparse import FitFile

hrr = []
for name in sys.argv[1:]:
    hrr += heart_rate.get_heart_rate(FitFile(name))

hrr.sort(key=lambda x: x['timestamp'])

data = []
for obj in hrr:
    data.append([obj['timestamp'], obj['heart_rate']])

data = np.array(data)
ts = data[:,0]
bpm = data[:,1]

# Filter large breaks in data continuity
dx = ts[1:] - ts[:-1]
x = np.insert(ts, np.where(dx>60)[0]+1, np.nan)
x = np.asarray(x, dtype='datetime64[s]')
x = list(map(lambda a: a - np.timedelta64(8, 'h'), x))  # UTC to EST
y = np.insert(bpm, np.where(dx>60)[0]+1, np.nan)

fig, ax = plt.subplots()
ax.plot(x, y, 'bo-', markersize=2)
ax.xaxis.set_major_locator(mdates.HourLocator(interval=1))
ax.xaxis.set_minor_locator(mdates.MinuteLocator(interval=15))
ax.xaxis.set_major_formatter(mdates.DateFormatter('%H:%m'))
ax.grid(True)

print(x[0])
print(x[-1])

#plt.plot(x, y)
plt.show()
