APP_NAME   := OpenChronoLog
ICON       := icon.png
# Change this to your reverse-DNS bundle ID, or override via:
#   make APP_ID=com.yourorg.openchronolog darwin-all
APP_ID     := org.openchronolog.app
APP_VER    := 1.0.0
FYNE_CROSS := fyne-cross

# Detect if local fyne CLI supports -arch flag
FYNE_HAS_ARCH := $(shell fyne package -h 2>&1 | grep -c -- '-arch')

# Host arch detection (for cgo cross-compile flags)
HOST_ARCH := $(shell uname -m)

# Compiler flags to build specific macOS arches with cgo
AMD64_CC  := clang -arch x86_64
AMD64_CXX := clang++ -arch x86_64
ARM64_CC  := clang -arch arm64
ARM64_CXX := clang++ -arch arm64

BUILD_DIR := build
DIST_DIR  := dist

# Build macOS Intel (amd64)
.PHONY: darwin-amd64
darwin-amd64:
ifeq ($(FYNE_HAS_ARCH),1)
	# New fyne tool path (preferred)
	fyne package -os darwin -arch amd64 -name $(APP_NAME) -icon $(ICON) --appID $(APP_ID) --appVersion $(APP_VER)
	mkdir -p $(DIST_DIR)/darwin-amd64
	rm -rf $(DIST_DIR)/darwin-amd64/$(APP_NAME).app
	mv -f $(APP_NAME).app $(DIST_DIR)/darwin-amd64/ || true
else
	# Fallback: pre-build amd64 exe and package using --executable
	mkdir -p $(BUILD_DIR)/darwin-amd64 $(DIST_DIR)/darwin-amd64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 CC="$(AMD64_CC)" CXX="$(AMD64_CXX)" \
		go build -o $(BUILD_DIR)/darwin-amd64/$(APP_NAME) .
	fyne package --os darwin --executable $(BUILD_DIR)/darwin-amd64/$(APP_NAME) --name $(APP_NAME) --icon $(ICON) --appID $(APP_ID) --appVersion $(APP_VER)
	rm -rf $(DIST_DIR)/darwin-amd64/$(APP_NAME).app
	mv -f $(APP_NAME).app $(DIST_DIR)/darwin-amd64/
endif

# Build macOS Apple Silicon (arm64)
.PHONY: darwin-arm64
darwin-arm64:
ifeq ($(FYNE_HAS_ARCH),1)
	fyne package -os darwin -arch arm64 -name $(APP_NAME) -icon $(ICON) --appID $(APP_ID) --appVersion $(APP_VER)
	mkdir -p $(DIST_DIR)/darwin-arm64
	rm -rf $(DIST_DIR)/darwin-arm64/$(APP_NAME).app
	mv -f $(APP_NAME).app $(DIST_DIR)/darwin-arm64/ || true
else
	mkdir -p $(BUILD_DIR)/darwin-arm64 $(DIST_DIR)/darwin-arm64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 CC="$(ARM64_CC)" CXX="$(ARM64_CXX)" \
		go build -o $(BUILD_DIR)/darwin-arm64/$(APP_NAME) .
	fyne package --os darwin --executable $(BUILD_DIR)/darwin-arm64/$(APP_NAME) --name $(APP_NAME) --icon $(ICON) --appID $(APP_ID) --appVersion $(APP_VER)
	rm -rf $(DIST_DIR)/darwin-arm64/$(APP_NAME).app
	mv -f $(APP_NAME).app $(DIST_DIR)/darwin-arm64/
endif

# Build both macOS targets
.PHONY: darwin-all
darwin-all: darwin-amd64 darwin-arm64

# Build Windows x86-64 (amd64) through fyne-cross.
.PHONY: windows-amd64
windows-amd64:
	$(FYNE_CROSS) windows -arch amd64 -name $(APP_NAME) -app-id $(APP_ID) -app-version $(APP_VER) -icon $(ICON)
	mkdir -p $(DIST_DIR)/windows-amd64
	cp -f fyne-cross/bin/windows-amd64/$(APP_NAME).exe $(DIST_DIR)/windows-amd64/$(APP_NAME).exe
	cp -f fyne-cross/dist/windows-amd64/$(APP_NAME).zip $(DIST_DIR)/windows-amd64/$(APP_NAME).zip

.PHONY: darwin-universal
darwin-universal:
ifeq ($(FYNE_HAS_ARCH),1)
	fyne package -os darwin -arch universal -name $(APP_NAME) -icon $(ICON) --appID $(APP_ID) --appVersion $(APP_VER)
	mkdir -p $(DIST_DIR)/darwin-universal
	mv -f $(APP_NAME).app $(DIST_DIR)/darwin-universal/ || true
else
	# Fallback: build both arches, package once, then replace binary with a lipo-universal
	$(MAKE) darwin-amd64
	$(MAKE) darwin-arm64
	mkdir -p $(DIST_DIR)/darwin-universal
	rm -rf $(DIST_DIR)/darwin-universal/$(APP_NAME).app
	cp -R $(DIST_DIR)/darwin-amd64/$(APP_NAME).app $(DIST_DIR)/darwin-universal/
	lipo -create -output $(DIST_DIR)/darwin-universal/$(APP_NAME).app/Contents/MacOS/$(APP_NAME) \
		$(BUILD_DIR)/darwin-amd64/$(APP_NAME) $(BUILD_DIR)/darwin-arm64/$(APP_NAME)
endif

# Clean fyne-cross artifacts
.PHONY: clean
clean:
	rm -rf fyne-cross/dist fyne-cross/tmp $(BUILD_DIR) $(DIST_DIR)

# Show where outputs go
.PHONY: where
where:
	@echo "macOS:  $(DIST_DIR)/darwin-{amd64,arm64,universal}/$(APP_NAME).app"
	@echo "Windows: $(DIST_DIR)/windows-amd64/$(APP_NAME).{exe,zip}"

.PHONY: doctor
doctor:
	@echo "fyne has -arch: $(FYNE_HAS_ARCH)"
