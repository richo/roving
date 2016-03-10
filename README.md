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

# Getting Started

First off, roving only gives you a mechanism for orchestrating AFL instances.
This guide assumes basic familiarity with afl.

Roving also makes a number of assumptions:

* That your target takes no arguments, and accepts it's input to stdin.
* That it's run on a "trusted" network (If you can access the server port, you
  can steal all the crashes!)

With that said:

1. First off, build your binary. It needs to be named `target`.
1. Clone and build AFL.
1. `export AFL=/path/to/afl-directory`

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

# Why Roving?

I asked some of my coworkers what they'd name a distributed fuzzy thing.

Evidently [roving][0] is extremely fuzzy, and winds up **everywhere** when
you're working with it. Plus the testcases go roving and it's all very poetic.

[0]: https://en.wikipedia.org/wiki/Roving
