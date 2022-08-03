default: build

bin: build-all clean

build: build-python build-go

build-go: tidy
	mkdir --parents dist/
	go build -o dist/fit main.go

build-python:
	pyinstaller --onefile src/fit_type.py
	pyinstaller --onefile src/track_to_line.py

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

tidy:
	go mod tidy

vendor: tidy
	go mod vendor

.PHONY: all bin build build-go build-python clean tidy vendor
