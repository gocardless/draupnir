VERSION=0.0.1
BUILD_COMMAND=go build -ldflags "-X main.version=$(VERSION)"

.PHONY: build clean

build:
	$(BUILD_COMMAND) -o draupnir *.go

test:

build-production: test
	GOOS=linux GOARCH=amd64 $(BUILD_COMMAND) -o draupnir.linux_amd64 *.go

deb: build-production
	bundle exec fpm -s dir -t $@ -n draupnir -v $(VERSION) \
		--description "Databases on demand" \
		--maintainer "GoCardless Engineering <engineering@gocardless.com>" \
		draupnir.linux_amd64=/usr/local/bin/draupnir \

clean:
	-rm -f draupnir draupnir.linux_amd64
