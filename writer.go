package rollingwriter

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Writer provide a synchronous file writer
// if Lock is set true, write will be guaranteed by lock
type Writer struct {
	m             Manager
	file          *os.File
	absPath       string
	fire          chan string
	cf            *Config
	rollingFileCh chan string
	writeCh       chan []byte
	errorCh       chan error
	ctx           context.Context
	cancel        context.CancelFunc
}

func (w *Writer) startFileWriterLoop() error {
	var err error
	c := w.cf

	// make dir for path if not exist
	if err = os.MkdirAll(filepath.Dir(c.FilePath), c.DirMode); err != nil {
		return fmt.Errorf("failed to create directories for %s: %w", c.FilePath, err)
	}

	if w.absPath, err = filepath.Abs(c.FilePath); err != nil {
		return ErrInvalidArgument
	}

	// open the file and get the FD
	file, err := os.OpenFile(w.absPath, DefaultFileFlag, c.FileMode)
	if err != nil {
		return fmt.Errorf("failed to open file - %s: %w", w.absPath, err)
	}

	w.file = file

	logbuffer := make([]byte, 0, c.BufferSize)
	bbuffer := bytes.NewBuffer(logbuffer)
	ticker := time.NewTicker(time.Duration(MaxWriteInterval) * time.Second)

	directWriteMsgSize := c.BufferSize / 4

	go func() {
		for {
			select {
			case logmsg := <-w.writeCh:
				// First, try to add the logmsg to buffer always
				if len(logmsg)+bbuffer.Len() < c.BufferSize {
					bbuffer.Write(logmsg)
				} else {
					n, err := bbuffer.WriteTo(w.file)
					if err != nil {
						log.Println("File write", n, err)
					}
					bbuffer.Reset()
					// if the new message is big, write to file directly
					if len(logmsg) > directWriteMsgSize {
						n, err := w.file.Write(logmsg)
						if err != nil {
							log.Println("File write", n, err)
						}
					} else {
						bbuffer.Write(logmsg)
					}
				}
			case filename := <-w.fire:
				if err := w.Reopen(filename); err != nil {
					log.Println("File rolling error", err)
				}
			case <-ticker.C:
				if bbuffer.Len() > 0 {
					n, err := bbuffer.WriteTo(w.file)
					if err != nil {
						log.Println("File write", n, err)
					}
					bbuffer.Reset()
				}
			case <-w.errorCh:
				// Stopping write
				if bbuffer.Len() > 0 {
					n, err := bbuffer.WriteTo(w.file)
					if err != nil {
						log.Println("File write", n, err)
					}
				}
				w.file.Close()
				w.errorCh <- nil
				return
			}
		}
	}()
	return nil
}

/*
	// TODO: make this into a function
	if c.MaxRemain > 0 {
		w.rollingFileCh = make(chan string, c.MaxRemain)
		dir, err := os.ReadDir(c.LogPath)
		if err != nil {
			w.errorCh <- err
			return
		}

		files := make([]string, 0, 10)
		for _, fi := range dir {
			if fi.IsDir() {
				continue
			}

			fileName := c.FileName + "." + c.FileExtension + "."
			if strings.Contains(fi.Name(), fileName) {
				fileSuffix := path.Ext(fi.Name())
				if len(fileSuffix) > 1 {
					_, err := time.Parse(c.TimeTagFormat, fileSuffix[1:])
					if err == nil {
						files = append(files, fi.Name())
					}
				}
			}
		}
		sort.Slice(files, func(i, j int) bool {
			fileSuffix1 := path.Ext(files[i])
			fileSuffix2 := path.Ext(files[j])
			t1, _ := time.Parse(c.TimeTagFormat, fileSuffix1[1:])
			t2, _ := time.Parse(c.TimeTagFormat, fileSuffix2[1:])
			return t1.Before(t2)
		})

		for _, file := range files {
		retry:
			select {
			case w.rollingFileCh <- path.Join(c.LogPath, file):
			default:
				w.DoRemove()
				goto retry // remove the file and retry
			}
		}
	}
*/

// NewWriterFromConfig generate the rollingWriter with given config
func NewWriterFromConfig(c *Config) (RollingWriter, error) {
	// Set defaults
	sanitizeConfig(c)

	// Start the Manager
	mng, err := NewManager(c)
	if err != nil {
		return nil, err
	}

	var rollingWriter RollingWriter
	writer := Writer{
		m:       mng,
		fire:    mng.Fire(),
		cf:      c,
		writeCh: make(chan []byte, c.QueueSize),
		errorCh: make(chan error),
	}

	writer.ctx, writer.cancel = context.WithCancel(context.Background())
	err = writer.startFileWriterLoop()
	if err != nil {
		mng.Close()
		log.Println("Some error", err)
		os.Exit(1)
	}
	rollingWriter = &writer
	return rollingWriter, nil
}

// NewWriter generate the rollingWriter with given option
func NewWriter(ops ...Option) (RollingWriter, error) {
	cfg := NewDefaultConfig()
	for _, opt := range ops {
		opt(&cfg)
	}
	return NewWriterFromConfig(&cfg)
}

func sanitizeConfig(c *Config) {
	if c.QueueSize < MinQueueSize {
		c.QueueSize = MinQueueSize
	}

	if c.BufferSize < MinBufferSize {
		c.BufferSize = MinBufferSize
	}

	if c.FileMode == 0 {
		c.FileMode = DefaultFileMode
	}

	if c.DirMode == 0 {
		c.DirMode = DefaultDirMode
	}
}

// DoRemove will delete the oldest file
func (w *Writer) DoRemove() bool {
	select {
	case file := <-w.rollingFileCh:
		// remove the oldest file
		if err := os.Remove(file); err != nil {
			// TODO: pass error back via errorCh
			log.Println("error in remove log file", file, err)
		}
		return true
	default:
		// Channel is empty, nothing to remove
		return false
	}
}

// CompressFile compress log file write into .gz
func (w *Writer) CompressFile(oldfile *os.File, cmpname string) error {
	cmpfile, err := os.OpenFile(cmpname, DefaultFileFlag, w.cf.FileMode)
	if err != nil {
		return err
	}
	defer cmpfile.Close()
	gw := gzip.NewWriter(cmpfile)
	defer gw.Close()

	if _, err = oldfile.Seek(0, 0); err != nil {
		return err
	}

	if _, err = io.Copy(gw, oldfile); err != nil {
		if errR := os.Remove(cmpname); errR != nil {
			return errR
		}
		return err
	}
	return nil
}

// Reopen do the rotate, open new file and swap FD then trate the old FD
func (w *Writer) Reopen(file string) error {
	if w.cf.FilterEmptyBackup {
		fileInfo, err := w.file.Stat()
		if err != nil {
			return err
		}

		if fileInfo.Size() == 0 {
			return nil
		}
	}

	w.file.Close()
	if err := os.Rename(w.absPath, file); err != nil {
		return err
	}
	newfile, err := os.OpenFile(w.absPath, DefaultFileFlag, w.cf.FileMode)
	if err != nil {
		return err
	}

	w.file = newfile

	go func() {
		if w.cf.Compress {
			if err := os.Rename(file, file+".tmp"); err != nil {
				log.Println("error in compress rename tempfile", err)
				return
			}
			oldfile, err := os.OpenFile(file+".tmp", DefaultFileFlag, w.cf.FileMode)
			if err != nil {
				log.Println("error in open tempfile", err)
				return
			}
			var closeOnce sync.Once
			defer closeOnce.Do(func() { oldfile.Close() })
			if err := w.CompressFile(oldfile, file); err != nil {
				log.Println("error in compress log file", err)
				return
			}
			closeOnce.Do(func() { oldfile.Close() })
			err = os.Remove(file + ".tmp")
			if err != nil {
				log.Println("error in remove tempfile", err)
				return
			}
		}

		if w.cf.MaxBackups > 0 {
		retry:
			select {
			case w.rollingFileCh <- file:
			default:
				w.DoRemove()
				goto retry // remove the file and retry
			}
		}
	}()
	return nil
}

func (w *Writer) Write(b []byte) (int, error) {
	w.writeCh <- b
	return len(b), nil
}

// Close the file and return
func (w *Writer) Close() error {
	defer recover()
	w.errorCh <- nil
	select {
	case <-w.errorCh:
	case <-time.After(4 * time.Second):
	}
	w.m.Close()
	return nil
}
