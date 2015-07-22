AFL ?= ~/afls/afl-1.83b/

all: server/server client/client

server/server: $(wildcard server/*.go)
	cd server && go build

client/client: $(wildcard client/*.go)
	cd client && go build

serve: server/server
	$(CURDIR)/server/server

target: example/target.c
	AFL_HARDEN=1 $(AFL)/afl-clang -o $@ $<

# Debug pretty printer
print-%: ; @echo $*=$($*)

.PHONY: all serve testing
