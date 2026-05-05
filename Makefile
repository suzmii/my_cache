.PHONY: test bench bench-lru bench-http bench-httptest bench-all

COUNT ?= 5

test:
	go test -v -count=1 ./...

# ---- benchmarks ----

bench:
	go test -bench=. -benchmem -count=$(COUNT) -run='^$$' ./...

bench-lru:
	go test -bench='BenchmarkLRU' -benchmem -count=$(COUNT) -run='^$$' ./...

bench-http:
	go test -bench='BenchmarkHandler' -benchmem -count=$(COUNT) -run='^$$' ./...

bench-httptest:
	go test -bench='BenchmarkHttptestOverhead' -benchmem -count=$(COUNT) -run='^$$' ./...

bench-all:
	@echo "=== LRU ==="
	go test -bench='BenchmarkLRU' -benchmem -count=$(COUNT) -run='^$$' ./...
	@echo ""
	@echo "=== HTTP ==="
	go test -bench='BenchmarkHandler' -benchmem -count=$(COUNT) -run='^$$' ./...
	@echo ""
	@echo "=== httptest overhead ==="
	go test -bench='BenchmarkHttptestOverhead' -benchmem -count=$(COUNT) -run='^$$' ./...
