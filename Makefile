server/server: $(wildcard server/*.go)
	cd server && go build

# Debug pretty printer
print-%: ; @echo $*=$($*)
