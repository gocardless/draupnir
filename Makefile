VERSION=0.1.4
BUILD_COMMAND=go build -ldflags "-X main.version=$(VERSION)"
CLIENT_BUILD_COMMAND=go build -ldflags "-X main.version=$(VERSION)"
PACKAGES=./routes ./models ./store ./auth ./cli ./client

.PHONY: build clean test test-integration dump-schema

build:
	$(BUILD_COMMAND) -o draupnir *.go

migrate:
	vendor/bin/sql-migrate up

dump-schema:
	pg_dump -sxOf structure.sql draupnir

test:
	go vet $(PACKAGES)
	go test $(PACKAGES)

test-integration:
	bundle exec kitchen destroy && bundle exec kitchen converge
	bundle exec kitchen exec -c "sudo -u postgres createdb draupnir"
	bundle exec kitchen exec -c "sudo -u postgres createuser draupnir"
	bundle exec kitchen exec -c "echo \"alter role draupnir password 'draupnir'\" | sudo -u postgres psql"
	bundle exec kitchen exec -c "cat /vagrant/structure.sql | sudo -u draupnir psql draupnir"
	bundle exec kitchen exec -c "sudo sh -c \"echo 'DRAUPNIR_ENVIRONMENT=test' >> /etc/environments/draupnir.env\""
	bundle exec kitchen exec -c "sudo cp /vagrant/fixtures/cert.pem /etc/ssl/certs/draupnir_cert.pem"
	bundle exec kitchen exec -c "sudo cp /vagrant/fixtures/key.pem /etc/ssl/certs/draupnir_key.pem"
	bundle exec kitchen exec -c "sudo service draupnir start"
	bundle exec rspec

setup-cookbook:
	mkdir -p tmp/cookbooks/
	git clone git@github.com:gocardless/chef-draupnir.git tmp/cookbooks/draupnir
	cd tmp/cookbooks/draupnir && bundle && bundle exec berks vendor

update-cookbook:
	cd tmp/cookbooks/draupnir && git pull && rm -rf berks-cookbooks && bundle && bundle exec berks vendor


build-production: test
	GOOS=linux GOARCH=amd64 $(BUILD_COMMAND) -o draupnir.linux_amd64 *.go

client: test
	GOOS=darwin GOARCH=amd64 $(CLIENT_BUILD_COMMAND) -o draupnir-client cli/*.go

deb: build-production
	bundle exec fpm -f -s dir -t $@ -n draupnir -v $(VERSION) \
		--description "Databases on demand" \
		--maintainer "GoCardless Engineering <engineering@gocardless.com>" \
		draupnir.linux_amd64=/usr/local/bin/draupnir \
		cmd/draupnir-finalise-image=/usr/local/bin/draupnir-finalise-image \
		cmd/draupnir-create-instance=/usr/local/bin/draupnir-create-instance \
		cmd/draupnir-destroy-image=/usr/local/bin/draupnir-destroy-image \
		cmd/draupnir-destroy-instance=/usr/local/bin/draupnir-destroy-instance

clean:
	-rm -f draupnir draupnir.linux_amd64
