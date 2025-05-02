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
	monitor          FileMonitor
	file             *os.File
	absPath          string
	fire             chan string
	conf             *Config
	rotationEventsCh chan string
	writeCh          chan []byte
	errorCh          chan error
	ctx              context.Context
	cancel           context.CancelFunc
}

func (w *Writer) startFileWriterLoop() error {
	var err error
	c := w.conf

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
			case data := <-w.writeCh:
				// First, try to add the data to buffer
				if len(data)+bbuffer.Len() < c.BufferSize {
					bbuffer.Write(data)
				} else {
					n, err := bbuffer.WriteTo(w.file)
					if err != nil {
						log.Println("File write", n, err)
					}
					bbuffer.Reset()
					// if the new message is big, write to file directly
					if len(data) > directWriteMsgSize {
						n, err := w.file.Write(data)
						if err != nil {
							log.Println("File write", n, err)
						}
					} else {
						bbuffer.Write(data)
					}
				}
			case filename := <-w.fire:
				if err := w.RotateFile(filename); err != nil {
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
		monitor:          mng,
		rotationEventsCh: mng.RotationEvents(),
		conf:             c,
		writeCh:          make(chan []byte, c.QueueSize),
		errorCh:          make(chan error),
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
	case file := <-w.rotationEventsCh:
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
func CompressFile(oldfile *os.File, cmpname string, fileMode os.FileMode) error {
	cmpfile, err := os.OpenFile(cmpname, DefaultFileFlag, fileMode)
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

// RotateFile do the rotate, open new file and swap FD then trate the old FD
func (w *Writer) RotateFile(newBackUpFile string) error {
	if w.conf.FilterEmptyBackup {
		fileInfo, err := w.file.Stat()
		if err != nil {
			return err
		}

		if fileInfo.Size() == 0 {
			return nil
		}
	}

	w.file.Close()
	if err := os.Rename(w.absPath, newBackUpFile); err != nil {
		return err
	}
	newfile, err := os.OpenFile(w.absPath, DefaultFileFlag, w.conf.FileMode)
	if err != nil {
		return err
	}

	w.file = newfile

	go func() {
		if w.conf.Compress {
			if err := os.Rename(newBackUpFile, newBackUpFile+".tmp"); err != nil {
				log.Println("error in compress rename tempfile", err)
				return
			}
			tmpBackupFile, err := os.OpenFile(newBackUpFile+".tmp", DefaultFileFlag, w.conf.FileMode)
			if err != nil {
				log.Println("error in open tempfile", err)
				return
			}
			var closeOnce sync.Once
			defer closeOnce.Do(func() { tmpBackupFile.Close() })
			if err := CompressFile(tmpBackupFile, newBackUpFile, w.conf.FileMode); err != nil {
				log.Println("error in compress log file", err)
				return
			}
			closeOnce.Do(func() { tmpBackupFile.Close() })
			err = os.Remove(newBackUpFile + ".tmp")
			if err != nil {
				log.Println("error in remove tempfile", err)
				return
			}
		}

		if w.conf.MaxBackups > 0 {
		retry:
			select {
			case w.rotationEventsCh <- newBackUpFile:
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
	w.monitor.Close()
	return nil
}
