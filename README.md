Distributed Afl
===============

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
