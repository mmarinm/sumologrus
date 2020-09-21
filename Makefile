ifndef CIRCLE_ARTIFACTS
CIRCLE_ARTIFACTS=tmp
endif

bootstrap:
	.buildscripts/bootstrap.sh

dependencies:
	@go get -v -t ./...

vet:
	@go vet ./...

test: vet
	@mkdir -p ${CIRCLE_ARTIFACTS}
	@cd .buildscripts; ./atomic_cover.sh
	@go test -race -coverprofile=${CIRCLE_ARTIFACTS}/cover.out .
	@go tool cover -func ${CIRCLE_ARTIFACTS}/cover.out -o ${CIRCLE_ARTIFACTS}/cover.txt
	@go tool cover -html ${CIRCLE_ARTIFACTS}/cover.out -o ${CIRCLE_ARTIFACTS}/cover.html

build: test
	@go build ./...


ci: dependencies test

.PHONY: bootstrap dependencies vet test ci