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
