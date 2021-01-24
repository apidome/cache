.PHONY: build
test:
	go clean -testcache
	go test -v -timeout 0 ./...

# Creates cache.test
.PHONY: build-tests
build-tests:
	go test -c ./...
