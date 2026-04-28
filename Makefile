BINARY   := gomexec
DIST     := dist
LDFLAGS  := -ldflags="-s -w"
BUILD    := CGO_ENABLED=0 go build $(LDFLAGS)

.PHONY: all build-all clean

all: build-all

build-all: $(DIST)
	GOOS=linux GOARCH=amd64  $(BUILD) -o $(DIST)/$(BINARY)-amd64   .
	GOOS=linux GOARCH=arm64  $(BUILD) -o $(DIST)/$(BINARY)-arm64   .
	GOOS=linux GOARCH=arm    $(BUILD) -o $(DIST)/$(BINARY)-arm     .
	GOOS=linux GOARCH=mips   $(BUILD) -o $(DIST)/$(BINARY)-mips    .
	GOOS=linux GOARCH=mipsle $(BUILD) -o $(DIST)/$(BINARY)-mipsle  .

$(DIST):
	mkdir -p $(DIST)

clean:
	rm -rf $(DIST)
