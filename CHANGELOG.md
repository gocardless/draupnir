Changelog
=========

Unreleased
----------

- Nothing

4.1.0
-----

- Perform dynamic whitelisting of user IP addresses, to conceal instances and
  prevent brute-forcing.
- Provide access to instances via a `draupnir` postgres user, rather than the
  `postgres` role which has SUPERUSER privileges.
- Don't run Draupnir instances as the postgres unix user.
- Further harden TLS cipher suite configuration.

4.0.2
-----

- Check when creating instances that non-authenticated connections are denied.
  - This was already the case, but this code serves as a defense for any
  potential regressions.

4.0.1
-----

- Harden TLS cipher suite configuration.

4.0.0
-----

**BREAKING CHANGES**

- Secure instances with client certificate authentication.
  - This is a mandatory feature and therefore requires a client running the same
    version in order to consume the additional API fields.

**Other improvements**

- Automatically destroy user instances where their token is no longer valid.
- Allow configuration of a port range for instances.
- Allow configuration of a separate hostname that Draupnir instances should be
  accessed via.
- Add Vagrant VM for development and validation.
- Replace custom parallel vacuum with vacuumdb.

**Bug fixes**

- Fix error-passing in middlewares.
- Prevent instance port collisions.

3.1.0
-----

- Upgrade to PostgreSQL 11

3.0.3
-----

- Actually fix the version passed to ldflags this time.

3.0.2
-----

- Fix version passed to ldflags

3.0.1
-----

- Wait for Postgres to boot fully during image finalisation.

3.0.0
-----

- Merge client and server binaries into one. Now you use `draupnir` to do everything.

2.0.1
-----

- Fix bug where the `POST /access_tokens` route was behind authentication,
  preventing users from authenticating for the first time.

2.0.0
-----

- Replace environment variable configuration with a configuration file installed
  at /etc/draupnir/config.toml. Proxy settings are still configured via the
  HTTP_PROXY and HTTPS_PROXY environment variables, but everything else is
  configured using the config file. The format is documented in the README.
- Vacuum databases by default after the anonymisation step, to avoid the vacuum
  processes starting in all instances of a draupnir database and causing large
  amounts of exclusive data to be generated per snapshot

1.7.0
-----

- Limit `temp_file_size` to 5GB (#47)
- Add error handling to routes, improve logging (#45)
- Deploy from CircleCI (#48)

1.6.0
-----

- Add --insecure flag to the draupnir-client
- Optionally unpack a database upload tar

1.5.0
-----

- Listen on https locally (#41)

1.4.0
-----

- Check version semantically- don't fail requests for exact version equality (#37)
- Client can now create and finalise images

1.3.0
-----

- Allow the upload user to delete any instance via API

1.2.0
-----

- Additional logging for the draupnir server
- Support reporting exceptions to [Sentry](https://sentry.io) via the
- DRAUPNIR_SENTRY_DSN environment variable

1.1.0
-----

- Allow the trusted email domain to be specified via an environment variable.
- Fix a bug where the Draupnir-Version header would not be included in the API
  response when the client's header didn't match the server version.
- Don't require the Draupnir-Version header for the health check endpoint.
- Change the config file format from JSON to TOML.
- Allow the default database to be set as a config option.
- Log HTTP requests to STDOUT.

1.0.0
-----

- Switch to using OAuth Refresh Tokens for authentication, so users don't have
  to authenticate so often.
- Require client and server versions to be identical to cooperate. This should
  make it easier to handle breaking changes, at the expense of requiring more
  frequent client upgrades.

0.1.4
-----

- Fix a bug where uploaded database archives were not being deleted.
- Minor fix to CLI output formatting

0.1.3
-----

- Cleanup compressed upload after extraction

0.1.2
-----

- Add quick start example to cli help

0.1.1
-----

- Add `new` command to client
  - This is a shortcut to create an instance of the latest image.

0.1.0
-----

- First release
