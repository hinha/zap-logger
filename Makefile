
.PHONY: bench
BENCH ?= .
bench:
	go list ./... | xargs -n1 go test -bench=$(BENCH) -run="^$$" $(BENCH_FLAGS)
