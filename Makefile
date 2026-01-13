# .PHONY: clean build test debug generate-protos

# # Ensure GOBIN is in PATH so protoc can find plugins
# export PATH := $(shell go env GOPATH)/bin:$(PATH)

# # Find all proto files recursively, excluding node_modules and build directories
# PROTO_SOURCES := $(shell find . -name "*.proto" -not -path "*/node_modules/*" -not -path "*/build/*")

# # Map protos to their build targets:
# # .../protos/.../file.proto -> .../build/.../file.pb.go
# PROTO_TARGETS := $(shell find . -name "*.proto" -not -path "*/node_modules/*" -not -path "*/build/*" | sed 's|/protos/|/build/|g; s|\.proto$$|.pb.go|g')

# generate-protos: $(PROTO_TARGETS)

# # Template to generate a rule for each proto file
# # $(1): Source file (e.g. core/lights/protos/light.proto)
# # $(2): Target file (e.g. core/lights/build/light.pb.go)
# define PROTO_RULE
# $(2): $(1)
# 	@mkdir -p $$(dir $$@)
# 	@echo "Compiling $(1) -> $(2)"
# 	@protoc \
# 		--proto_path=$(shell echo $(1) | sed 's|/protos/.*|/protos|') \
# 		--proto_path=. \
# 		--go_out=$(shell echo $(2) | sed 's|/build/.*|/build|') --go_opt=paths=source_relative \
# 		--go-grpc_out=$(shell echo $(2) | sed 's|/build/.*|/build|') --go-grpc_opt=paths=source_relative \
# 		$(1)
# endef

# # Evaluate the rule for each source/target pair
# $(foreach src,$(PROTO_SOURCES),$(eval $(call PROTO_RULE,$(src),$(shell echo $(src) | sed 's|/protos/|/build/|g; s|\.proto$$|.pb.go|g'))))


# build: generate-protos
# 	docker-compose up --build

# clean:
# 	rm -rf $(BUILD_DIR)

# debug:
# 	@echo "PROTO_DIR: '$(PROTO_DIR)'"
# 	@echo "BUILD_DIR: '$(BUILD_DIR)'" 
# 	@echo "PROTO_SOURCES: '$(PROTO_SOURCES)'"
# 	@echo "PROTO_TARGETS: '$(PROTO_TARGETS)'"
# 	@echo "MODULE_NAME: '$(MODULE_NAME)'"

.PHONY: all generate-protos build clean debug

# 1. Configurable Variables
# Using the official buf Docker image ensures everyone uses the same version
BUF_VERSION := 1.47.2
DOCKER_BUF := docker run --rm -v "$(shell pwd):/workspace" -w /workspace bufbuild/buf:$(BUF_VERSION)

all: generate-protos build

# 2. Zero-Setup Proto Generation
# This pulls the buf image, updates dependencies (like googleapis), and generates code.
generate-protos:
	@echo "--- Generating stubs with Buf (via Docker) ---"
	@$(DOCKER_BUF) mod update
	@$(DOCKER_BUF) generate

# 3. Standard Build
build: generate-protos
	@echo "--- Starting Docker Compose ---"
	docker-compose up --build

# 4. Clean up generated files
clean:
	@echo "--- Cleaning up ---"
	find . -type d -name "build" -exec rm -rf {} +

# 5. Debugging helper
debug:
	@echo "Buf Docker Command: $(DOCKER_BUF)"
	@echo "Working Directory: $(shell pwd)"
	@$(DOCKER_BUF) --version