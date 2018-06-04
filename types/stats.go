package types

import (
	"fmt"
	"strconv"
	"strings"
)

type FuzzerStats struct {
	StartTime     uint64
	LastUpdate    uint64
	FuzzerPid     uint64
	CyclesDone    uint64
	ExecsDone     uint64
	ExecsPerSec   float64
	PathsTotal    uint64
	PathsFavored  uint64
	PathsFound    uint64
	PathsImported uint64
	MaxDepth      uint64
	CurPath       uint64
	PendingFavs   uint64
	PendingTotal  uint64
	VariablePaths uint64
	BitmapCvg     float64
	UniqueCrashes uint64
	UniqueHangs   uint64
	LastPath      uint64
	LastCrash     uint64
	LastHang      uint64
	ExecTimeout   uint64
	AflBanner     string
	AflVersion    string
	CommandLine   string
}

func ParseStats(stats string) (*FuzzerStats, error) {
	var fields_covered uint = 0
	// Storage for eventual values
	var start_time uint64
	var last_update uint64
	var fuzzer_pid uint64
	var cycles_done uint64
	var execs_done uint64
	var execs_per_sec float64
	var paths_total uint64
	var paths_favored uint64
	var paths_found uint64
	var paths_imported uint64
	var max_depth uint64
	var cur_path uint64
	var pending_favs uint64
	var pending_total uint64
	var variable_paths uint64
	var bitmap_cvg float64
	var unique_crashes uint64
	var unique_hangs uint64
	var last_path uint64
	var last_crash uint64
	var last_hang uint64
	var exec_timeout uint64
	var afl_banner string
	var afl_version string
	var command_line string
	// I don't want to use := anywhere below because I have no idea what
	// it'll do to these vars, thus we allocate the error handle now
	var err error

	lines := strings.Split(stats, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		values := strings.SplitN(line, ":", 2)
		key, value := values[0], values[1]
		key = strings.Trim(key, " ")
		value = strings.Trim(value, " ")

		switch key {
		case "start_time":
			start_time, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 0
		case "last_update":
			last_update, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 1
		case "fuzzer_pid":
			fuzzer_pid, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 2
		case "cycles_done":
			cycles_done, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 3
		case "execs_done":
			execs_done, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 4
		case "execs_per_sec":
			execs_per_sec, err = strconv.ParseFloat(value, 64)
			fields_covered |= 1 << 5
		case "paths_total":
			paths_total, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 6
		case "paths_favored":
			paths_favored, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 7
		case "paths_found":
			paths_found, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 8
		case "paths_imported":
			paths_imported, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 9
		case "max_depth":
			max_depth, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 10
		case "cur_path":
			cur_path, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 11
		case "pending_favs":
			pending_favs, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 12
		case "pending_total":
			pending_total, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 13
		case "variable_paths":
			variable_paths, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 14
		case "bitmap_cvg":
			value = strings.Trim(value, "%")
			bitmap_cvg, err = strconv.ParseFloat(value, 64)
			fields_covered |= 1 << 15
		case "unique_crashes":
			unique_crashes, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 16
		case "unique_hangs":
			unique_hangs, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 17
		case "last_path":
			last_path, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 18
		case "last_crash":
			last_crash, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 19
		case "last_hang":
			last_hang, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 20
		case "exec_timeout":
			exec_timeout, err = strconv.ParseUint(value, 10, 64)
			fields_covered |= 1 << 21
		case "afl_banner":
			afl_banner = value
			fields_covered |= 1 << 22
		case "afl_version":
			afl_version = value
			fields_covered |= 1 << 23
		case "command_line":
			command_line = value
			fields_covered |= 1 << 24
			// since latest version of afl add some new keywords such as 'stability'.
			// patch out this two line can keep roving compatibled the latest vesion of afl.
			//	default:
			//		return nil, fmt.Errorf("Unexpected key: %s", key)
		}
		if err != nil {
			return nil, fmt.Errorf("Invalid value for %s: %s", key, value)
		}
	}

	// TODO: Actually have a lookup and tell the user which fields were
	// missing
	if 33554431 != fields_covered {
		return nil, fmt.Errorf("Missing fields in stats")
	}

	return &FuzzerStats{
		StartTime:     start_time,
		LastUpdate:    last_update,
		FuzzerPid:     fuzzer_pid,
		CyclesDone:    cycles_done,
		ExecsDone:     execs_done,
		ExecsPerSec:   execs_per_sec,
		PathsTotal:    paths_total,
		PathsFavored:  paths_favored,
		PathsFound:    paths_found,
		PathsImported: paths_imported,
		MaxDepth:      max_depth,
		CurPath:       cur_path,
		PendingFavs:   pending_favs,
		PendingTotal:  pending_total,
		VariablePaths: variable_paths,
		BitmapCvg:     bitmap_cvg,
		UniqueCrashes: unique_crashes,
		UniqueHangs:   unique_hangs,
		LastPath:      last_path,
		LastCrash:     last_crash,
		LastHang:      last_hang,
		ExecTimeout:   exec_timeout,
		AflBanner:     afl_banner,
		AflVersion:    afl_version,
		CommandLine:   command_line,
	}, nil
}
