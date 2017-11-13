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

GO_SRC=$(wildcard ./*.go ./gendry/*.go)
MODEL_SRC=$(filter-out %.marlow.go, $(wildcard ./gendry/models/*.go))
MODEL_OBJS=$(patsubst %.go,%.marlow.go,$(MODEL_SRC))

all: $(EXE)

$(EXE): $(VENDOR_DIR) $(GO_SRC) $(MAIN) $(MODEL_OBJS)
	$(GO) build $(BUILD_FLAGS) -o $(EXE) $(MAIN)

$(MODEL_OBJS): $(MODEL_SRC)
	$(GO) generate ./gendry/models/...

test: $(VENDOR_DIR)
	$(GO) vet $(MAIN)
	$(GOLINT) $(LINT_FLAGS) $(MAIN)
	$(MISSPELL) -error $(MAIN)
	$(GOLINT) $(LINT_FLAGS) ./gendry/...
	$(GO) vet ./gendry/...
	$(GO) test $(TEST_FLAGS) ./gendry/...

$(VENDOR_DIR):
	$(GO) get -v -u github.com/golang/lint/golint
	$(GO) get -v -u github.com/client9/misspell/cmd/misspell
	$(GO) get -v -u github.com/Masterminds/glide
	$(GO) get -v -u github.com/dadleyy/marlow/marlowc
	$(GLIDE) install

clean:
	rm -rf $(EXE)
	rm -rf $(VENDOR_DIR)
	rm -rf $(MODEL_OBJS)
