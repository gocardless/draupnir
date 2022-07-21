#!/usr/bin/env bash

set -euo pipefail
set -x

iptables_add_if_missing() {
  iptables -C "$@" 2>/dev/null || iptables -A "$@"
}

log_on_failure() {
    echo "Provisioning failed"
    # It's useful to see the logs of the Draupnir server when a failure occurs.
    journalctl -u draupnir | tail -100

    exit 1
}

trap log_on_failure ERR

# prevent psql error messages
cd /

# add postgres repo
cat > /etc/apt/sources.list.d/pgdg.list <<END
deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main
deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -cs)-pgdg 14
END

# get the signing key and import it
curl -Ss https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -

# fetch the metadata from the new repo
apt-get update

# install postgres 14 and go. build-essential is required for cgo
apt-get install -y --no-install-recommends build-essential postgresql-14 postgresql-common
cd /tmp
wget https://dl.google.com/go/go1.17.linux-amd64.tar.gz
sudo tar -xvf go1.17.linux-amd64.tar.gz
sudo mv go /usr/local

export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH

# install sql-migrate
go get -v github.com/rubenv/sql-migrate/...
cp /root/go/bin/sql-migrate /usr/local/bin

mkdir -p /data

# create and mount btrfs
if ! btrfs filesystem df /data >/dev/null 2>&1; then
    mkfs.btrfs -f /dev/sdb
    mount /dev/sdb /data
fi

# create system user
getent passwd draupnir >/dev/null || useradd --groups ssl-cert --create-home draupnir

# create draupnir directories
mkdir -p /data/{image_uploads,image_snapshots,instances}
chown draupnir /data/{image_uploads,image_snapshots,instances}

# create draupnir postgres instance user
getent passwd draupnir-instance >/dev/null || useradd draupnir-instance

# create draupnir postgres instance log directory
mkdir -p /var/log/postgresql-draupnir-instance
chgrp draupnir-instance /var/log/postgresql-draupnir-instance
chmod 775 /var/log/postgresql-draupnir-instance

# Ubuntu starts the DB after installation. Stop so that we can make a copy of the DB.
pg_ctlcluster 14 main stop
# wait for postgres to stop, so that the pid file disappears
sleep 1

if [ ! -d /data/example_db ]; then
  mkdir /data/example_db
  chown postgres:postgres /data/example_db
  sudo -u postgres /usr/lib/postgresql/14/bin/initdb /data/example_db

  sudo -u postgres /usr/lib/postgresql/14/bin/pg_ctl -D /data/example_db -o '-c data_directory=/data/example_db' start
  sudo -u postgres psql -f /draupnir/vagrant/example_db.sql
  sudo -u postgres /usr/lib/postgresql/14/bin/pg_ctl -D /data/example_db -o '-c data_directory=/data/example_db' stop
fi

# start draupnir postgres
pg_ctlcluster 14 main start

# create draupnir user
if ! sudo -u postgres psql -Atc "SELECT 1 FROM pg_roles WHERE rolname='draupnir'" | grep -q 1; then
  sudo -u postgres createuser draupnir
fi
# create draupnir database
if ! sudo -u postgres psql -Atc "SELECT 1 FROM pg_database WHERE datname='draupnir'" | grep -q 1; then
  sudo -u postgres createdb --owner=draupnir draupnir
fi

cd /draupnir && sudo -u draupnir sql-migrate up -env=vagrant && cd -

# Prepare TLS certificates for Draupnir API server
DRAUPNIR_TLS_PATH=/var/draupnir/certificates
mkdir -p "$DRAUPNIR_TLS_PATH"

# Create a CA
openssl req -new -nodes -text \
    -out "${DRAUPNIR_TLS_PATH}/ca.csr" -keyout "${DRAUPNIR_TLS_PATH}/ca.key" \
      -subj "/CN=Draupnir API server certification authority"
chmod 600 "${DRAUPNIR_TLS_PATH}/ca.key"
openssl x509 -req -in "${DRAUPNIR_TLS_PATH}/ca.csr" -text -days 365 \
    -extfile /etc/ssl/openssl.cnf -extensions v3_ca \
      -signkey "${DRAUPNIR_TLS_PATH}/ca.key" -out "${DRAUPNIR_TLS_PATH}/ca.crt"

# Create a server certificate
openssl req -new -nodes -text \
    -out "${DRAUPNIR_TLS_PATH}/server.csr" -keyout "${DRAUPNIR_TLS_PATH}/server.key" \
      -subj "/CN=localhost"
chmod 600 "${DRAUPNIR_TLS_PATH}/server.key"
openssl x509 -req -in "${DRAUPNIR_TLS_PATH}/server.csr" -text -days 30 \
    -extfile <(printf "subjectAltName=DNS:localhost")
    -CA "${DRAUPNIR_TLS_PATH}/ca.crt" -CAkey "${DRAUPNIR_TLS_PATH}/ca.key" -CAcreateserial \
      -out "${DRAUPNIR_TLS_PATH}/server.crt"
chown draupnir "${DRAUPNIR_TLS_PATH}/server.key" "${DRAUPNIR_TLS_PATH}/server.crt"

# Ensure that our cert is trusted via a full certificate chain, so that we can
# use the Draupnir CLI without needing to specify the `--skip-verify` flag
cp "${DRAUPNIR_TLS_PATH}/ca.crt" /usr/local/share/ca-certificates
update-ca-certificates

# prepare configuration
mkdir -p /etc/draupnir
ln -sf /draupnir/vagrant/draupnir_config.toml /etc/draupnir/config.toml
ln -sf /draupnir/vagrant/draupnir_client_config.toml /root/.draupnir
ln -sf /draupnir/vagrant/draupnir.service /etc/systemd/system/draupnir.service

# Make the draupnir binary accessible for use in PATH
ln -sf /draupnir/draupnir.linux_amd64 /usr/local/bin/draupnir
# Make scripts availabe on PATH
ln -sf /draupnir/cmd/draupnir-* /usr/local/bin
# Allow Draupnir to sudo its scripts
cp -f /draupnir/vagrant/sudoers_draupnir /etc/sudoers.d/draupnir

mkdir -p /usr/lib/draupnir/bin
ln -sf /draupnir/scripts/iptables /usr/lib/draupnir/bin/iptables

# Setup iptables rules, to enable whitelisting functionality
iptables -N DRAUPNIR-WHITELIST || echo "Chain exists"
iptables_add_if_missing INPUT -i lo -p tcp -m tcp --dport 7432:8432 -j ACCEPT
iptables_add_if_missing INPUT -p tcp -m tcp --dport 7432:8432 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
iptables_add_if_missing INPUT -p tcp -m tcp --dport 7432:8432 -m conntrack --ctstate NEW -j DRAUPNIR-WHITELIST
iptables_add_if_missing INPUT -p tcp -m tcp --dport 7432:8432 -j DROP

# Perform a full restart in case there have been changes to the binary or config.
systemctl daemon-reload
systemctl stop draupnir
systemctl start draupnir

# wait for the server to boot up, before trying to create an image
sleep 1

# create an image, if one doesn't already exist
if ! draupnir images list | grep -E 'READY:.*true'; then
    # create draupnir image
    IMAGE_ID=$(draupnir images create "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "/draupnir/vagrant/anonymisation.sql" | awk '{print $1}')
    IMAGE_PATH="/data/image_uploads/${IMAGE_ID}"

    cp -rp /data/example_db/* "${IMAGE_PATH}"

    cat > "${IMAGE_PATH}/pg_hba.conf" <<-EOF
    local   all     all                     trust
    host    all     all     0.0.0.0/0       trust
EOF

    chown -R postgres:postgres "${IMAGE_PATH}"
    draupnir images finalise "${IMAGE_ID}"
fi
