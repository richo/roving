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
* `./server/server /path/to/dir/with-target-and-input`

On each of your clients
* `./client/client name-of.server.com:port`
* Do this once for each core

* find crashes and hangs in `work-server`

In theory that's it!

This is super lightly tested, YMMV, patches/bug reports accepted, etc.

# Running the examples

The example code lives in the `example/` directory.

## C

* `make example-target` to compile the example target
* `make example-server-c` to run the example server serving the C example target on the default port 8000
* `make example-client` to run the example client

Your client should find a crash within 30 seconds.

## Ruby

* `bundle install --gemfile example/client/ruby/Gemfile` to install the `afl` gem
* `make example-server-ruby` to run the example server serving the Ruby example target on the default port 8000
* `make example-client` to run the example client

Your client should again find a crash within 30 seconds.

# Why Roving?

I asked some of my coworkers what they'd name a distributed fuzzy thing.

Evidently [roving][0] is extremely fuzzy, and winds up **everywhere** when
you're working with it. Plus the testcases go roving and it's all very poetic.

[0]: https://en.wikipedia.org/wiki/Roving
