Roving
======

A Go Thing for running many AFL's.

# Server

The serve expects to be run in a directory with a few members:

### `target`

The binary to be fuzzed

### `input`

The directory with the input corpus in it

Once up, it will create a directory called `output` which should look familiar
to anyone used to using afl. It will aggregate crashes and hangs in that
directory.

# Usage

* export `AFL` with the path to afl, or make sure `afl-fuzz` is on `PATH`

* Create a `target`
* Populate a directory called `input` with a corpus
* `make`
* `./server/server`

On each of your clients
* `./client/client name-of.server.com:port`
* Do this once for each core

* find crashes and hangs in `work-server`

In theory that's it!

This is super lightly tested, YMMV, patches/bug reports accepted, etc.
