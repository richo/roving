package main

import (
	"flag"
	"io/ioutil"
	"log"
	"time"

	"github.com/richo/roving/server"
	"github.com/richo/roving/types"
)

func main() {
	var portArg int
	flag.IntVar(
		&portArg,
		"port",
		1414,
		"The port the roving server should listen on")

	var workdirArg string
	flag.StringVar(
		&workdirArg,
		"workdir",
		"",
		"The afl workdir the roving server should store inputs and outputs in")

	var metricsReportIntervalArg time.Duration
	flag.DurationVar(
		&metricsReportIntervalArg,
		"metrics-report-interval",
		0,
		"The interval at which metrics should be reported to the external metrics service")

	var fuzzerSyncIntervalArg time.Duration
	flag.DurationVar(
		&fuzzerSyncIntervalArg,
		"fuzzer-sync-interval",
		300*time.Second,
		"The interval at which clients should sync their work with the server")

	var memLimitMbArg int
	flag.IntVar(
		&memLimitMbArg,
		"mem-limit-mb",
		0,
		"The AFL memory limit, in MB")

	var timeoutMsArg int
	flag.IntVar(
		&timeoutMsArg,
		"timeout-ms",
		0,
		"The AFL timeout period, in ms")

	var archiveTypeArg string
	flag.StringVar(
		&archiveTypeArg,
		"archive-type",
		"",
		"The type of work archival to run")

	var archiveIntervalArg time.Duration
	flag.DurationVar(
		&archiveIntervalArg,
		"archive-interval",
		0,
		"The interval at which to archive work")

	var archiveDiskRootArg string
	flag.StringVar(
		&archiveDiskRootArg,
		"archive-disk-root",
		"",
		"The root folder at which to archive to disk")

	var archiveS3RootKeyArg string
	flag.StringVar(
		&archiveS3RootKeyArg,
		"archive-s3-root-key",
		"",
		"The root key at which to archive to S3")

	var archiveS3BucketArg string
	flag.StringVar(
		&archiveS3BucketArg,
		"archive-s3-bucket",
		"",
		"The S3 bucket to archive to")

	var archiveS3AWSRegionArg string
	flag.StringVar(
		&archiveS3AWSRegionArg,
		"archive-s3-aws-region",
		"",
		"The S3 AWS region to archive to")

	var binaryPathArg string
	flag.StringVar(
		&binaryPathArg,
		"binary-path",
		"",
		"The path of the binary to fuzz")

	var useDictArg bool
	flag.BoolVar(
		&useDictArg,
		"use-dict",
		false,
		"Whether an AFL dictionary should be used. Name it dict.txt and place it alongside the input/ and output/ folders.")

	flag.Parse()

	command := flag.Args()

	useBinary := (binaryPathArg != "")

	fuzzerConf := types.FuzzerConfig{
		UseBinary:    useBinary,
		UseDict:      useDictArg,
		SyncInterval: fuzzerSyncIntervalArg,
		Command:      command,
		MemLimitMb:   memLimitMbArg,
		TimeoutMs:    timeoutMsArg,
	}

	archiveS3Conf := types.S3ArchiveConfig{
		RootKey:    archiveS3RootKeyArg,
		BucketName: archiveS3BucketArg,
		AwsRegion:  archiveS3AWSRegionArg,
	}
	archiveDiskConf := types.DiskArchiveConfig{
		DstRoot: archiveDiskRootArg,
	}
	archiveConf := types.ArchiveConfig{
		Type:     archiveTypeArg,
		Interval: archiveIntervalArg,
		Disk:     archiveDiskConf,
		S3:       archiveS3Conf,
	}

	conf := types.ServerConfig{
		Port:                  portArg,
		Workdir:               workdirArg,
		BinaryPath:            binaryPathArg,
		MetricsReportInterval: metricsReportIntervalArg,
		Fuzzer:                fuzzerConf,
		Archive:               archiveConf,
	}

	err := conf.ValidateConfig()
	if err != nil {
		log.Fatal(err)
	}

	var targetBinary types.TargetBinary

	if conf.BinaryPath != "" {
		targetBinary, err = ioutil.ReadFile(conf.BinaryPath)
		if err != nil {
			log.Panicf("Couldn't load target binary")
		}
	}

	archiveConfig := conf.Archive
	fuzzerConfig := conf.Fuzzer

	log.Printf("----TARGET DETAILS-----")
	log.Printf("Use binary?:\t%t", fuzzerConfig.UseBinary)
	log.Printf("Binary size: %d", len(targetBinary))
	log.Printf("Target command:\t%s", fuzzerConfig.Command)
	log.Printf("Sync interval:\t%ds", fuzzerConfig.SyncInterval/time.Second)
	log.Printf("Workdir:\t%s", conf.Workdir)
	log.Printf("Dictionary:\t%t", fuzzerConfig.UseDict)

	log.Printf("Archive type:\t%s", archiveConfig.Type)
	switch archiveConfig.Type {
	case "disk":
		log.Printf("Archive Interval:\t%s", archiveConfig.Interval)
		log.Printf("Archive Dst:\t%s", archiveConfig.Disk.DstRoot)
	case "s3":
		log.Printf("Archive Interval:\t%s", archiveConfig.Interval)
		log.Printf("Archive Region:\t%s", archiveConfig.S3.AwsRegion)
		log.Printf("Archive Bucket:\t%s", archiveConfig.S3.BucketName)
		log.Printf("Archive Root Key:\t%s", archiveConfig.S3.RootKey)
		log.Printf("Archive Is Local?:\t%t", archiveConfig.S3.IsLocal)
	default:
		log.Printf("Output archiving disabled")
	}
	log.Printf("--------")

	server.SetupAndServe(
		conf.Port,
		targetBinary,
		fuzzerConfig,
		archiveConfig,
		conf.MetricsReportInterval,
		conf.Workdir,
	)
}
