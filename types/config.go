package types

import (
	"errors"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type ServerConfig struct {
	Port                  int           `yaml:"port"`
	Workdir               string        `yaml:"workdir"`
	BinaryPath            string        `yaml:"binary_path"`
	MetricsReportInterval time.Duration `yaml:"metrics_report_interval"`

	Fuzzer  FuzzerConfig  `yaml:"fuzzer"`
	Archive ArchiveConfig `yaml:"archive"`
}

// A FuzzerConfig is initially constructed from a config file by
// roving-srv. roving-client retrieves it from roving-srv over HTTP.
type FuzzerConfig struct {
	UseBinary    bool          `yaml:"use_binary"`
	UseDict      bool          `yaml:"use_dict"`
	SyncInterval time.Duration `yaml:"sync_interval"`

	Command    []string `yaml:"command"`
	MemLimitMb int      `yaml:"mem_limit_mb"`
	TimeoutMs  int      `yaml:"timeout_ms"`
}

type ArchiveConfig struct {
	Type     string            `yaml:"type"`
	Interval time.Duration     `yaml:"interval"`
	Disk     DiskArchiveConfig `yaml:"disk"`
	S3       S3ArchiveConfig   `yaml:"s3"`
}

type DiskArchiveConfig struct {
	DstRoot string `yaml:"dst_root"`
}

type S3ArchiveConfig struct {
	RootKey    string `yaml:"root_key"`
	BucketName string `yaml:"bucket_name"`
	AwsRegion  string `yaml:"aws_region"`
	IsLocal    bool   `yaml:"is_local"`
}

func (r *ServerConfig) ValidateConfig() error {
	err := r.makePathsAbsolute()
	if err != nil {
		return err
	}

	if r.Workdir == "" {
		return errors.New("Must specify workdir!")
	}

	if r.Fuzzer.UseBinary && len(r.Fuzzer.Command) > 0 {
		return errors.New("Can only specify target_command if binary_path is not set")
	}

	switch r.Archive.Type {
	case "disk":
		archiveDst := r.Archive.Disk.DstRoot
		if archiveDst == "" {
			log.Fatal("Must specify dst_root if archiving to disk!")
		}
	case "s3":
		if r.Archive.S3.RootKey == "" {
			return errors.New("Must specify root_key if archiving to S3!")
		}
		if r.Archive.S3.BucketName == "" {
			return errors.New("Must specify bucket_name if archiving to S3!")
		}
		if r.Archive.S3.AwsRegion == "" {
			return errors.New("Must specify aws_region if archiving to S3!")
		}
	case "":
	default:
		log.Fatalf("Unrecognized archive type: %s", r.Archive.Type)
	}

	return nil
}

// makePathsAbsolute converts paths that were specified relative to the
// current working directory to absolute paths.
func (r *ServerConfig) makePathsAbsolute() error {
	if r.Archive.Type == "disk" {
		absDst, err := filepath.Abs(r.Archive.Disk.DstRoot)
		if err != nil {
			return err
		}
		r.Archive.Disk.DstRoot = absDst
	}

	if r.Workdir != "" {
		workdir, err := filepath.Abs(r.Workdir)
		if err != nil {
			return err
		}
		r.Workdir = workdir
	}

	return nil
}

// A ClientConfig is loaded from a config file on the client itself,
// unlike FuzzerConfig.
type ClientConfig struct {
	ServerAddress string `yaml:"server_address"`
	Parallelism   int    `yaml:"parallelism"`
}

func (r *ClientConfig) ValidateConfig() error {
	r.canonicalizeServerAddress()
	r.setDefaultParallelism()

	if r.ServerAddress == "" {
		return errors.New("Must specify server_address")
	}

	return nil
}

// canonicalizeServerAddress ensures that the server address has
// an "http://" prefix. Note that roving does not currently support
// HTTPS.
func (r *ClientConfig) canonicalizeServerAddress() {
	httpPrefix := "http://"
	if !strings.HasPrefix(r.ServerAddress, httpPrefix) {
		r.ServerAddress = httpPrefix + r.ServerAddress
	}
}

// setDefaultParallelism sets the client's parallelism to
// the number of CPUs on the machine if the value from the config
// is -1.
func (r *ClientConfig) setDefaultParallelism() {
	if r.Parallelism == -1 {
		numCPU := runtime.NumCPU()
		log.Printf("Parallelism not set - defaulting to %d (num CPUs)", numCPU)

		r.Parallelism = numCPU
	}
}
