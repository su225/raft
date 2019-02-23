VERSION=0
DOCKER_NW=dockerraft
CLUSTER_SZ=3

RAFT_HEADLESS_SVC=raft-headless-svc.yaml
K8S_DEPLOY_DESCRIPTOR=raft-k8s-deploy.yaml

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

k8s-deploy:
	kubectl create -f $(RAFT_HEADLESS_SVC)
	kubectl create -f $(K8S_DEPLOY_DESCRIPTOR)

k8s-undeploy:
	kubectl delete -f $(K8S_DEPLOY_DESCRIPTOR)
	kubectl delete -f $(RAFT_HEADLESS_SVC)

clean:
	rm -rf raft
	rm -rf local-cluster