VERSION=0.0.1
BUILD_COMMAND=go build -ldflags "-X main.version=$(VERSION)"

.PHONY: build clean test test-integration dump-schema

build:
	$(BUILD_COMMAND) -o draupnir *.go

migrate:
	vendor/bin/sql-migrate up

dump-schema:
	pg_dump -sxOf structure.sql draupnir

test:
	go test ./routes ./models ./store

test-integration:
	vagrant destroy -f && vagrant up
	vagrant ssh -c "sudo service draupnir start"
	bundle exec rspec

setup-cookbook:
	mkdir -p tmp/cookbooks/
	git clone git@github.com:gocardless/chef-draupnir.git tmp/cookbooks/draupnir

update-cookbook:
	cd tmp/cookbooks/draupnir && git pull && bundle && bundle exec berks

build-production: test
	GOOS=linux GOARCH=amd64 $(BUILD_COMMAND) -o draupnir.linux_amd64 *.go

deb: build-production
	bundle install
	bundle exec fpm -f -s dir -t $@ -n draupnir -v $(VERSION) \
		--description "Databases on demand" \
		--maintainer "GoCardless Engineering <engineering@gocardless.com>" \
		draupnir.linux_amd64=/usr/local/bin/draupnir \
		cmd/draupnir-finalise-image=/usr/local/bin/draupnir-finalise-image

clean:
	-rm -f draupnir draupnir.linux_amd64
