VERSION=0
DOCKER_NW=dockerraft
CLUSTER_SZ=3

build:
	go build -race . 

build-release:
	go build .

test-all:
	go test -v -race ./...

test:
	go test -timeout=3s $(PACKAGE) 

gen-pb: pb/*.proto
	protoc -I pb/ pb/raft.proto --go_out=plugins=grpc:pb

setup-local-cluster:
	./scripts/cluster_manager.rb --generate --dry-run

run-local-cluster: build
	./scripts/cluster_manager.rb --generate --launch \
		--cluster-size=$(CLUSTER_SZ)

run-docker-cluster:
	./scripts/cluster_manager.rb --generate --launch \
		--cluster-size=$(CLUSTER_SZ) \
		--docker-mode --docker-network-name=$(DOCKER_NW)

destroy-docker-cluster:
	./scripts/cluster_manager.rb --docker-mode --docker-destroy \
		--cluster-size=$(CLUSTER_SZ)

build-docker-container: gen-pb
	docker build -t raft:local .

push-to-registry: build-docker-container
	docker login
	docker tag raft:local $(REPO)/raft:$(VERSION)
	docker push $(REPO)/raft:$(VERSION)

clean:
	rm -rf raft
	rm -rf local-cluster