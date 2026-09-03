APP=themis-bench

# Optional overrides for `make bench`.
DATE?=
ENDPOINT?=
VARIANT?=

BENCH_FLAGS=$(if $(DATE),--date $(DATE))
RUN_FLAGS=$(if $(ENDPOINT),--endpoint $(ENDPOINT)) $(if $(VARIANT),--variant $(VARIANT))

# Variant runs are recorded under MODEL@VARIANT.
BENCH_NAME=$(MODEL)$(if $(VARIANT),@$(VARIANT))

.PHONY: build run clean fmt vet test check bench

build:
	go build -o bin/$(APP) ./cmd/$(APP)

# Run the full pipeline (run -> evaluate -> validate -> report) for MODEL.
#
#   make bench MODEL=cyberpal20b
#   make bench MODEL=cyberpal20b ENDPOINT=http://remote:11434 DATE=2026-08-19
#   make bench MODEL=cyberpal20b VARIANT=json-strict   # prompt A/B run
bench: build
	@test -n "$(MODEL)" || { echo "Usage: make bench MODEL=<model> [ENDPOINT=<url>] [DATE=YYYY-MM-DD] [VARIANT=<name>]"; exit 1; }
	./bin/$(APP) run $(BENCH_FLAGS) $(RUN_FLAGS) $(MODEL)
	./bin/$(APP) evaluate $(BENCH_FLAGS) "$(BENCH_NAME)"
	./bin/$(APP) validate $(BENCH_FLAGS) "$(BENCH_NAME)"
	./bin/$(APP) report $(BENCH_FLAGS) "$(BENCH_NAME)"

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
