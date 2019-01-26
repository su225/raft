build: gen-pb
	go build -race . 

build-release:
	go build .

test-all:
	go test -timeout=3s ./node/cluster
	go test -timeout=3s ./node/log

test:
	go test -timeout=3s $(PACKAGE) 

gen-pb: pb/*.proto
	protoc -I pb/ pb/raft.proto --go_out=plugins=grpc:pb

setup-local-cluster:
	./scripts/setup_cluster_dir.rb

clean:
	rm -rf pb/raft.pb.go
	rm -rf raft
	rm -rf cluster