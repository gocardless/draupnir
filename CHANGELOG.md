Changelog
=========

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
