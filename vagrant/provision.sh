#!/usr/bin/env bash

set -euo pipefail
set -x

iptables_add_if_missing() {
  iptables -C "$@" || iptables -A "$@"
}

# prevent psql error messages
cd /

# add postgres repo
cat > /etc/apt/sources.list.d/pgdg.list <<END
deb http://apt.postgresql.org/pub/repos/apt/ bionic-pgdg main
END

# get the signing key and import it
curl -Ss https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -

# fetch the metadata from the new repo
apt-get update

# install postgres 11 and go. build-essential is required for cgo
apt-get install -y --no-install-recommends build-essential postgresql-11 golang-go
export PATH=$PATH:/root/go/bin

# install sql-migrate
go get -v github.com/rubenv/sql-migrate/...
cp /root/go/bin/sql-migrate /usr/local/bin

mkdir -p /data

# create and mount btrfs
if ! btrfs filesystem df /data >/dev/null 2>&1; then
    mkfs.btrfs -f /dev/sdc
    mount /dev/sdc /data
fi

# create system user
getent passwd draupnir >/dev/null || useradd --groups ssl-cert --create-home draupnir

# create draupnir directories
mkdir -p /data/{image_uploads,image_snapshots,instances}
# Ignore a failing status code, as this will error if re-provisioning after
# btrfs snapshots have been created.
chown -R draupnir /data || echo "Failed to chown some directories"

# create draupnir postgres instance user
getent passwd draupnir-instance >/dev/null || useradd draupnir-instance

# create draupnir postgres instance log directory
mkdir -p /var/log/postgresql-draupnir-instance
chgrp draupnir-instance /var/log/postgresql-draupnir-instance
chmod 775 /var/log/postgresql-draupnir-instance

# Ubuntu starts the DB after installation. Stop so that we can make a copy of the DB.
pg_ctlcluster 11 main stop
# wait for postgres to stop, so that the pid file disappears
sleep 1

if [ ! -d /data/example_db ]; then
  mkdir /data/example_db
  chown postgres:postgres /data/example_db
  sudo -u postgres /usr/lib/postgresql/11/bin/initdb /data/example_db

  sudo -u postgres /usr/lib/postgresql/11/bin/pg_ctl -D /data/example_db -o '-c data_directory=/data/example_db' start
  sudo -u postgres psql -f /draupnir/vagrant/example_db.sql
  sudo -u postgres /usr/lib/postgresql/11/bin/pg_ctl -D /data/example_db -o '-c data_directory=/data/example_db' stop
fi

# start draupnir postgres
pg_ctlcluster 11 main start

# create draupnir user
if ! sudo -u postgres psql -Atc "SELECT 1 FROM pg_roles WHERE rolname='draupnir'" | grep -q 1; then
  sudo -u postgres createuser draupnir
fi
# create draupnir database
if ! sudo -u postgres psql -Atc "SELECT 1 FROM pg_database WHERE datname='draupnir'" | grep -q 1; then
  sudo -u postgres createdb --owner=draupnir draupnir
fi

cd /draupnir && sudo -u draupnir sql-migrate up -env=vagrant && cd -

# prepare configuration
mkdir -p /etc/draupnir
ln -sf /draupnir/vagrant/draupnir_config.toml /etc/draupnir/config.toml
ln -sf /draupnir/vagrant/draupnir_client_config.toml /root/.draupnir
ln -sf /draupnir/vagrant/draupnir.service /etc/systemd/system/draupnir.service

# make scripts availabe on PATH
ln -sf /draupnir/cmd/draupnir-* /usr/local/bin
# allow Draupnir to sudo its scripts
cp -f /draupnir/vagrant/sudoers_draupnir /etc/sudoers.d/draupnir

mkdir -p /usr/lib/draupnir/bin
ln -sf /draupnir/scripts/iptables /usr/lib/draupnir/bin/iptables

# Setup iptables rules, to enable whitelisting functionality
iptables -N DRAUPNIR-WHITELIST || echo "Chain exists"
iptables_add_if_missing INPUT -i lo -p tcp -m tcp --dport 7432:8432 -j ACCEPT
iptables_add_if_missing INPUT -p tcp -m tcp --dport 7432:8432 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
iptables_add_if_missing INPUT -p tcp -m tcp --dport 7432:8432 -m conntrack --ctstate NEW -j DRAUPNIR-WHITELIST
iptables_add_if_missing INPUT -p tcp -m tcp --dport 7432:8432 -j DROP

systemctl start draupnir

# wait for the server to boot up, before trying to create an image
sleep 1

# create an image, if one doesn't already exist
if ! /draupnir/draupnir.linux_amd64 --insecure images list | grep -E 'READY:.*true'; then
    # create draupnir image
    IMAGE_ID=$(/draupnir/draupnir.linux_amd64 --insecure images create "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "/draupnir/vagrant/anonymisation.sql" | awk '{print $1}')
    IMAGE_PATH="/data/image_uploads/${IMAGE_ID}"

    cp -rp /data/example_db/* "${IMAGE_PATH}"

    cat > "${IMAGE_PATH}/pg_hba.conf" <<-EOF
    local   all     all                     trust
    host    all     all     0.0.0.0/0       trust
EOF

    chown -R postgres:postgres "${IMAGE_PATH}"
    /draupnir/draupnir.linux_amd64 --insecure images finalise "${IMAGE_ID}"
fi
