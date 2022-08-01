default: build

bin: build clean

build:
	mkdir --parents dist/
	pyinstaller --onefile fit_type.py
	pyinstaller --onefile track_to_line.py
	go build -o dist/fit main.go

clean:
	rm --recursive --force \
		*.spec \
		__pycache__/ \
		build/ \
		out.csv \
		out.line
	go clean

format fmt:
	gofmt -l -w .

vendor:
	go mod tidy
	go mod vendor

.PHONY: all build clean vendor
