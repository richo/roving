AFL ?= ~/afls/afl-1.83b/

server/server: $(wildcard server/*.go)
	cd server && go build

target: example/target.c
	AFL_HARDEN=1 $(AFL)/afl-clang -o $@ $<

# Debug pretty printer
print-%: ; @echo $*=$($*)
