include .env

run:
	go run cmd/main.go

docker:
	@read -p "enter tag: " tag; \
	docker build -t quibbble:$$tag -t quibbble:latest -f build/Dockerfile .
