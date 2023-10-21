include .env
export

run:
	go run cmd/main.go

docker_build:
	@read -p "enter tag: " tag; \
	docker build -t quibbble:$$tag -t quibbble:latest -f build/Dockerfile .

docker_run:
	docker run -d --name quibbble -p 8080:8080 --init -m 750m --cpus=1.5 \
		-e QUIBBBLE_DATASTORE_COCKROACH_ENABLED=${QUIBBBLE_DATASTORE_COCKROACH_ENABLED} \
		-e QUIBBBLE_DATASTORE_COCKROACH_HOST="${QUIBBBLE_DATASTORE_COCKROACH_HOST}" \
		-e QUIBBBLE_DATASTORE_COCKROACH_USERNAME="${QUIBBBLE_DATASTORE_COCKROACH_USERNAME}" \
		-e QUIBBBLE_DATASTORE_COCKROACH_PASSWORD="${QUIBBBLE_DATASTORE_COCKROACH_PASSWORD}" \
		-e QUIBBBLE_DATASTORE_COCKROACH_DATABASE="${QUIBBBLE_DATASTORE_COCKROACH_DATABASE}" \
		-e QUIBBBLE_DATASTORE_COCKROACH_SSLMODE="${QUIBBBLE_DATASTORE_COCKROACH_SSLMODE}" \
		quibbble:latest

docker_stop:
	docker stop quibbble

docker_rm:
	docker rm quibbble
