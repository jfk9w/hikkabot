MODULE := $(shell head -1 go.mod | cut -d ' ' -f2)

GOIMPORTS := $(shell go env GOPATH)/bin/goimports

$(GOIMPORTS):
	go install golang.org/x/tools/cmd/goimports@latest

fmt: $(GOIMPORTS)
	$(GOIMPORTS) -local $(MODULE) -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

gen: $(OGEN)
	go generate ./...

test: gen
	go test -v ./...

bin/%: gen $(wildcard ./internal/**/*) $(wildcard ./cmd$@/**/*)
	go build -buildvcs=false -ldflags "-X main.GitCommit=${VERSION}" -o $@ -v ./$(subst bin,cmd,$@)

bin: $(subst ./cmd,bin,$(wildcard ./cmd/*))

%/schema.yaml: bin/hikkabot
	mkdir -p $(dir $@) && ./$^ --config.schema=yml > $@

%/defaults.yaml: bin/hikkabot
	mkdir -p $(dir $@) && ./$^ --config.values=yml > $@

config: config/schema.yaml config/defaults.yaml

install: bin
	cp bin/* /usr/local/bin/

uninstall:
	rm -f $(subst ./cmd,/usr/local/bin,$(wildcard ./cmd/*))

clean:
	rm -rf bin/*
