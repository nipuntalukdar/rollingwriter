package rollingwriter

import (
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron"
)

type manager struct {
	thresholdSize    int64
	startAt          time.Time
	cr               *cron.Cron
	rotationEventsCh chan string
	doneCh           chan bool
	wg               sync.WaitGroup
	lock             sync.Mutex
}

// NewManager generate the Manager with config
func NewManager(c *Config) (FileMonitor, error) {
	m := &manager{
		startAt:          time.Now(),
		cr:               cron.New(),
		rotationEventsCh: make(chan string),
		doneCh:           make(chan bool),
		wg:               sync.WaitGroup{},
	}

	// start the manager according to policy
	switch c.RollingPolicy {
	default:
		fallthrough
	case WithoutRolling:
		return m, nil
	case TimeRolling:
		if err := m.cr.AddFunc(c.RollingTimePattern, func() {
			m.rotationEventsCh <- m.GenNewBackupFileName(c)
		}); err != nil {
			return nil, err
		}
		m.cr.Start()
	case VolumeRolling:
		m.ParseVolume(c)
		go func() {
			timer := time.NewTicker(time.Duration(Precision) * time.Second)
			defer timer.Stop()

			var file *os.File
			var err error

			for {
				select {
				case <-m.doneCh:
					return
				case <-timer.C:
					if file, err = os.Open(c.FilePath); err != nil {
						continue
					}
					if info, err := file.Stat(); err == nil && info.Size() > m.thresholdSize {
						m.rotationEventsCh <- m.GenNewBackupFileName(c)
					}
					file.Close()
					// check if you need to prune backups
				}
			}
		}()
	}
	return m, nil
}

// RotationEvents returns a channel that provides new backup filenames when rotation events occur
func (m *manager) RotationEvents() chan string {
	return m.rotationEventsCh
}

// Close stop the manager and returns
func (m *manager) Close() {
	close(m.doneCh)
	m.cr.Stop()
}

// ParseVolume parse the config volume format and return threshold
func (m *manager) ParseVolume(c *Config) {
	s := []byte(strings.ToUpper(c.RollingVolumeSize))
	if !strings.Contains(string(s), "K") && !strings.Contains(string(s), "KB") &&
		!strings.Contains(string(s), "M") && !strings.Contains(string(s), "MB") &&
		!strings.Contains(string(s), "G") && !strings.Contains(string(s), "GB") &&
		!strings.Contains(string(s), "T") && !strings.Contains(string(s), "TB") {

		// set the default threshold with 1GB
		m.thresholdSize = 1024 * 1024 * 1024
		return
	}

	var unit int64 = 1
	p, _ := strconv.Atoi(string(s[:len(s)-1]))
	unitstr := string(s[len(s)-1])

	if s[len(s)-1] == 'B' {
		p, _ = strconv.Atoi(string(s[:len(s)-2]))
		unitstr = string(s[len(s)-2:])
	}

	switch unitstr {
	default:
		fallthrough
	case "T", "TB":
		unit *= 1024
		fallthrough
	case "G", "GB":
		unit *= 1024
		fallthrough
	case "M", "MB":
		unit *= 1024
		fallthrough
	case "K", "KB":
		unit *= 1024
	}
	m.thresholdSize = int64(p) * unit
}

// GenNewBackupFileName generates a new backup file
func (m *manager) GenNewBackupFileName(c *Config) string {
	m.lock.Lock()
	defer func() {
		m.startAt = time.Now()
		m.lock.Unlock()
	}()

	timeTag := m.startAt.Format(c.TimeTagFormat)
	if c.Compress {
		return path.Join(c.FilePath + ".gz." + timeTag)
	}

	return path.Join(c.FilePath + "." + timeTag)
}
