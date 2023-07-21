run:
	export $(grep -v '^#' .env | xargs) > /dev/null
	go run cmd/main.go
