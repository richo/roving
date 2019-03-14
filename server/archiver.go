package server

import (
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	raven "github.com/getsentry/raven-go"

	"github.com/richo/roving/types"
)

// Allow overriding `getTimestamp` in tests
var getTimestamp func() int64

func init() {
	getTimestamp = func() int64 {
		return time.Now().Unix()
	}
}

// ArchiveToTimestampedDirsForever uses an `archiver` to repeatedly
// archive `absSrcPath` every `interval`. Each archive preserves
// the original directory structure, and is saved at
// `${archiver.DstRoot}/$TIMESTAMP/`.
//
// This function never returns.
func ArchiveToTimestampedDirsForever(archiver Archiver, absSrcPath string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			log.Printf("Archiving to timestamped dir src=%s dst=%s", absSrcPath, archiver.DescribeDstRoot())
			err := archiveToTimestampedDir(archiver, absSrcPath)
			if err != nil {
				raven.CaptureErrorAndWait(err, nil)
				log.Print(err)
			}
		}
	}
}

// ArchiveToNamedDir uses an `archiver` to archive `absSrcPath`
// once. The archive preserves the original directory structure, and
// is saved at `dstSubRoot`.
func ArchiveToNamedDir(a Archiver, absSrcPath, dstSubRoot string) error {
	manifest := newSimpleManifest(absSrcPath, dstSubRoot)
	return ArchiveManifest(a, manifest)
}

// ArchiveManifest uses an `archiver` to archive a `manifest` once.
func ArchiveManifest(a Archiver, manifest Manifest) error {
	types.SubmitMetricCount("archive_manifest", float32(len(manifest.entries)), map[string]string{"root": manifest.srcRoot})
	for _, entry := range manifest.entries {
		absSrcPath := filepath.Join(manifest.srcRoot, entry.src)
		relDstPath := entry.dst
		err := a.archiveOne(absSrcPath, relDstPath)

		if err != nil {
			types.SubmitMetricCount("archive_one.fail", 1, map[string]string{"srcRoot": manifest.srcRoot})
			log.Print(err)
			raven.CaptureError(err, map[string]string{
				"dst":     entry.dst,
				"src":     entry.src,
				"srcRoot": manifest.srcRoot,
			})
			continue
		}
		types.SubmitMetricCount("archive_one.success", 1, map[string]string{"srcRoot": manifest.srcRoot})
	}
	return nil
}

// archiveToTimestampedDir archives `absSrcPath` to a timestamped dir once.
func archiveToTimestampedDir(a Archiver, absSrcPath string) error {
	ts := getTimestamp()
	tsStr := strconv.FormatInt(ts, 10)

	return ArchiveToNamedDir(a, absSrcPath, tsStr)
}

// Roving servers use `Archiver`s to regularly copy
// their output directory to backup storage. This
// prevents us from losing valuable work if the server
// goes down.
//
// The Archiver interface allows us to implement archivers
// that write to different types of storage.
type Archiver interface {
	// Archive the local dir `absSrcPath` to `relDstPath`
	archiveOne(absSrcPath, relDstPath string) error
	// Lists all filenames in `relDstRoot`
	LsDstFiles(relDstRoot string) ([]string, error)

	// Describes the dst's root path. For display only.
	DescribeDstRoot() string
	// Describes a relative dst path. For display only.
	DescribeDstLoc(relDstPath string) string
}

// An archiver that conforms to the Archiver interface
// but does nothing.
type NullArchiver struct{}

func (a NullArchiver) archiveOne(absSrcPath, relDstPath string) error {
	return nil
}
func (a NullArchiver) LsDstFiles(relDstRoot string) ([]string, error) {
	return []string{}, nil
}
func (a NullArchiver) DescribeDstRoot() string {
	return ""
}
func (a NullArchiver) DescribeDstLoc(relDstPath string) string {
	return ""
}

// A basic Archiver that copies the source to another
// location on local disk. Mostly useful for testing.
type DiskArchiver struct {
	DstRoot string
}

// archiveOne copies the file at `absSrcPath` to `relDstPath`.
// `relDstPath` is resolved relative to the current working dir.
func (a DiskArchiver) archiveOne(absSrcPath, relDstPath string) error {
	var err error
	var srcFd *os.File
	var dstFd *os.File

	absDstPath := filepath.Join(a.DstRoot, relDstPath)

	dirs := filepath.Dir(absDstPath)
	if err = os.MkdirAll(dirs, 0755); err != nil {
		return err
	}

	if srcFd, err = os.Open(absSrcPath); err != nil {
		return err
	}
	defer srcFd.Close()

	if dstFd, err = os.Create(absDstPath); err != nil {
		return err
	}
	defer dstFd.Close()

	if _, err = io.Copy(dstFd, srcFd); err != nil {
		return err
	}
	return nil
}

// LsDstFiles returns the names of all files in the given
// dir. It does not include directory names, and  does not
// walk sub-trees.
func (a DiskArchiver) LsDstFiles(dstSubRoot string) ([]string, error) {
	dstFullRoot := filepath.Join(a.DstRoot, dstSubRoot)
	_, err := os.Stat(dstFullRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return []string{}, err
	}

	fileInfos, err := ioutil.ReadDir(dstFullRoot)
	if err != nil {
		return []string{}, err
	}

	filenames := make([]string, 0, 0)
	for _, fileInfo := range fileInfos {
		if !fileInfo.IsDir() {
			filenames = append(filenames, filepath.Base(fileInfo.Name()))
		}
	}
	return filenames, nil
}

func (a DiskArchiver) DescribeDstRoot() string {
	return a.DstRoot
}
func (a DiskArchiver) DescribeDstLoc(dstRelPath string) string {
	return filepath.Join(a.DstRoot, dstRelPath)
}

func NewDiskArchiver(config types.ArchiveConfig) (DiskArchiver, error) {
	dstRoot := config.Disk.DstRoot
	if err := os.MkdirAll(dstRoot, 0755); err != nil {
		return DiskArchiver{}, err
	}
	return DiskArchiver{DstRoot: dstRoot}, nil
}

// S3Archiver archives files to S3.
type S3Archiver struct {
	bucketName string
	rootKey    string
	s3client   s3iface.S3API
}

func (a S3Archiver) archiveOne(absSrcPath, relDstPath string) error {
	srcFd, err := os.Open(absSrcPath)
	if err != nil {
		return err
	}

	dstKey := a.dstKey(relDstPath)

	input := &s3.PutObjectInput{
		Bucket: aws.String(a.bucketName),
		Key:    aws.String(dstKey),
		Body:   srcFd,
	}
	_, err = a.s3client.PutObject(input)
	if err != nil {
		return err
	}
	log.Printf("Archiving file to S3 bucket=%v key=%v", a.bucketName, dstKey)

	srcFd.Close()
	return nil
}

func (a S3Archiver) LsDstFiles(relDstRoot string) ([]string, error) {
	fullPrefix := filepath.Join(a.rootKey, relDstRoot)
	input := s3.ListObjectsInput{
		Bucket: aws.String(a.bucketName),
		Prefix: aws.String(fullPrefix),
	}

	filenames := make([]string, 0, 0)
	err := a.s3client.ListObjectsPages(
		&input,
		func(output *s3.ListObjectsOutput, lastPage bool) bool {
			for _, obj := range output.Contents {
				filename, err := filepath.Rel(relDstRoot, *obj.Key)
				if err != nil {
					log.Fatal(err)
				}
				filenames = append(filenames, filename)
			}
			return !lastPage
		})
	if err != nil {
		return []string{}, err
	}

	return filenames, nil
}

func (a S3Archiver) DescribeDstRoot() string {
	u := url.URL{
		Scheme: "s3",
		Path:   filepath.Join(a.bucketName, a.rootKey),
	}
	return u.String()
}
func (a S3Archiver) DescribeDstLoc(dstRelPath string) string {
	u := url.URL{
		Scheme: "s3",
		Path:   filepath.Join(a.bucketName, a.dstKey(dstRelPath)),
	}
	return u.String()
}
func (a S3Archiver) dstKey(dstRelPath string) string {
	return path.Join(a.rootKey, dstRelPath)
}

func NewS3Archiver(config types.ArchiveConfig) (S3Archiver, error) {
	rootKey := config.S3.RootKey
	bucketName := config.S3.BucketName
	awsRegion := config.S3.AwsRegion
	isLocal := config.S3.IsLocal

	if isLocal && os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		log.Fatal("No AWS access key ID found. Configure AWS access with AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
	}
	s3client := s3.New(session.Must(session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})))

	return S3Archiver{
		bucketName: bucketName,
		rootKey:    rootKey,
		s3client:   s3client,
	}, nil
}

// A Manifest represents an archiver's intention to archive
// a set of file. It consists of a `srcRoot` and a collection
// of `ManifestEntry`s. Each `ManifestEntry` is a `src` (relative
// to the `srcRoot`) and a `dst` (relative to the archiver's
// `DstRoot` field).
type Manifest struct {
	srcRoot string
	entries []ManifestEntry
}
type ManifestEntry struct {
	src string
	dst string
}

// newSimpleManifest constructs a manifest of all files in a dir.
// This is useful because it allows us to copy something that
// is as close to a point-in-time snapshot as reasonably possible.
func newSimpleManifest(srcRoot, dstSubRoot string) Manifest {
	manifest := Manifest{srcRoot: srcRoot}

	// TODO(rob): use readDir to make this more atomic
	filepath.Walk(srcRoot, func(absSrcPath string, info os.FileInfo, walkErr error) error {
		// If current node is a dir then we don't have to add anything to the manifest
		if !info.IsDir() {
			relSrcPath, err := filepath.Rel(manifest.srcRoot, absSrcPath)
			if err != nil {
				log.Print(err)
				return nil
			}
			relDstRoot := filepath.Join(dstSubRoot, relSrcPath)
			if err != nil {
				log.Print(err)
				return nil
			}

			entry := ManifestEntry{
				src: relSrcPath,
				dst: relDstRoot,
			}
			manifest.entries = append(manifest.entries, entry)
		}
		return nil
	})

	return manifest
}
