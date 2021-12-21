DEPS:=cesscrypto.h cesscrypto.pc libcesscrypto.a

all: $(DEPS)
.PHONY: all

# Create a file so that parallel make doesn't call `./install-cesscrypto` for
# each of the deps
$(DEPS): .install-cesscrypto  ;

.install-cesscrypto: rust
	go clean -cache -testcache .
	./install-cesscrypto
	@touch $@

clean:
	go clean -cache -testcache .
	rm -rf $(DEPS) .install-cesscrypto
	rm -f ./runner
	cd rust && cargo clean && cd ..
.PHONY: clean

go-lint: $(DEPS)
	golangci-lint run -v --concurrency 2 --new-from-rev origin/master
.PHONY: go-lint

shellcheck:
	shellcheck install-cesscrypto

lint: shellcheck go-lint

cgo-leakdetect: runner
	valgrind --leak-check=full --show-leak-kinds=definite ./runner
.PHONY: cgo-leakdetect

cgo-gen: $(DEPS)
	go run github.com/xlab/c-for-go --nostamp cesscrypto.yml
.PHONY: cgo-gen

runner: $(DEPS)
	rm -f ./runner
	go build -o ./runner ./cgoleakdetect/
.PHONY: runner
