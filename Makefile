default: build

build:
	pyinstaller --onefile fit_type.py

clean:
	rm -r build/ dist/ __pycache__/

.PHONY: all build clean
