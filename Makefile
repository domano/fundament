SWIFT_SHIM_DIR := swift/FundamentShim
SWIFT_MODULE_CACHE := $(CURDIR)/.swift-module-cache
SWIFT_BUILD_CACHE := $(CURDIR)/.swift-build-cache

.PHONY: all swift swift-debug go examples clean

all: swift go

swift:
	mkdir -p $(SWIFT_MODULE_CACHE) $(SWIFT_BUILD_CACHE)
	MACOSX_DEPLOYMENT_TARGET=15.0 \
	SWIFTPM_ENABLE_SANDBOX=0 \
	CLANG_MODULE_CACHE_PATH=$(SWIFT_MODULE_CACHE) \
	SWIFT_MODULECACHE_PATH=$(SWIFT_MODULE_CACHE) \
	swift build --package-path $(SWIFT_SHIM_DIR) -c release --cache-path $(SWIFT_BUILD_CACHE) --disable-sandbox

swift-debug:
	mkdir -p $(SWIFT_MODULE_CACHE) $(SWIFT_BUILD_CACHE)
	MACOSX_DEPLOYMENT_TARGET=15.0 \
	SWIFTPM_ENABLE_SANDBOX=0 \
	CLANG_MODULE_CACHE_PATH=$(SWIFT_MODULE_CACHE) \
	SWIFT_MODULECACHE_PATH=$(SWIFT_MODULE_CACHE) \
	swift build --package-path $(SWIFT_SHIM_DIR) --cache-path $(SWIFT_BUILD_CACHE) --disable-sandbox

go:
	go build ./...

examples: swift
	go build ./examples/simple
	go build ./examples/structured
	go build ./examples/streaming

clean:
	swift package --package-path $(SWIFT_SHIM_DIR) reset
	rm -rf $(SWIFT_SHIM_DIR)/.build
	rm -rf $(SWIFT_MODULE_CACHE) $(SWIFT_BUILD_CACHE)
