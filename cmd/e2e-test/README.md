# e2e-test

This is a program that runs an entire mini-roving system and
checks that it is doing roughly the things that we expect.

This program:

* Runs 1 roving server
* Runs 3 roving clients that talk to this server
* After giving the clients time to do some work and sync it,
prints stats about how work is synced across the system

The stats look like:

```
###############
### CLIENT 0 ###
###############
---QUEUE---
N in both server and client:	 17
N in server not client:	 0
N in client not server:	 0
---CRASHES---
N in both server and client:	 1
N in server not client:	 2
N in client not server:	 0
```

### Evaluating the output

#### Queue

* `N in both server and client` should be high, because
most of each client's queue should be synced
* There may be 1 or 2 in `N in client not server`, because
clients may have found some more interesting testcases
since their last sync. Anything more than this is concerning
* There may also be 1 or 2 in `N in server not client`, because
a client may have synced its before other clients synced their's.
Anything more than this is concerning.

#### Crashes

In practice, you will almost always see:

```
N in both server and client:	 1 or 2
N in server not client:	 2
N in client not server:	 0
```

TODO: better evaluation of crash syncing.
