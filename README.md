Draupnir
========

> *Odin laid upon the pyre the gold ring called Draupnir; this quality attended it: that every ninth night there fell from it eight gold rings of equal weight.*

Draupnir is a tool that provides on-demand Postgres databases with preloaded data.

**Looking to use Draupnir? Please check the developer handbook**

Development
-----------

Prerequisites:
- Go
- Postgresql

Install [dep](https://github.com/golang/dep) if you want to add/remove
dependencies
```
brew install dep
```

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

It will often be desirable to run a full virtual machine, with btrfs, in order
to test the complete Draupnir flow. This can be achieved via the included
Vagrant configuration.

Install prerequisites:
```
brew cask install virtualbox vagrant
```

Build the Linux binary
```
make build-linux
```

Boot Vagrant VM:
```
vagrant up
```

Login and use Draupnir
```
vagrant ssh
$ sudo su -
# eval $(/draupnir/draupnir.linux_amd64 --insecure new)
# export PGHOST=localhost
# psql
```

Tests
-----

To run the unit tests:
```
make test
```

To run the integration tests:
```
make deb && test-integration
```

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

| Field                      | Required | Description
|----------------------------|----------|---------------------------------------|
| `database_url`             | True     | A postgresql [connection URI](https://www.postgresql.org/docs/9.5/static/libpq-connect.html#LIBPQ-CONNSTRING) for draupnir's internal database.
| `data_path`                | True     | The path to draupnir's data directory, where all images and instances will be stored.
| `environment`              | True     | The environment. This can be any value, but if it is set to "test", draupnir will use a stubbed authentication client which allows all requests specifying an access token of `the-integration-access-token`. This is intended for integration tests - don't use it in production. The environment will be included in all log messages.
| `shared_secret`            | True     | A hardcoded access token that can be used by automated scripts which can't authenticate via OAuth. At GoCardless we use this to automatically create new images.
| `trusted_user_email_domain`| True     | The domain under which users are considered "trusted". This is draupnir's rudimentary form of authentication: if a user athenticates via OAuth and their email address is under this domain, they will be allowed to use the service. This domain must start with a `@`, e.g. `@gocardless.com`.
| `sentry_dsn`               | False    | The DSN for your [Sentry](https://sentry.io/) project, if you're using Sentry.
| `http.port`                | True     | The port that the HTTPS server will bind to.
| `http.insecure_port`       | True     | The port that the HTTP server will bind to.
| `http.tls_certificate`     | True     | The path to the TLS certificate file that the HTTPS server will use.
| `http.tls_private_key`     | True     | The path to the TLS private key that the HTTPS server will use.
| `oauth.redirect_url`       | True     | The redirect URL for the OAuth flow.
| `oauth.client_id`          | True     | The OAuth client ID.
| `oauth.client_secret`      | True     | The OAuth client secret.

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
