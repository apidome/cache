FROM scratch

WORKDIR /
COPY tests/ tests
COPY vendor/ vendor
COPY cache.go cache.go
COPY go.mod go.mod
COPY go.sum go.sum