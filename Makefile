VERSION=0.1.0

COMMIT=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

CURRENT_DIR=$(shell pwd)
BUILD_DIR=$(CURRENT_DIR)/build
REPORT_DIR=$(CURRENT_DIR)/report

BINARY=medi

VET_REPORT=$(REPORT_DIR)/vet.report
LINT_REPORT=$(REPORT_DIR)/lint.report
TEST_REPORT=$(REPORT_DIR)/test.report
COVERAGE_REPORT=$(REPORT_DIR)/coverage.html
COVERAGE_OUT=$(REPORT_DIR)/coverage.out

LDFLAGS = -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.branch=${BRANCH}"

.PHONY: clean vet fmt lint build test dep cover

all: clean vet fmt lint build test

dep:
	dep ensure -v

build:
	cd cmd/medi; go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)

cover:
	-mkdir -p $(REPORT_DIR)
	go test ./... -coverprofile=$(COVERAGE_OUT) 2>&1 | tee ${TEST_REPORT}
	go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_REPORT)
	open $(COVERAGE_REPORT)

test:
	-mkdir -p $(REPORT_DIR)
	go test ./... 2>&1 | tee ${TEST_REPORT}

vet:
	-mkdir -p $(REPORT_DIR)
	go vet $$(go list ./...) 2>&1 | tee $(VET_REPORT)

fmt:
	goimports -w $$(go list -f "{{.Dir}}" ./...)

lint:
	-mkdir -p $(REPORT_DIR)
	golint $$(go list ./...) | sed "s:^$(CURRENT_DIR)/::" | tee $(LINT_REPORT)

clean:
	-rm -rf $(BUILD_DIR)
	-rm -rf $(REPORT_DIR)
