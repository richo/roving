package server

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"

	"github.com/richo/roving/types"
)

func writeFile(path, contents string) {
	var err error

	dirs := filepath.Dir(path)
	if err = os.MkdirAll(dirs, 0755); err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintf(w, contents)
	w.Flush()
}

func readFile(path string) string {
	contentsBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return string(contentsBytes)
}

func TestDiskArchiverNamedDir(t *testing.T) {
	var err error

	srcDir, err := ioutil.TempDir("", "roving-archiver-test-src")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := ioutil.TempDir("", "roving-archiver-test-dst")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dstDir)

	hiTherePath := filepath.Join(srcDir, "hi", "there")
	goodbyePath := filepath.Join(srcDir, "goodbye")
	writeFile(hiTherePath, "hi there\n")
	writeFile(goodbyePath, "goodbye\n")

	config := types.ArchiveConfig{Disk: types.DiskArchiveConfig{DstRoot: dstDir}}
	archiver, err := NewDiskArchiver(config)
	if err != nil {
		log.Fatal(err)
	}

	err = ArchiveToNamedDir(archiver, srcDir, ".")
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, "hi there\n", readFile(hiTherePath))
	assert.Equal(t, "goodbye\n", readFile(goodbyePath))
}

// XXX: s3iface.S3API won't allow me to make PutObject have a pointer
// receiver, so the easiest way to track inputs is using this global
// variable.
var putInputs []*s3.PutObjectInput
var putBodies []string

func resetS3stubs() {
	putInputs = []*s3.PutObjectInput{}
	putBodies = []string{}
}

type mockS3Client struct {
	s3iface.S3API
}

func (m mockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	putInputs = append(putInputs, input)

	bodyBytes, err := ioutil.ReadAll(input.Body)
	if err != nil {
		log.Fatal(err)
	}
	putBodies = append(putBodies, string(bodyBytes))

	return nil, nil
}

// XXX: Doesn't respect the contents of the ListObjectsInput, just returns
// everything that has been put-ed.
func (m mockS3Client) ListObjectsPages(input *s3.ListObjectsInput, fn func(*s3.ListObjectsOutput, bool) bool) error {
	for _, input := range putInputs {
		filename := filepath.Base(*input.Key)
		output := s3.ListObjectsOutput{Contents: []*s3.Object{
			&s3.Object{Key: &filename},
		}}
		fn(&output, false)
	}
	return nil
}

func TestS3ArchiverNamedDir(t *testing.T) {
	var err error
	resetS3stubs()

	srcDir, err := ioutil.TempDir("", "roving-archiver-test-src")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	hiTherePath := filepath.Join(srcDir, "hi", "there")
	goodbyePath := filepath.Join(srcDir, "goodbye")
	writeFile(hiTherePath, "hi there\n")
	writeFile(goodbyePath, "goodbye\n")

	s3client := mockS3Client{}
	archiver := S3Archiver{
		bucketName: "test-bucket",
		rootKey:    "data/more-data",
		s3client:   s3client,
	}
	err = ArchiveToNamedDir(archiver, srcDir, ".")
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(t, 2, len(putBodies))

	assert.Equal(t, "test-bucket", *putInputs[0].Bucket)
	assert.Equal(t, "data/more-data/goodbye", *putInputs[0].Key)
	assert.Equal(t, "goodbye\n", putBodies[0])

	assert.Equal(t, "test-bucket", *putInputs[1].Bucket)
	assert.Equal(t, "data/more-data/hi/there", *putInputs[1].Key)
	assert.Equal(t, "hi there\n", putBodies[1])
}

func TestS3ArchiverTimestampedDir(t *testing.T) {
	getTimestamp = func() int64 {
		return 4815162342
	}
	var err error
	resetS3stubs()

	srcDir, err := ioutil.TempDir("", "roving-archiver-test-src")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	hiTherePath := filepath.Join(srcDir, "hi", "there")
	goodbyePath := filepath.Join(srcDir, "goodbye")
	writeFile(hiTherePath, "hi there\n")
	writeFile(goodbyePath, "goodbye\n")

	s3client := mockS3Client{}
	archiver := S3Archiver{
		bucketName: "test-bucket",
		rootKey:    "data/more-data",
		s3client:   s3client,
	}

	err = archiveToTimestampedDir(archiver, srcDir)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(t, 2, len(putBodies))

	assert.Equal(t, "test-bucket", *putInputs[0].Bucket)
	assert.Equal(t, "data/more-data/4815162342/goodbye", *putInputs[0].Key)
	assert.Equal(t, "goodbye\n", putBodies[0])

	assert.Equal(t, "test-bucket", *putInputs[1].Bucket)
	assert.Equal(t, "data/more-data/4815162342/hi/there", *putInputs[1].Key)
	assert.Equal(t, "hi there\n", putBodies[1])
}

func TestS3ArchiverLsDstFiles(t *testing.T) {
	var err error
	resetS3stubs()

	srcDir, err := ioutil.TempDir("", "roving-archiver-test-src")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	bonjourPath := filepath.Join(srcDir, "bonjour")
	goodbyePath := filepath.Join(srcDir, "goodbye")
	writeFile(bonjourPath, "bonjour\n")
	writeFile(goodbyePath, "goodbye\n")

	s3client := mockS3Client{}
	archiver := S3Archiver{
		bucketName: "test-bucket",
		rootKey:    "data/more-data",
		s3client:   s3client,
	}

	if err = ArchiveToNamedDir(archiver, srcDir, ""); err != nil {
		log.Fatal(err)
	}
	assert.Equal(t, 2, len(putBodies))

	res, err := archiver.LsDstFiles(".")
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(t, []string{"bonjour", "goodbye"}, res)
}
