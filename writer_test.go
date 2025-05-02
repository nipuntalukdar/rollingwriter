package rollingwriter

import (
	"crypto/rand"
	"io"
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
	cfg.FilePath = "./test/unittest.log"
	w, _ := NewWriterFromConfig(&cfg)
	return w.(*Writer)
}

func newVolumeWriter() *Writer {
	cfg := NewDefaultConfig()
	cfg.RollingPolicy = 3
	cfg.RollingVolumeSize = "1mb"
	cfg.FilePath = "./test/unittest.log"
	w, _ := NewWriterFromConfig(&cfg)
	return w.(*Writer)
}

func TestNewWriter(t *testing.T) {
	if _, err := NewWriter(
		WithTimeTagFormat("200601021504"), WithFilePath("./foo.log"),
		WithCompress(),
		WithMaxBackups(3), WithRollingVolumeSize("100mb"), WithRollingTimePattern("0 0 0 * * *"),
	); err != nil {
		t.Fatal("error in test new writer", err)
	}
	os.Remove("./foo.log")
}

func TestWrite(t *testing.T) {
	var writer io.WriteCloser
	c := 1000
	l := 1024

	writer = newWriter()
	for range c {
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
	}
	writer.Close()
	clean()

	writer = newVolumeWriter()
	for range c {
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
	}
	writer.Close()
	clean()
}

func TestWriteParallel(t *testing.T) {
	var writer io.WriteCloser
	c := 1000
	l := 1024

	t.Run("none", func(t *testing.T) {
		t.Parallel()
		writer = newWriter()
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
		for range c {
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
	c := 1000
	l := 1024

	t.Run("none", func(t *testing.T) {
		t.Parallel()
		writer = newVolumeWriter()
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
		for range c {
			bf := make([]byte, l)
			rand.Read(bf)
			writer.Write(bf)
		}
		writer.Close()
		clean()
	})
}

func TestRotateFile(t *testing.T) {
	c := 1000
	l := 1024

	t.Run("none", func(t *testing.T) {
		t.Parallel()
		writer := newWriter()
		for range c {
			bf := make([]byte, l)
			rand.Read(bf)
			writer.Write(bf)
		}
		writer.RotateFile("./test/unittest.reopen")
		writer.Close()
		clean()
	})
}

func TestAutoRemove(t *testing.T) {
	c := 1000
	l := 1024

	writer := newWriter()
	for range c {
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
	}
	writer.Close()
	writer.cf.MaxBackups = 0
	clean()
}

func TestCompress(t *testing.T) {
	c := 1000
	l := 1024

	writer := newWriter()
	for range c {
		bf := make([]byte, l)
		rand.Read(bf)
		writer.Write(bf)
	}
	CompressFile(writer.file, "./test/unittest.gz", os.FileMode(0644))
	writer.Close()
	clean()
}
