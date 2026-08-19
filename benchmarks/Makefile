APP=themis-bench

# Optional overrides for `make bench`.
DATE?=
ENDPOINT?=

BENCH_FLAGS=$(if $(DATE),--date $(DATE))
RUN_FLAGS=$(if $(ENDPOINT),--endpoint $(ENDPOINT))

.PHONY: build run clean fmt vet test check bench

build:
	go build -o bin/$(APP) ./cmd/$(APP)

# Run the full pipeline (run -> evaluate -> validate -> report) for MODEL.
#
#   make bench MODEL=cyberpal20b
#   make bench MODEL=cyberpal20b ENDPOINT=http://remote:11434 DATE=2026-08-19
bench: build
	@test -n "$(MODEL)" || { echo "Usage: make bench MODEL=<model> [ENDPOINT=<url>] [DATE=YYYY-MM-DD]"; exit 1; }
	./bin/$(APP) run $(BENCH_FLAGS) $(RUN_FLAGS) $(MODEL)
	./bin/$(APP) evaluate $(BENCH_FLAGS) $(MODEL)
	./bin/$(APP) validate $(BENCH_FLAGS) $(MODEL)
	./bin/$(APP) report $(BENCH_FLAGS) $(MODEL)

run:
	go run ./cmd/$(APP)

clean:
	rm -rf bin

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

check: fmt vet test
