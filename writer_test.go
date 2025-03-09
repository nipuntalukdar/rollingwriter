package rollingwriter

import (
	"io"
	"math/rand"
	"os"
	"testing"
)

func clean() {
	os.Remove("./test/unittest.log")
	os.Remove("./test/unittest.reopen")
	os.Remove("./test/unittest.gz")
	os.Remove("./test")
}

func newWriter() *Writer {
	cfg := NewDefaultConfig()
	cfg.LogPath = "./test"
	cfg.FileName = "unittest"
	w, _ := NewWriterFromConfig(&cfg)
	return w.(*Writer)
}

func newVolumeWriter() *Writer {
	cfg := NewDefaultConfig()
	cfg.RollingPolicy = 3
	cfg.RollingVolumeSize = "1mb"
	cfg.LogPath = "./test"
	cfg.FileName = "unittest"
	w, _ := NewWriterFromConfig(&cfg)
	return w.(*Writer)
}

func TestNewWriter(t *testing.T) {
	if _, err := NewWriter(
		WithTimeTagFormat("200601021504"), WithLogPath("./"), WithFileName("foo"),
		WithCompress(),
		WithMaxRemain(3), WithRollingVolumeSize("100mb"), WithRollingTimePattern("0 0 0 * * *"),
	); err != nil {
		t.Fatal("error in test new writer", err)
	}
	os.Remove("./foo.log")
}

func TestWrite(t *testing.T) {
	var writer io.WriteCloser
	var c int = 1000
	var l int = 1024

	writer = newWriter()
	for i := 0; i < c; i++ {
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
	}
	writer.Close()
	clean()

	writer = newVolumeWriter()
	for i := 0; i < c; i++ {
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
	}
	writer.Close()
	clean()

}

func TestWriteParallel(t *testing.T) {
	var writer io.WriteCloser
	var c int = 1000
	var l int = 1024

	t.Run("none", func(t *testing.T) {
		t.Parallel()
		writer = newWriter()
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
		for i := 0; i < c; i++ {
			bf := make([]byte, l)
			rand.Read(bf)
			writer.Write(bf)
		}
		writer.Close()
		clean()
	})
}

func TestVolumeWriteParallel(t *testing.T) {
	var writer io.WriteCloser
	var c int = 1000
	var l int = 1024

	t.Run("none", func(t *testing.T) {
		t.Parallel()
		writer = newVolumeWriter()
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
		for i := 0; i < c; i++ {
			bf := make([]byte, l)
			rand.Read(bf)
			writer.Write(bf)
		}
		writer.Close()
		clean()
	})
}

func TestReopen(t *testing.T) {
	var c int = 1000
	var l int = 1024

	t.Run("none", func(t *testing.T) {
		t.Parallel()
		writer := newWriter()
		for i := 0; i < c; i++ {
			bf := make([]byte, l)
			rand.Read(bf)
			writer.Write(bf)
		}
		writer.Reopen("./test/unittest.reopen")
		writer.Close()
		clean()
	})
}

func TestAutoRemove(t *testing.T) {
	var c int = 1000
	var l int = 1024

	writer := newWriter()
	for i := 0; i < c; i++ {
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
	}
	writer.Close()
	writer.cf.MaxRemain = 0
	clean()
}

func TestCompress(t *testing.T) {
	var c int = 1000
	var l int = 1024

	writer := newWriter()
	for i := 0; i < c; i++ {
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
	}
	writer.CompressFile(writer.file, "./test/unittest.gz")
	writer.Close()
	clean()
}
