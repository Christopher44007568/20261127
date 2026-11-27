 .PHONY: render test vet fmt check

render:
	go run .

test:
	go test ./... -count=1

vet:
	go vet ./...

fmt:
	gofmt -w main.go furnace/*.go

check: fmt vet test
	go list -m all
