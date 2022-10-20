DOCKER_BUILD=docker build

DOCKER_RUN=docker run

.PHONY: build
build:
	xk6 build master --with github.com/grafana/xk6-distributed-tracing="${PWD}/../xk6-distributed-tracing"

.PHONY: proto
proto:
	$(DOCKER_RUN) -v ${PWD}/crocospans:/defs namely/protoc-all -f *.proto -l go
	cp -r ${PWD}/crocospans/gen/pb-go/*.pb.go ${PWD}/crocospans