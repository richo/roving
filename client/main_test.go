package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"net/http/httptest"
	"testing"
)

const stats = `start_time     : 1457551917
last_update    : 1457570256
fuzzer_pid     : 93363
cycles_done    : 0
execs_done     : 174753
execs_per_sec  : 9.31
paths_total    : 1464
paths_favored  : 141
paths_found    : 1463
paths_imported : 90
max_depth      : 3
cur_path       : 98
pending_favs   : 142
pending_total  : 1462
variable_paths : 49
bitmap_cvg     : 4.04%
unique_crashes : 59
unique_hangs   : 10
last_path      : 1457566053
last_crash     : 10
last_hang      : 1457567010
exec_timeout   : 160
afl_banner     : fuzz
afl_version    : 1.96b
command_line   : afl-fuzz -i input -o output -- ./fuzz
`

type ArbitraryFuzzCommand struct {
	execCmd *exec.Cmd
}

func newArbitraryFuzzer() Fuzzer {
	fuzzCommand := ArbitraryFuzzCommand{
		execCmd: exec.Command("ls"),
	}
	id := "foo"

	return Fuzzer{
		Id:          id,
		started:     false,
		fuzzCommand: fuzzCommand,
	}
}

func (f ArbitraryFuzzCommand) cmd() *exec.Cmd {
	return f.execCmd
}

func TestMain(t *testing.T) {
	var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `[{"Id": "123"}]`)
	}))

	fuzzer := newArbitraryFuzzer()

	relativeOutputDir := fmt.Sprintf("./output/%s/", fuzzer.Id)
	absOutputDir, _ := filepath.Abs(relativeOutputDir)
	os.MkdirAll(absOutputDir, 0755)
	defer os.RemoveAll(absOutputDir)

	writeFakeOutputData(absOutputDir, stats)

	Start(fuzzer, apiStub.URL)
}

func writeFakeOutputData(dir string, stats string) {
	os.Mkdir(fmt.Sprintf("%s/crashes", dir), 0755)
	os.Mkdir(fmt.Sprintf("%s/hangs", dir), 0755)

	writeToFile(fmt.Sprintf("%s/fuzzer_stats", dir), []byte(stats))
	writeToFile(fmt.Sprintf("%s/queue", dir), []byte{})
}

func writeToFile(path string, data []byte) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Panicf("Couldn't open %s for writing: %s", path, err)
	}
	f.Write(data)
}
