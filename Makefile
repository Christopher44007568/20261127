.PHONY: render test fmt check

render:
	go run ./cmd/render -out out

test:
	go test ./... -count=1

fmt:
	gofmt -w *.go cmd/render/*.go

check: test
