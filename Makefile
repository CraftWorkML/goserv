majorVersion = 1
minorVersion = 0
patchVersion = 0
version = $(majorVersion).$(minorVersion).$(patchVersion)

binaryPath = ./target/
registry = registry.com
image = app
tag = deploy-$(version)

update-path:
	export PATH=$PATH:/usr/local/go/bin

update-version:
	sed -i '' "s/^patchVersion =.*/patchVersion = $(shell echo $$(($(patchVersion) + 1)))/" ./Makefile

binary:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.version=$(version)'" -a -o sidecar-injector ./src/cmd/ 
test:
	go test ./...

docker-build:
	docker build -t $(registry)/app:$(tag) . --build-arg binaryVersion=$(version)

docker-push:
	docker push $(registry)/app:$(tag)

deploy:
	kubectl apply -f ./manifests/serviceaccount.yaml -n awhs && \
	kubectl apply -f ./manifests/clusterrole.yaml -n awhs && \
	kubectl apply -f ./manifests/clusterrolebinding.yaml -n awhs && \
	kubectl apply -f ./manifests/service.yaml -n awhs && \
	kubectl apply -f ./manifests/deployment.yaml -n awhs
	
	
	
