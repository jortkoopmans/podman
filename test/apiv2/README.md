API v2 tests
============

This directory contains tests for the podman version 2 API (HTTP).

Tests themselves are in files of the form 'NN-NAME.at' where NN is a
two-digit number, NAME is a descriptive name, and '.at' is just
an extension I picked.

Running Tests
=============

The main test runner is `test-apiv2`. Usage is:

    $ sudo ./test-apiv2 [NAME [...]]

...where NAME is one or more optional test names, e.g. 'image' or 'pod'
or both. By default, `test-apiv2` will invoke all `*.at` tests.

`test-apiv2` connects to *localhost only* and *via TCP*. There is
no support here for remote hosts or for UNIX sockets. This is a
framework for testing the API, not all possible protocols.

`test-apiv2` will start the service if it isn't already running.


Writing Tests
=============

The main test function is `t`. It runs `curl` against the server,
with POST parameters if present, and compares return status and
(optionally) string results from the server:

    t GET /_ping 200 OK
      ^^^ ^^^^^^ ^^^ ^^
      |   |      |   +--- expected string result
      |   |      +------- expected return code
      |   +-------------- endpoint to access
      +------------------ method (GET, POST, DELETE, HEAD)


    t POST libpod/volumes/create name=foo 201 .ID~[0-9a-f]\\{12\\}
           ^^^^^^^^^^^^^^^^^^^^^ ^^^^^^^^ ^^^ ^^^^^^^^^^^^^^^^^^^^
           |                     |        |   JSON '.ID': expect 12-char hex
           |                     |        +-- expected code
           |                     +----------- POST params
           +--------------------------------- note the missing slash

Notes:

* If the endpoint has a leading slash (`/_ping`), `t` leaves it unchanged.
If there's no leading slash, `t` prepends `/v1.40`. This is a simple
convenience for simplicity of writing tests.

* When method is POST, the argument after the endpoint must be a series
of POST arguments in the form 'key=value', separated by commas. `t` will
convert those to JSON form for passing to the server.

* The final arguments are one or more expected string results. If an
argument starts with a dot, `t` will invoke `jq` on the output to
fetch that field, and will compare it to the right-hand side of
the argument. If the separator is `=` (equals), `t` will require
an exact match; if `~` (tilde), `t` will use `expr` to compare.