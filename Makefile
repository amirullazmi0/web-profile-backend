generate:
	go generate ./...

test:
	go test -race ./...

vet:
	go vet ./...

run:
	go run .

dev:
	air

seed:
	go run . -seed

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -o docs
