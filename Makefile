build: gen-pb
	go build -race . 

build-release:
	go build .

test:
	go test $(PACKAGE) 

gen-pb: pb/*.proto
	protoc -I pb/ pb/raft.proto --go_out=plugins=grpc:pb

clean:
	rm -rf pb/raft.pb.go
	rm -rf raft