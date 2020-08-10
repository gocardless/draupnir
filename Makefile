VERSION="$(shell cat DRAUPNIR_VERSION)"
BUILD_COMMAND=go build -ldflags "-X github.com/gocardless/draupnir/pkg/version.Version=$(VERSION)"

.PHONY: build client clean test test-integration dump-schema publish-circleci-dockerfile

build-linux:
	GOOS=linux GOARCH=amd64 $(BUILD_COMMAND) -o draupnir.linux_amd64 cmd/draupnir/draupnir.go

build-osx:
	GOOS=darwin GOARCH=amd64 $(BUILD_COMMAND) -o draupnir.darwin_amd64 cmd/draupnir/draupnir.go

build: build-linux build-osx

migrate:
	# https://github.com/rubenv/sql-migrate
	sql-migrate up

dump-schema:
	pg_dump --schema-only --no-privileges --no-owner --file structure.sql draupnir

test:
	go test ./...
	go vet ./...

test-integration:
	bundle exec rspec

build-production: test
	GOOS=linux GOARCH=amd64 $(BUILD_COMMAND) -o draupnir.linux_amd64 cmd/draupnir/draupnir.go
	GOOS=darwin GOARCH=amd64 $(BUILD_COMMAND) -o draupnir.darwin_amd64 cmd/draupnir/draupnir.go

deb: build-production
	fpm -f -s dir -t $@ -n draupnir -v $(VERSION) \
		--description "Databases on demand" \
		--maintainer "GoCardless Engineering <engineering@gocardless.com>" \
		draupnir.linux_amd64=/usr/local/bin/draupnir \
		cmd/draupnir-create-instance=/usr/local/bin/draupnir-create-instance \
		cmd/draupnir-destroy-image=/usr/local/bin/draupnir-destroy-image \
		cmd/draupnir-destroy-instance=/usr/local/bin/draupnir-destroy-instance \
		cmd/draupnir-finalise-image=/usr/local/bin/draupnir-finalise-image \
		cmd/draupnir-start-image=/usr/local/bin/draupnir-start-image

clean:
	-rm -f draupnir draupnir.*_amd64 *.deb

publish-base-dockerfile:
	docker build -t gocardless/draupnir-base . \
		&& docker push gocardless/draupnir-base

publish-circleci-dockerfile:
	docker build -t gocardless/draupnir-circleci .circleci \
		&& docker push gocardless/draupnir-circleci
