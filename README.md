Draupnir
========

> *Odin laid upon the pyre the gold ring called Draupnir; this quality attended it: that every ninth night there fell from it eight gold rings of equal weight.*

Draupnir is a tool that provides on-demand Postgres databases with preloaded data.

Development
===========

Prerequisites:
- Go
- Postgresql

Install [dep](https://github.com/golang/dep)
```
brew install dep
```

Install dependencies
```
dep ensure
```

Create the database
```
createdb draupnir
```

Migrate the database
```
make migrate
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

204 No Content
```

You can create as many instances of a particular image as you want, without
worrying about disk space. Draupnir will only consume disk space for new data
that you write to your instances.

CLI
---

Draupnir comes with a command-line client, `draupnir-client`. Once you
authenticate via OAuth, you can use it to manage your instances.

The CLI has built-in help (`draupnir-client help`). For help on sub-commands,
use an invocation like `draupnir-client images help` instead of
`draupnir-client help images`.

#### Authenticate
```
draupnir-client authenticate
```

#### List Images
```
draupnir-client images list
```

#### Create an instance of Image 3
```
draupnir-client instances create 3
```

#### Connect to instance 4
```
eval $(draupnir-client env 4)
psql
```

#### Destroy instance 4
```
draupnir-client instances destroy 4
```

API
===

### Images
#### List Images
```http
GET /images HTTP/1.1
Content-Type: application/json
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
Authorization: Bearer 123

204 No Content
```
