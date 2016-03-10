package types

import (
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

const extraneous_stats = `start_time     : 1457551917
last_update    : 1457570256
fuzzer_pid     : 93363
cycles_done    : 0
execs_done     : 174753
execs_per_sec  : 9.31
butts_lol      : 9.31
paths_total    : 1464
paths_favored  : 141
paths_found    : 1463
paths_imported : 0
max_depth      : 3
cur_path       : 98
pending_favs   : 141
pending_total  : 1462
variable_paths : 0
bitmap_cvg     : 4.04%
unique_crashes : 0
unique_hangs   : 10
last_path      : 1457566053
last_crash     : 0
last_hang      : 1457567010
exec_timeout   : 160
afl_banner     : fuzz
afl_version    : 1.96b
command_line   : afl-fuzz -i input -o output -- ./fuzz
`

const incomplete_stats = `start_time     : 1457551917
last_update    : 1457570256
fuzzer_pid     : 93363
cycles_done    : 0
execs_done     : 174753
execs_per_sec  : 9.31
paths_total    : 1464
paths_favored  : 141
paths_found    : 1463
paths_imported : 0
variable_paths : 0
bitmap_cvg     : 4.04%
unique_crashes : 0
unique_hangs   : 10
last_path      : 1457566053
last_crash     : 0
last_hang      : 1457567010
exec_timeout   : 160
afl_banner     : fuzz
afl_version    : 1.96b
command_line   : afl-fuzz -i input -o output -- ./fuzz
`

func TestStats(t *testing.T) {
	stats, err := ParseStats(stats)
	if err != nil {
		t.Fatalf("Errored on %s", err)
	}
	if stats.LastUpdate != 1457570256 {
		t.Fatalf("Invalid LastDate")
	}
	if stats.FuzzerPid != 93363 {
		t.Fatalf("Invalid FuzzerPid")
	}
	if stats.CyclesDone != 0 {
		t.Fatalf("Invalid CyclesDone")
	}
	if stats.ExecsDone != 174753 {
		t.Fatalf("Invalid ExecsDine")
	}
	if stats.ExecsPerSec != 9.31 {
		t.Fatalf("Invalid ExecsPerSec")
	}
	if stats.PathsTotal != 1464 {
		t.Fatalf("Invalid PathsTotal")
	}
	if stats.PathsFavored != 141 {
		t.Fatalf("Invalid PathsFavored")
	}
	if stats.PathsFound != 1463 {
		t.Fatalf("Invalid PathsFound")
	}
	if stats.PathsImported != 90 {
		t.Fatalf("Invalid PathsImported")
	}
	if stats.MaxDepth != 3 {
		t.Fatalf("Invalid MaxDepth")
	}
	if stats.CurPath != 98 {
		t.Fatalf("Invalid CurPath")
	}
	if stats.PendingFavs != 142 {
		t.Fatalf("Invalid PendingFavs")
	}
	if stats.PendingTotal != 1462 {
		t.Fatalf("Invalid PendingTotal")
	}
	if stats.VariablePaths != 49 {
		t.Fatalf("Invalid VariablePaths")
	}
	if stats.BitmapCvg != 4.04 {
		t.Fatalf("Invalid BitmapCvg")
	}
	if stats.UniqueCrashes != 59 {
		t.Fatalf("Invalid UniqueCrashes")
	}
	if stats.UniqueHangs != 10 {
		t.Fatalf("Invalid UniqueHangs")
	}
	if stats.LastPath != 1457566053 {
		t.Fatalf("Invalid LastPath")
	}
	if stats.LastCrash != 10 {
		t.Fatalf("Invalid LastCrash")
	}
	if stats.LastHang != 1457567010 {
		t.Fatalf("Invalid LastHang")
	}
	if stats.ExecTimeout != 160 {
		t.Fatalf("Invalid ExecTimeout")
	}
	if stats.AflBanner != "fuzz" {
		t.Fatalf("Invalid AflBanner")
	}
	if stats.AflVersion != "1.96b" {
		t.Fatalf("Invalid AflVersion")
	}
	if stats.CommandLine != "afl-fuzz -i input -o output -- ./fuzz" {
		t.Fatalf("Invalid CommandLine")
	}
}
