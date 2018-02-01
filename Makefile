VERSION="$(shell cat DRAUPNIR_VERSION)"
BUILD_COMMAND=go build -ldflags "-X github.com/gocardless/draupnir/version.Version=$(VERSION)"

.PHONY: build client clean test test-integration dump-schema publish-circleci-dockerfile

build:
	$(BUILD_COMMAND) -o draupnir *.go

migrate:
	vendor/bin/sql-migrate up

dump-schema:
	pg_dump -sxOf structure.sql draupnir

test:
	go test ./...
	go vet ./...

test-integration:
	bundle exec rspec

build-production: test
	GOOS=linux GOARCH=amd64 $(BUILD_COMMAND) -o draupnir.linux_amd64 *.go

client: test
	GOOS=darwin GOARCH=amd64 $(BUILD_COMMAND) -o draupnir-client cli/*.go

deb: build-production
	fpm -f -s dir -t $@ -n draupnir -v $(VERSION) \
		--description "Databases on demand" \
		--maintainer "GoCardless Engineering <engineering@gocardless.com>" \
		draupnir.linux_amd64=/usr/local/bin/draupnir \
		cmd/draupnir-finalise-image=/usr/local/bin/draupnir-finalise-image \
		cmd/draupnir-create-instance=/usr/local/bin/draupnir-create-instance \
		cmd/draupnir-destroy-image=/usr/local/bin/draupnir-destroy-image \
		cmd/draupnir-destroy-instance=/usr/local/bin/draupnir-destroy-instance

clean:
	-rm -f draupnir draupnir.linux_amd64 *.deb

publish-base-dockerfile:
	docker build -t gocardless/draupnir-base . \
		&& docker push gocardless/draupnir-base

publish-circleci-dockerfile:
	docker build -t gocardless/draupnir-circleci .circleci \
		&& docker push gocardless/draupnir-circleci
