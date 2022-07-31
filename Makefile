default: build

build:
	pyinstaller --onefile fit_type.py
	pyinstaller --onefile track_to_line.py

clean:
	rm -r build/ dist/ __pycache__/ *.spec out.csv out.line

.PHONY: all build clean
