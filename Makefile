
OUT_DIR = _output

export GOBIN

# Example:
#   make
#   make all
all build:
	mkdir -p ${OUT_DIR}/bin
	go build -o ${OUT_DIR}/bin/watcher cmd/watcher/watcher.go

clean:
	rm -rf ${OUT_DIR}
.PHONY: all build
