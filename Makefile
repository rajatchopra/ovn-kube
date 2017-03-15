
OUT_DIR = _output
export OUT_DIR

# Example:
#   make
#   make all
all build:
	hack/build-go.sh cmd/ovnkube/ovnkube.go

clean:
	rm -rf ${OUT_DIR}
.PHONY: all build
