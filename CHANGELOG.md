Changelog
=========

Unreleased
----------

No changes.

1.3.0
-----
Allow the upload user to delete any instance via API

1.2.0
-----

Additional logging for the draupnir server
Support reporting exceptions to [Sentry](https://sentry.io) via the
DRAUPNIR_SENTRY_DSN environment variable

1.1.0
-----

Allow the trusted email domain to be specified via an environment variable.
Fix a bug where the Draupnir-Version header would not be included in the API
response when the client's header didn't match the server version.
Don't require the Draupnir-Version header for the health check endpoint.
Change the config file format from JSON to TOML.
Allow the default database to be set as a config option.
Log HTTP requests to STDOUT.

1.0.0
-----

Switch to using OAuth Refresh Tokens for authentication, so users don't have to
authenticate so often.
Require client and server versions to be identical to cooperate. This should
make it easier to handle breaking changes, at the expense of requiring more
frequent client upgrades.

0.1.4
-----

Fix a bug where uploaded database archives were not being deleted.
Minor fix to CLI output formatting

0.1.3
-----

Cleanup compressed upload after extraction

0.1.2
-----

Add quick start example to cli help

0.1.1
-----

Add `new` command to client
- This is a shortcut to create an instance of the latest image.

0.1.0
-----

First release
