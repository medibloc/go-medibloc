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

COLOR="\033[32;1m"
ENDCOLOR="\033[0m"

.PHONY: clean vet fmt lint build test dep cover test-slow

all: fmt build vet lint test-slow

dep:
	dep ensure -v

build:
	@echo $(COLOR)[BUILD] $(BUILD_DIR)/$(BINARY)$(ENDCOLOR)
	@cd cmd/medi; go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)

cover:
	@echo $(COLOR)[COVER] $(TEST_REPORT)$(ENDCOLOR)
	@-mkdir -p $(REPORT_DIR)
	@go test ./... -coverpkg=./... -coverprofile=$(COVERAGE_OUT) 2>&1 | tee ${TEST_REPORT}
	@go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_REPORT)
	@open $(COVERAGE_REPORT)

test:
	@echo $(COLOR)[TEST] $(TEST_REPORT)$(ENDCOLOR)
	@-mkdir -p $(REPORT_DIR)
	@go test ./... 2>&1 | tee ${TEST_REPORT} | grep -v "^?"

test-slow:
	@echo $(COLOR)[TEST] $(TEST_REPORT)$(ENDCOLOR)
	@-mkdir -p $(REPORT_DIR)
	@go test -race ./... 2>&1 | tee ${TEST_REPORT} | grep -v "^?"

vet:
	@echo $(COLOR)[VET] $(VET_REPORT)$(ENDCOLOR)
	@-mkdir -p $(REPORT_DIR)
	@go vet $$(go list ./...) 2>&1 | tee $(VET_REPORT)

fmt:
	@echo $(COLOR)[FMT] ./...$(ENDCOLOR)
	@goimports -w $$(go list -f "{{.Dir}}" ./...)

lint:
	@echo $(COLOR)[LINT] $(LINT_REPORT)$(ENDCOLOR)
	@-mkdir -p $(REPORT_DIR)
	@golint $$(go list ./...) | sed "s:^$(CURRENT_DIR)/::" | tee $(LINT_REPORT)

clean:
	-rm -rf $(BUILD_DIR)
	-rm -rf $(REPORT_DIR)
	@go clean -testcache
