package rollingwriter

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"os"
	"path"
	"sort"
	"strings"
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
	rollingfilech chan string
	writech       chan []byte
	errorch       chan error
}

func (w *Writer) fileWriter() {
	// makeup log path and create
	c := w.cf
	if c.LogPath == "" || c.FileName == "" {
		w.errorch <- ErrInvalidArgument
		return
	}

	if c.FileExtension == "" {
		c.FileExtension = "log"
	}

	// make dir for path if not exist
	if err := os.MkdirAll(c.LogPath, 0700); err != nil {
		w.errorch <- err
		return
	}

	filepath := LogFilePath(c)
	w.absPath = filepath
	// open the file and get the FD
	file, err := os.OpenFile(filepath, DefaultFileFlag, DefaultFileMode)
	if err != nil {
		w.errorch <- err
		return
	}
	if c.MaxRemain > 0 {
		w.rollingfilech = make(chan string, c.MaxRemain)
		dir, err := os.ReadDir(c.LogPath)
		if err != nil {
			w.errorch <- err
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
			case w.rollingfilech <- path.Join(c.LogPath, file):
			default:
				w.DoRemove()
				goto retry // remove the file and retry
			}
		}
	}

	w.file = file
	w.errorch <- nil

	logbuffer := make([]byte, 0, BufferSize)
	bbuffer := bytes.NewBuffer(logbuffer)
	ticker := time.NewTicker(time.Duration(MaxWriteInterval) * time.Second)

	if BufferSize < 2048 {
		BufferSize = 2048
	}
	directWriteMsgSize := BufferSize / 4

	for {
		select {
		case logmsg := <-w.writech:
			// First, try to add the logmsg to buffer always
			if len(logmsg)+bbuffer.Len() < BufferSize {
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
		case <-w.errorch:
			// Stopping write
			if bbuffer.Len() > 0 {
				n, err := bbuffer.WriteTo(w.file)
				if err != nil {
					log.Println("File write", n, err)
				}
			}
			w.file.Close()
			w.errorch <- nil
			return
		}
	}
}

// NewWriterFromConfig generate the rollingWriter with given config
func NewWriterFromConfig(c *Config) (RollingWriter, error) {
	// Start the Manager
	mng, err := NewManager(c)
	if err != nil {
		return nil, err
	}

	var rollingWriter RollingWriter
	if QueueSize < 64 {
		QueueSize = 64
	}
	writer := Writer{
		m:       mng,
		fire:    mng.Fire(),
		cf:      c,
		writech: make(chan []byte, QueueSize),
		errorch: make(chan error),
	}

	go writer.fileWriter()
	err = <-writer.errorch
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

// NewWriterFromConfigFile generate the rollingWriter with given config file
func NewWriterFromConfigFile(path string) (RollingWriter, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := NewDefaultConfig()
	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(buf, &cfg); err != nil {
		return nil, err
	}
	return NewWriterFromConfig(&cfg)
}

// DoRemove will delete the oldest file
func (w *Writer) DoRemove() {
	select {
	case file := <-w.rollingfilech:
		// remove the oldest file
		if err := os.Remove(file); err != nil {
			log.Println("error in remove log file", file, err)
		}
	}
}

// CompressFile compress log file write into .gz
func (w *Writer) CompressFile(oldfile *os.File, cmpname string) error {
	cmpfile, err := os.OpenFile(cmpname, DefaultFileFlag, DefaultFileMode)
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
	newfile, err := os.OpenFile(w.absPath, DefaultFileFlag, DefaultFileMode)
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
			oldfile, err := os.OpenFile(file+".tmp", DefaultFileFlag, DefaultFileMode)
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

		if w.cf.MaxRemain > 0 {
		retry:
			select {
			case w.rollingfilech <- file:
			default:
				w.DoRemove()
				goto retry // remove the file and retry
			}
		}
	}()
	return nil
}

func (w *Writer) Write(b []byte) (int, error) {
	w.writech <- b
	return len(b), nil
}

// Close the file and return
func (w *Writer) Close() error {
	defer recover()
	w.errorch <- nil
	select {
	case <-w.errorch:
	case <-time.After(4 * time.Second):
	}
	w.m.Close()
	return nil
}
