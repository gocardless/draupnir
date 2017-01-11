VERSION=0.0.1
BUILD_COMMAND=go build -ldflags "-X main.version=$(VERSION)"

.PHONY: build clean test test-integration

build:
	$(BUILD_COMMAND) -o draupnir *.go

test:

test-integration:
	vagrant destroy -f && vagrant up
	vagrant ssh -c "sudo service draupnir start"
	be rspec

build-production: test
	GOOS=linux GOARCH=amd64 $(BUILD_COMMAND) -o draupnir.linux_amd64 *.go

deb: build-production
	bundle install
	bundle exec fpm -f -s dir -t $@ -n draupnir -v $(VERSION) \
		--description "Databases on demand" \
		--maintainer "GoCardless Engineering <engineering@gocardless.com>" \
		draupnir.linux_amd64=/usr/local/bin/draupnir \

clean:
	-rm -f draupnir draupnir.linux_amd64
