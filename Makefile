all: server/server client/client

server/server: $(wildcard server/*.go) $(wildcard types/*.go)
	cd server && go build

client/client: $(wildcard client/*.go) $(wildcard types/*.go)
	cd client && go build

example-target: check-env
	AFL_HARDEN=1 $(AFL)/afl-clang -o examples/server/$@ examples/server/target.c

example-server-c: server/server
	$(CURDIR)/server/server $(CURDIR)/examples/server

example-server-ruby: server/server
	$(CURDIR)/server/server $(CURDIR)/examples/server ~/.rbenv/versions/2.4.1/bin/ruby $(CURDIR)/examples/client/ruby/harness.rb

example-client: client/client
	cd $(CURDIR)/examples/client && rm -rf work && ../../client/client 127.0.0.1:8000

# Debug pretty printer
print-%: ; @echo $*=$($*)

check-env:
ifndef AFL
	$(error please set the AFL env var to the path to your afl repo)
endif

.PHONY: all serve testing
