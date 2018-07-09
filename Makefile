all: server/server client/client

server/server: $(wildcard server/*.go) $(wildcard types/*.go)
	cd server && go build

client/client: $(wildcard client/*.go) $(wildcard types/*.go)
	cd client && go build

example-target: check-env
	AFL_HARDEN=1 $(AFL)/afl-clang -o example/server/$@ example/server/target.c

example-server: server/server
	$(CURDIR)/server/server $(CURDIR)/example/server

example-client: client/client
	cd $(CURDIR)/example/client && rm -rf work && ../../client/client 127.0.0.1:8000

# Debug pretty printer
print-%: ; @echo $*=$($*)

check-env:
ifndef AFL
	$(error please set the AFL env var to the path to your afl repo)
endif

.PHONY: all serve testing
