BUILD=$$( \
	if command -v vtag 1>/dev/null 2>&1; then \
		vtag; \
	else \
		printf \
			'0.0.1-unknown+%s' \
			"$$(git rev-list -n1 HEAD | head -c7)"; \
	fi \
)
LDFLAGS=--ldflags "-X main.Version=${BUILD}"

default: build

bin: clean build

build: build-python build-go

build-go: tidy
	mkdir -p dist/
	go build ${LDFLAGS} -o dist/fit -v ./cmd/fit

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
