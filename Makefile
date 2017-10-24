GO=go
GOLINT=golint
GLIDE=glide
MISSPELL=misspell

EXE=./dist/bin/gendry

LDFLAGS="-s -w"
BUILD_FLAGS=-x -v -ldflags $(LDFLAGS)
TEST_FLAGS=-v -coverprofile=coverage.txt -covermode=atomic
LINT_FLAGS=-set_exit_status
VENDOR_DIR=./vendor

MAIN=./main.go
GO_SRC=$(wildcard ./*.go)

all: $(EXE)

$(EXE): $(VENDOR_DIR) $(GO_SRC) $(MAIN)
	$(GO) build $(BUILD_FLAGS) -o $(EXE) $(MAIN)

test: $(VENDOR_DIR)
	$(GO) vet $(MAIN)
	$(GOLINT) $(LINT_FLAGS) $(MAIN)
	$(MISSPELL) -error $(MAIN)
	$(GO) test $(TEST_FLAGS) ./...

$(VENDOR_DIR):
	$(GO) get -v -u github.com/golang/lint/golint
	$(GO) get -v -u github.com/client9/misspell/cmd/misspell
	$(GO) get -v -u github.com/Masterminds/glide
	$(GLIDE) install

clean:
	rm -rf $(EXE)
	rm -rf $(VENDOR_DIR)
