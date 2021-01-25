.PHONY: build
test:
	go clean -testcache
	go test -ginkgo.v -timeout 5m ./...

# Creates cache.test
.PHONY: build-tests
build-tests:
	go test -c ./...
