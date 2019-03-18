Roving
======

Distributed fuzzing using AFL.

# Overview

## AFL

AFL is a "fuzzer". You give it a target program, and it runs that target
program zillions of times, trying to find input that causes it to crash.

It uses instrumentation of the target program's code to try to manipulate its
input so that it explores as much of the target program as possible.

## Roving

A roving cluster runs multiple copies of AFL on multiple machines, all
fuzzing the same target. Roving's key contribution is to allow these
machines to share and benefit from each other's work. If machine A finds
an "interesting" test case that causes a new function to get invoked,
machines B, C and D can all use this discovery to explore the rest of
the program more efficiently.

### Cluster structure

A roving cluster consists of 1 server and N clients. Each client runs M
copies of AFL (using AFL's existing parallelism settings), and uses the
server to share their work with their peers. Each fuzzer on the client
periodically (by default, every 5 mins) uploads to the server their
current AFL state, including their queue. The server saves these states
in memory.

Fuzzers take advantage of the work of their peers by downloading
from the server the state of all clients in the cluster. They replace
their current queue with the combined queues of all clients, and then
continue fuzzing as before. This allows all clients to benefit from the
new, interesting testcases that any individual client discovers.

This approach relies on the non-determinism of AFL. If every client
deterministically ran the same test cases when given the same queue,
we would simply be repeating the same work N times across N different
clients. In reality, clients take the same queue and run in wildly
different directions with it. This means that we cover more of the search
space, faster.

That said, there is no formal partitioning of work, and there *will* be
some amount of duplication of work between clients. We do not currently
have any estimates of how much work is duplicated, but it is safe to say
that running 10 roving clients will not get you 10x the edge-discovery
rate of 1 client. Roving uses the same principle as AFL's own
single-machine parallelism, so we still have good reason to believe
that it is effective.

# Usage

## Bazel

For now roving uses [Bazel][https://docs.bazel.build/versions/master/install.html]
for its build. You'll need to download it in order to build roving.

## Roving Server

* Export `AFL` with the path to afl, or make sure `afl-fuzz` is on `PATH`
* In the workdir, create a `target` binary [optional]
* In the workdir, make a directory called `input` and populate it with a corpus
* Run `bazel build //cmd/srv`
* Run `bazel-bin/cmd/srv/darwin_amd64_stripped/srv`

Once up, it will create a directory called `output` that mirrors the
structure of the `output` directory created by AFL. It will aggregate
crashes, hangs, and the queue.

There is also a basic (but improving!) admin page at `SERVER_URL:SERVER_PORT/admin`.

## Roving Clients

Clients should require almost no configuration.

* Run `bazel build //cmd/client`
* Run `bazel-bin/cmd/client/darwin_amd64_stripped/client -- -server-hostport XYZ:123 -parallelism X`

Clients will accumulate crashes and hangs in their working dir. They will
sync them to the server.

## Advanced usage

Run the compiled binaries with the `-help` flag or see the files in the `cmd/`
folder for advanced options.

# Development

## Tests

The test suite is not particularly extensive, but you can run it
using:

```
bin/test
```

## Design principles

### Roving clients should be very dumb

Roving clients should be very dumb and have very
little configuration. This is so that clients can easily
be brought up, pointed at any roving server of any type, and
quickly start working.

If a roving server requires clients to be configured
in a particular way (perhaps the server wants them
to sync their work with it more frequently than normal),
this should be passed as configuration to the *server*,
which should then send it to the client when it starts up
and joins the cluster.

### Fuzzer-agnosticism is good but currently not essential

We would like roving to be fuzzer-agnostic in the future. It should be
possible to power your fuzzing using `afl`, `libfuzzer`, `hongfuzz`, or
any other reasonable fuzzer.

All of these fuzzers work in somewhat different ways and have somewhat
different structures and opinions. We are comfortable loosely coupling
ourselves to `afl` for now - for example, we assume that fuzzer input and
output is structured in the way that `afl` expects. However, we would like
multi-fuzzer support to be an achievable goal in the future, and would like
to avoid making decisions that would make this unreasonably difficult.

## Running the examples

The example code bash scripts live in the `examples/` directory.

### C

* `examples/c-server` to build the target and run the example server serving the C example target on the default port 1414
* `examples/generic-client` to run the example client

Your client should find a crash within 30 seconds.

### Ruby

* [Install `afl-ruby`][afl-ruby]
* `examples/ruby-server` to run the example server serving the Ruby example target on the default port 1414
* `examples/generic-client` to run the example client

Your client should again find a crash within 30 seconds.

# Why Roving?

I asked some of my coworkers what they'd name a distributed fuzzy thing.

Evidently [roving][0] is extremely fuzzy, and winds up **everywhere** when
you're working with it. Plus the testcases go roving and it's all very poetic.

# Credit

* [Stripe][stripe] has substantially contributed to Roving, by directly supporting its development in paid time, as well as contributing that development back to the open source project.
* [Rob Heaton][robert] spent huge amounts of time adding features, finding bugs, and documenting the project.


[0]: https://en.wikipedia.org/wiki/Roving
[stripe]: https://stripe.com
[robert]: https://github.com/robert
[afl-ruby]: https://github.com/richo/afl-ruby
