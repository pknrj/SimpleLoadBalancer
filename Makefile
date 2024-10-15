build:
	go build -o bin/LoadServer cmd/loadbalance-server/main.go

run:build
	./bin/LoadServer
