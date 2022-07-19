Upgrading db.tar
-----------

In order to upgrade the Postgresql version that draupnir uses, we must also upgrade the `db.tar` file.
`db.tar` is a PG data directory, and is needed in our integration tests in order to ensure that everything runs correctly.

Here's how to upgrade it:

1. Update the Postgresql version in the Dockerfile to the version you are upgrading to, see this [PR's Dockerfile modification for an example](https://github.com/gocardless/draupnir/commit/e87ffe7fb8603d195236d63835876437b9788a64).

2. Build the draupnir container image, start up a draupnir docker container and install the old postgresql version

```bash
docker build -t gocardless/draupnir-base .
docker run -it gocardless/draupnir-base bash

# Install the old Postgresql version
apt-get upgrade
apt-get install -y postgresql-<YOUR-OLD-PG-VERSION>
```

3. Get the container's id and copy the `db.tar` to the container's filesystem.

```bash
docker ps
# copy the container id
docker cp draupnir/spec/fixtures/db.tar  <CONTAINER_ID>:/tmp
```

4. In the running container, delete the old PG data directory and extract the `db.tar` to it.

```bash
rm -rf /var/lib/postgresql/<YOUR-OLD-PG-VERSION>/main/*
tar -xf /tmp/db.tar -C /var/lib/postgresql/<YOUR-OLD-PG-VERSION>/main/
```

5. Init the new postgresql DB, run pg_upgrade, and create a new db tar file.

```bash
cd /tmp

sudo -u postgres /usr/lib/postgresql/<YOUR-NEW-PG-VERSION>/bin/initdb -E SQL_ASCII -D /tmp/postgresql-<YOUR-NEW-PG-VERSION>/

sudo -u postgres /usr/lib/postgresql/<YOUR-NEW-PG-VERSION>/bin/pg_upgrade \
  --old-datadir "/var/lib/postgresql/<YOUR-OLD-PG-VERSION>/main" \
  --new-datadir "/tmp/postgresql-<YOUR-NEW-PG-VERSION>" \
  --old-bindir "/usr/lib/postgresql/<YOUR-OLD-PG-VERSION>/bin" \
  --new-bindir "/usr/lib/postgresql/<YOUR-NEW-PG-VERSION>/bin"

cd /tmp/postgres-<YOUR-NEW-PG-VERSION>
tar -czf /tmp/new-db.tar .
```

6. extract new-db.tar file to your machine's filesystem

```bash
docker cp <YOUR-CONTAINER-ID>:/tmp/new-db.tar ~/new-db.tar
```