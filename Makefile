GO=go
GOLINT=golint
GLIDE=glide
MISSPELL=misspell

EXE=./dist/bin/gendry

LINT_FLAGS=-set_exit_status
VENDOR_DIR=./vendor

MAIN=./main.go
$GO_SRC=$(wildcard main.go)

all: $(EXE)

$(EXE): $(VENDOR_DIR) $(GO_SRC) $(MAIN)
	$(GO) build -x -v -o $(EXE) $(MAIN)

test: $(VENDOR_DIR)
	$(GO) vet $(MAIN)
	$(GOLINT) $(LINT_FLAGS) $(MAIN)
	$(MISSPELL) -error $(MAIN)
	$(GO) test -v ./...

$(VENDOR_DIR):
	$(GO) get -v -u github.com/golang/lint/golint
	$(GO) get -v -u github.com/client9/misspell/cmd/misspell
	$(GO) get -v -u github.com/Masterminds/glide
	$(GLIDE) install

clean:
	rm -rf $(EXE)
	rm -rf $(VENDOR_DIR)
