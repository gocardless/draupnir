Draupnir
========

> *Odin laid upon the pyre the gold ring called Draupnir; this quality attended it: that every ninth night there fell from it eight gold rings of equal weight.*

Draupnir is a tool that provides on-demand Postgres databases with preloaded data.

Development
-----------

Prerequisites:
- Go
- Postgresql

Install [gom](https://github.com/mattn/gom)
```
go get github.com/mattn/gom
```

Install dependencies
```
gom install
```

Create the database
```
createdb draupnir
```

Migrate the database
```
vendor/bin/sql-migrate up
```

Tests
-----

To run the unit tests:
```
go test $(go list ./... | grep -v /vendor)
```

To run the integration tests:
```
vagrant up
bundle
rspec
```
