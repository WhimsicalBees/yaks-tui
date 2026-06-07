.PHONY: build test lint run demo
build:
	go build -o bin/yaks-tui .
test:
	go test ./...
lint:
	go vet ./...
	staticcheck ./... || true
run:
	go run .
# Regenerate the README demo GIF. Requires vhs (brew install vhs) and the
# yaks-tui binary on PATH (so the tape can launch it). VHS drives a headless
# Chrome via go-rod; on macOS we point it at the system Chrome so rod doesn't
# try (and fail) to download its own.
demo: build
	PATH="$(PWD)/bin:$$PATH" \
	ROD_BROWSER_BIN="$${ROD_BROWSER_BIN:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}" \
	vhs docs/demo.tape
