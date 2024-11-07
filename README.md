Draupnir
========

[![explore database with Azimutt](https://img.shields.io/badge/PostgreSQL-browse_online-gray?labelColor=4169E1&logo=postgresql&logoColor=fff&style=flat)](https://azimutt.app/create?sql=https://raw.githubusercontent.com/gocardless/draupnir/refs/heads/master/structure.sql&name=Draupni)

Draupnir is a tool that provides on-demand Postgres databases with preloaded data.

> *Odin laid upon the pyre the gold ring called Draupnir; this quality attended it: that every ninth night there fell from it eight gold rings of equal weight.*

Development
-----------

Prerequisites:
- Go
- Postgresql
- Ruby

Create the database
```
createdb draupnir
```

Migrate the database
```
make migrate
```

Development (Vagrant VM)
------------------------

**!! Disclaimer, the vagrant VM is currently unsupported on Apple Silicon**
It will often be desirable to run a full virtual machine, with btrfs, in order
to test the complete Draupnir flow. This can be achieved via the included
Vagrant configuration.

Install prerequisites:
```
brew cask install virtualbox vagrant
```

Build the Linux binary:
```
make build-linux
```

Boot Vagrant VM:
```
vagrant up
```

Login and use Draupnir:
```
vagrant ssh
$ sudo su -
# eval $(draupnir new)
# psql -d myapp
```

After making changes to the code, to restart the server:
```
make build-linux && vagrant up --provision
```

Tests
-----

To run the unit tests:
```
make test
```

To run the integration tests, ensure you've run `make build-linux` before running:
```
make test-integration
```

# Releases
For releases, this project uses [GoReleaser](https://goreleaser.com/). The configuration was done in such a way that
releases happen on any commit to the main branch that also updates [DRAUPNIR_VERSION](./DRAUPNIR_VERSION), and should be
accompanied by an update to [CHANGELOG.md](CHANGELOG.md) to make it explicit what has changed.

Version updates should follow the [Semantic Versioning](https://semver.org/) guidelines.

Usage
=====

Draupnir provides an API to create, use and manage instances of your database.
There are two API resources: _Images_ and _Instances_. An Image is a database
backup that you upload to Draupnir. Instances are lightweight copies of a
particular Image that you can create, use and destroy with ease. We'll walk
through the basics of using Draupnir. The full API reference is at the bottom of
this document.

### Creating an Image
Create a new Image by `POST`ing to `/images`, providing a timestamp for the
backup and an anonymisation script that will be run against the backup. You can
use this to remove any sensitive data from your backup before serving it to
users.
```http
POST /images HTTP/1.1
Content-Type: application/json
Draupnir-Version: 1.0.0
Authorization: Bearer 123

{
  "data": {
    "type": "images",
    "attributes": {
      "backed_up_at": "2017-05-01T12:00:00Z",
      "anonymisation_script": "\c my_db\nDELETE FROM secret_tokens;"
    }
  }
}

201 Created
{
  "data": {
    "type": "images",
    "id": 1,
    "attributes": {
      "backed_up_at": "2017-05-01T12:00:00Z",
      "created_at": "2017-05-01T15:00:00Z",
      "updated_at": "2017-05-01T15:00:00Z",
      "ready": false
    }
  }
}
```

### Uploading an Image
Once you've created an Image, you can upload it. This is done by `scp`ing a
tarball of the database data directory to Draupnir. The upload is authenticated
with an ssh key which you'll create when setting up Draupnir.
```
scp -i key.pem db_backup.tar.gz upload@my-draupnir.tld:/draupnir/image_uploads/1
```

Once you've uploaded the backup, inform Draupnir that you're ready to finalise
the image. This may take some time, as Draupnir will spin up Postgres and run
the anonymisation script.
```http
POST /images/1/done HTTP/1.1
Content-Type: application/json
Draupnir-Version: 1.0.0
Authorization: Bearer 123

200 OK
{
  ...
}
```

### Creating Instances
Now you've got an image, you can create instances of it. The process for this is
very simple.
```http
POST /instances HTTP/1.1
Content-Type: application/json
Draupnir-Version: 1.0.0
Authorization: Bearer 123

{
  "data": {
    "type": "instances",
    "attributes": {
      "image_id": 1
    }
  }
}

201 Created
{
  "data": {
    "type": "instances",
    "id": 1,
    "attributes": {
      "created_at": "2017-05-01T16:00:00Z",
      "updated_at": "2017-05-01T16:00:00Z",
      "image_id": 1,
      "port": "5678"
    }
  }
}
```

You now have a Postgres server up and running, containing a copy of your
database. You can connect to it like you would any other database.
```
PGHOST=my-draupnir.tld PGPORT=5678 psql my-db
```

You can make any modifications to this database and they won't affect the
original backup. When you're done, just destroy the instance.
```http
DELETE /instances/1
Authorization: Bearer 123
Draupnir-Version: 1.0.0

204 No Content
```

You can create as many instances of a particular image as you want, without
worrying about disk space. Draupnir will only consume disk space for new data
that you write to your instances.

Configuration
------------

When draupnir boots it looks for a config file at `/etc/draupnir/config.toml`.
This file must specify all required configuration variables in order for
Draupnir to boot. The variables are as follows:

| Field                          | Required | Description
|--------------------------------|----------|---------------------------------------|
| `database_url`                 | True     | A postgresql [connection URI](https://www.postgresql.org/docs/9.5/static/libpq-connect.html#LIBPQ-CONNSTRING) for draupnir's internal database.
| `data_path`                    | True     | The path to draupnir's data directory, where all images and instances will be stored.
| `environment`                  | True     | The environment. This can be any value, but if it is set to "test", draupnir will use a stubbed authentication client which allows all requests specifying an access token of `the-integration-access-token`. This is intended for integration tests - don't use it in production. The environment will be included in all log messages.
| `shared_secret`                | True     | A hardcoded access token that can be used by automated scripts which can't authenticate via OAuth. At GoCardless we use this to automatically create new images.
| `trusted_user_email_domain`    | True     | The domain under which users are considered "trusted". This is draupnir's rudimentary form of authentication: if a user athenticates via OAuth and their email address is under this domain, they will be allowed to use the service. This domain must start with a `@`, e.g. `@gocardless.com`.
| `public_hostname`              | True     | The hostname that will be set as PGHOST. This is configurable as it may be different to the hostname of the _API address_ that clients communicate with.
| `sentry_dsn`                   | False    | The DSN for your [Sentry](https://sentry.io/) project, if you're using Sentry.
| `clean_interval`               | True     | The interval at which Draupnir checks and removes any instance associated with a user that no longer has a valid refresh token. Valid values are a sequence of digits followed by a unit, such as "30m", "6h". See [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration).
| `min_instance_port`            | True     | The minimum port number (inclusive) that may be used when creating a Draupnir instance.
| `max_instance_port`            | True     | The maximum port number (exclusive) that may be used when creating a Draupnir instance.
| `enable_ip_whitelisting`       | False    | Whether to enable the [IP whitelisting module](#ip-address-whitelisting).
| `whitelist_reconcile_interval` | False    | If IP whitelisting is enabled, this is the interval at which Draupnir reconciles the IP address whitelist with what's in iptables, in order to clean up incorrect state. Uses the same format as `clean_interval`.
| `use_x_forwarded_for`          | False    | Whether to use the `X-Forwarded-For` header when determining the real user IP address. See [documentation](#identification-of-user-ip-addresses).
| `trusted_proxy_cidrs`          | False    | A list of CIDRs that will match your load balancer IP addresses. Example: `["10.32.0.0/16"]`. See [documentation](#identification-of-user-ip-addresses).
| `http.listen_address`          | False    | The address and port that the HTTPS server will bind to.
| `http.insecure_listen_address` | False    | The address and port that the HTTP server will bind to.
| `http.tls_certificate`         | False    | The path to the TLS certificate file that the HTTPS server will use.
| `http.tls_private_key`         | False    | The path to the TLS private key that the HTTPS server will use.
| `oauth.redirect_url`           | True     | The redirect URL for the OAuth flow.
| `oauth.client_id`              | True     | The OAuth client ID.
| `oauth.client_secret`          | True     | The OAuth client secret.

For a complete example of this file, see `spec/fixtures/config.toml`.

CLI
---

Draupnir ships as a single binary which can be used to run the server or use as a client
to manage your instances.

The CLI has built-in help (`draupnir help`). For help on sub-commands, use an invocation
like `draupnir images help` instead of `draupnir help images`.

#### Authenticate
```
draupnir authenticate
```

#### List Images
```
draupnir images list
```

#### Create an instance of Image 3
```
draupnir instances create 3
```

#### Connect to instance 4
```
eval $(draupnir env 4)
psql
```

#### Destroy instance 4
```
draupnir instances destroy 4
```

API
===

The Draupnir API roughly follows the JSON API spec, with a few deviations.
The only supported `Content-Type` is `application/json`. Authentication is
required for most API endpoints and is provided in the form of an access token
in the `Authorization` header.

The API also requires a `Draupnir-Version` header to be set. This version must
be exactly equal to the version of Draupnir serving the API. The CLI and server
are distributed as one, and share a version number. We enforce equality here as
a conservative measure to ensure that the CLI and API can interoperate
seamlessly. In the future we might relax this constraint.

### Images
#### List Images
```http
GET /images HTTP/1.1
Content-Type: application/json
Draupnir-Version: 1.0.0
Authorization: Bearer 123

200 OK
{
  "data": [
    {
      "type": "images",
      "attributes": {
        "backed_up_at": "2017-05-01T12:00:00Z",
        "anonymisation_script": "\c my_db\nDELETE FROM secret_tokens;"
      }
    }
  ]
}
```

#### Get Image
```http
GET /images/1 HTTP/1.1
Content-Type: application/json
Draupnir-Version: 1.0.0
Authorization: Bearer 123

200 OK
{
  "data": {
    "type": "images",
    "attributes": {
      "backed_up_at": "2017-05-01T12:00:00Z",
      "anonymisation_script": "\c my_db\nDELETE FROM secret_tokens;"
    }
  }
}
```

#### Create Image
```http
POST /images HTTP/1.1
Content-Type: application/json
Draupnir-Version: 1.0.0
Authorization: Bearer 123

{
  "data": {
    "type": "images",
    "attributes": {
      "backed_up_at": "2017-05-01T12:00:00Z",
      "anonymisation_script": "\c my_db\nDELETE FROM secret_tokens;"
    }
  }
}

201 Created
{
  "data": {
    "type": "images",
    "id": 1,
    "attributes": {
      "backed_up_at": "2017-05-01T12:00:00Z",
      "created_at": "2017-05-01T15:00:00Z",
      "updated_at": "2017-05-01T15:00:00Z",
      "ready": false
    }
  }
}
```

#### Finalise Image
```http
POST /images/1/done HTTP/1.1
Content-Type: application/json
Draupnir-Version: 1.0.0
Authorization: Bearer 123

200 OK
{
  "data": {
    "type": "images",
    "id": 1,
    "attributes": {
      "backed_up_at": "2017-05-01T12:00:00Z",
      "created_at": "2017-05-01T15:00:00Z",
      "updated_at": "2017-05-01T15:01:00Z",
      "ready": true
    }
  }
}
```

#### Destroy Image
```http
DELETE /images/1
Authorization: Bearer 123

204 No Content
```

### Instances
#### List Instances
```http
GET /instances HTTP/1.1
Content-Type: application/json
Draupnir-Version: 1.0.0
Authorization: Bearer 123

200 Ok
{
  "data": [
    {
      "type": "instances",
      "id": 1,
      "attributes": {
        "created_at": "2017-05-01T16:00:00Z",
        "updated_at": "2017-05-01T16:00:00Z",
        "image_id": 1,
        "port": "5678"
      }
    }
  ]
}
```

#### Get Instance
```http
GET /instances HTTP/1.1
Content-Type: application/json
Draupnir-Version: 1.0.0
Authorization: Bearer 123

200 Ok
{
  "data": {
    "type": "instances",
    "id": 1,
    "attributes": {
      "created_at": "2017-05-01T16:00:00Z",
      "updated_at": "2017-05-01T16:00:00Z",
      "image_id": 1,
      "port": "5678"
    }
  }
}
```

#### Create Instance
```http
POST /instances HTTP/1.1
Content-Type: application/json
Draupnir-Version: 1.0.0
Authorization: Bearer 123

{
  "data": {
    "type": "instances",
    "attributes": {
      "image_id": 1
    }
  }
}

201 Created
{
  "data": {
    "type": "instances",
    "id": 1,
    "attributes": {
      "created_at": "2017-05-01T16:00:00Z",
      "updated_at": "2017-05-01T16:00:00Z",
      "image_id": 1,
      "port": "5678"
    }
  }
}
```

#### Destroy Instance
```
DELETE /instances/1 HTTP/1.1
Draupnir-Version: 1.0.0
Authorization: Bearer 123

204 No Content
```

# Internal Architecture

Draupnir is basically two things: a manager for [BTRFS](https://btrfs.wiki.kernel.org/index.php/Main_Page)
volumes and a supervisor of PostgreSQL processes.
Each image is stored in its own BTRFS subvolume, and instances are created by
creating a snapshot of the image's subvolume, and booting a Postgres instance in
it. In order to do this, Draupnir requires read-write access to a disk formatted
with BTRFS. The path to this disk is specified at runtime by the `DRAUPNIR_DATA_PATH` environment variable.
The whole process looks like this (assuming `DRAUPNIR_DATA_PATH=/draupnir`):

1. An image is created via the API (`POST /images`). This creates a record in Draupnir's
   internal database and an empty subvolume is created at
   `/draupnir/image_uploads/1` (where `1` is the image ID). The user may specify
   an anonymisation script to be run on the data before it is made available. At
   this point, the image is marked as "not ready", meaning it cannot be used to
   create instances.
2. A PostgreSQL backup, in the form of a tar file, is pushed into the server
   over SCP. The ssh credentials for this operation are set when the machine is
   provisioned, via the [chef cookbook](https://github.com/gocardless/chef-draupnir).
   The backup is pushed directly into `/draupnir/image_uploads/1`.
3. The image is finalised via the API (`POST /images/1/done`). This indicates to Draupnir that the
   backup has completed and no more data needs to be pushed. Draupnir prepares
   the directory so Postgres will boot from it, and runs the anonymisation
   script. For more detail on this step see `cmd/draupnir-finalise-image`.
   Finally, Draupnir will create a BTRFS snapshot of the subvolume at
   `/draupnir/image_snapshots/1`. This snapshot is read-only and ensures that the image
   will not change from now on. At this point Draupnir marks the image as
   "ready", meaning that instances can be created from it.
4. A user creates an instance from this image via the API (`POST /instances`).
   First, draupnir creates a corresponding record in its database. Then it will
   take a further snapshot of the image: `/draupnir/image_snapshots/1 ->
   /draupnir/instances/1` (where `1` is the instance ID). It will start a
   Postgres process, setting the data directory to `/draupnir/instances/1` and
   binding it to a random port (which we persist in the database as part of the
   instance).
5. The instance is now running and can accept external connections (the port
   range used for instances is exposed via an iptables rule in the cookbook).
   The user can connect to the instance as if it were any other database, simply
   by specifying the host (whatever server Draupnir is running on), the port
   (serialised in the API) and valid user credentials.  We expect that the user
   already knows the credentials for a user in their database, or alternatively
   they can use the `postgres` user which we create (with no password) as part
   of step 3.
6. The user destroys the instance via the API (`DELETE /instances/1`). Draupnir
   stops the Postgres process for that instance and deletes the snapshot
   `/draupnir/instances/1`.
7. The image is destroyed via an API call (`DELETE /images`). All instances of
   this image are destroyed as per step 6, and then the image is destroyed by
   removing the directories `/draupnir/image_snapshots/1` and
   `/draupnir/image_uploads/1`.

All interaction with BTFS and Postgres is done via a collection of small shell
scripts in the `cmd` directory - read them if you want to know more.

Right now modifications to images (creation, finalisation, deletion) are
restricted to a single "upload" user, who authenticates with the API via a
shared secret.

## Security model

Draupnir has been designed to be deployed on a publicly-accessible instance, but
restrict access to the potentially sensitive data in the Draupnir images to
authorised users only.

### API access

Access to the API is secured via Google OAuth. A user must have a valid token in
order to create, retrieve or destroy a Draupnir instance.

### Connecting to Draupnir Postgres instances

Access to a Draupnir Postgres instance is secured via a client-authenticated TLS
connection.
The client certificate and key are served via the API and then
stored in a secure temporary location on the client machine. The paths to these
files are then used to set `PGSSLCERT` and `PGSSLKEY`.

Additionally, `PGSSLMODE` is set to `verify-ca`, meaning that the Postgres
client will *only* attempt to connect via TLS, and will also only successfully
connect to the instance if it provides the expected CA certificate.

On the server side, when an image is finalised the `pg_hba.conf` file is setup
so that the only method of access is client-authenticated TLS, and this
therefore propagates to every instance created from the image.
This property ensures that even if a user was to login as a Postgres superuser
on their instance and set a blank password for a given database user, then still
nobody would be able to connect without a valid client certificate and key.

Each instance has a unique CA, server and client certificate, all generated at
creation time, meaning that certificates and keys cannot be reused across
instances and that once the instance is created the locally-stored credentials
are useless.
Given that an instance's details (and therefore credentials) can only
be retrieved by the user that created that instance, it also means that only the
owning user has access to connect to the instance.

### IP address whitelisting

Draupnir provides the ability to dynamically whitelist user IP addresses
to further secure the Postgres instances that it creates, protecting them from
automated scans and attacks.

This is achieved via iptables rules. If this component is enabled
(`enable_ip_whitelisting = true` in the server config) then the Draupnir
daemon will maintain an iptables chain named `DRAUPNIR-WHITELIST`. It is
therefore **the administrator's responsibility** to provision other iptables
rules that reference this chain.

When a user creates an instance, or retrieves the details of one of their own
instances, this chain will be populated with a rule that allows _new
connections_ to the Postgres instance port, from their IP address only. The rule
will be removed as soon as the instance is destroyed.

An example configuration is provided below. This will work in configurations
where the default policy of the `INPUT` chain is `ACCEPT`. For those with
defaults of `DROP`, the third rule can be omitted.

```
# In this example, our Draupnir port range is 6432-7432.

# Setup the DRAUPNIR-WHITELIST chain. The server will create this itself if it's
# missing, but we  won't be able to reference the chain unless we do this.
iptables -N DRAUPNIR-WHITELIST

# Allow all connections on the loopback interface
iptables -A INPUT -i lo -p tcp -m tcp --dport 6432:7432 -j ACCEPT

# For any connections which have been successfully opened, allow further
# communication.
iptables -A INPUT -p tcp -m tcp --dport 6432:7432 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT

# For any new connections, pass them through to the DRAUPNIR-WHITELIST chain
iptables -A INPUT -p tcp -m tcp --dport 6432:7432 -m conntrack --ctstate NEW -j DRAUPNIR-WHITELIST

# For any connections that have not been accepted by the whitelist, drop the
# packet
iptables -A INPUT -p tcp -m tcp --dport 6432:7432 -j DROP
```

The iptables wrapper library used in this project requires root access, and [does
not support sudo](https://github.com/coreos/go-iptables/issues/55). Because it
is strongly recommended to _not_ run the Draupnir server as the root user, this
can be worked around by using the provided [wrapper script](./scripts/iptables)
which is installed into the `/usr/lib/draupnir/bin` directory by the Debian
package.
The Draupnir server process must be executed with a `PATH` variable that places
this directory at the beginning, in order to ensure it is used instead of the
real `iptables` binary.

#### Identification of user IP addresses

The Draupnir server creates whitelist rules based on the IP address of the
user, which it determines by inspecting the HTTP request that was made to its
API.

If your Draupnir API server is fronted by a load balancer, then the HTTP
connection that the Draupnir server receives will originate from that, rather
than the user directly. In this instance a separate mechanism of determining the
user's IP address must be employed; the `X-Forwarded-For` header.

If this scenario applies to you then the following steps must be taken:
1. Ensure that your load balancer places the 'real' user IP address in the
   `X-Forwarded-For` header.
2. Enable the use of the `X-Forwarded-For` header for IP address identification
   by setting the `use_x_forwarded_for` variable to `true`.
3. Define a list of trusted proxies, via the `trusted_proxy_cidrs` setting.
   Any IP addresses in the `X-Forwarded-For` header that match any of these
   CIDRs will be ignored.
   The real user IP address is then determined by taking the resulting list of
   elements of the `X-Forwarded-For` header and using the last one (under the
   assumption that this is the one that your load balancer has added).

If you are not using a load balancer then it is imperative that the
`use_x_forwarded_for` setting remains disabled. If it is enabled without a load
balancer present, rewriting the contents of the header, then it's possible for
an authenticated user to send API requests with a fabricated `X-Forwarded-For`
header and therefore open up their instance(s) to unauthorized IP addresses.

### Cleanup of revoked user instances

When a user creates an instance Draupnir stores the user's refresh token so that
it can, at the `clean_interval`, check that the refresh token is still valid.
In the event that the token isn't valid, the instance is deleted. This ensures
that instances don't remain available longer than the users have access to
Draupnir.

Common causes for an invalid refresh token are:
- The user has revoked the application's third-party access in the Google
  account dashboard.
- The user is suspended via G Suite.
- The user has been deleted.
