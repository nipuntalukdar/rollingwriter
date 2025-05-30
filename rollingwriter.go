package rollingwriter

import (
	"errors"
	"io"
	"os"
	"time"
)

// RollingPolicies giveout 3 policy for rolling.
const (
	WithoutRolling = iota
	TimeRolling
	VolumeRolling

	// DefaultFileMode set the default open mode rw-r--r-- by default
	DefaultFileMode = os.FileMode(0644)
	// DefaultDirMode set the default open mode rwx------ by default
	DefaultDirMode = os.FileMode(0700)

	// MinQueueSize define the minimum queue size for asynchronize write
	DefaultQueueSize = 8 * 1024
	// MinBufferSize define the minimum buffer size for log messages, 1 MB
	DefaultBufferSize = 1024 * 1024

	// MinQueueSize define the minimum queue size for asynchronize write
	MinQueueSize = 64
	// MinBufferSize define the minimum buffer size for log messages
	MinBufferSize = 2048
)

var (
	// Precision defined the precision about the reopen operation condition
	// check duration within second
	Precision = 1
	// Max write to file interval in seconds
	MaxWriteInterval = 1

	// DefaultFileFlag set the default file flag
	DefaultFileFlag = os.O_RDWR | os.O_CREATE | os.O_APPEND

	// ErrInternal defined the internal error
	ErrInternal = errors.New("error internal")
	// ErrClosed defined write while ctx close
	ErrClosed = errors.New("error write on close")
	// ErrInvalidArgument defined the invalid argument
	ErrInvalidArgument = errors.New("error argument invalid")
	// ErrQueueFull defined the queue full
	ErrQueueFull = errors.New("async log queue full")
)

// FileMonitor writes new backup file event.
type FileMonitor interface {
	// RotationEvents returns a string channel
	// when rotation events occur, new backup filenames will be sent on this channel
	RotationEvents() chan string
	// Close the Manager
	Close()
}

// RollingWriter implement the io writer
type RollingWriter interface {
	io.Writer
	Close() error
}

// LogFileFormatter log file format function
type LogFileFormatter func(time.Time) string

// Config give out the config for manager
type Config struct {
	// FilePath defines the full path of log file
	// there comes out 2 different log file:
	//
	// 1. the current active file
	//	file path is located @:
	//	FilePath
	//
	// 2. the truncated log file
	//	the truncated log file is backup here:
	//	[fileDir]/[fileName].[fileExt].[TimeTag]
	//  if compressed true
	//	[fileDir]/[fileName].[fileExt].gz.[TimeTag]
	//
	// NOTICE: blank field will be ignored
	// By default we using '-' as separator, you can set it yourself
	TimeTagFormat string `json:"time_tag_format,omitempty"`
	FilePath      string `json:"file_path,omitempty"`

	// Mode of log files created
	FileMode os.FileMode `json:"file_mode,omitempty"`

	// Directory mode, mode of directory created
	DirMode os.FileMode `json:"dir_mode,omitempty"`

	// MaxBackups is the maximum number of old files to retain, if set 0 will
	// all old files will be retained.
	MaxBackups int `json:"max_remain,omitempty"`

	// RollingPolicy give out the rolling policy
	// We got 3 policies(actually, 2):
	//
	//	1. WithoutRolling: no rolling will happen
	//	2. TimeRolling: rolling by time
	//	3. VolumeRolling: rolling by file size
	RollingPolicy      int    `json:"rolling_ploicy,omitempty"`
	RollingTimePattern string `json:"rolling_time_pattern,omitempty"`
	RollingVolumeSize  string `json:"rolling_volume_size,omitempty"`

	// Compress will compress log file with gzip
	Compress bool `json:"compress,omitempty"`

	// FilterEmptyBackup will not backup empty file if you set it true
	FilterEmptyBackup bool `json:"filter_empty_backup,omitempty"`

	// Maximum buffer  size
	BufferSize int `json:"max_buffer_size,omitempty"`

	// Max queue size for log messages
	QueueSize int `json:"max_queue_size,omitempty"`
}

// NewDefaultConfig return the default config
func NewDefaultConfig() Config {
	return Config{
		FilePath:           "./log/log.log",
		TimeTagFormat:      "200601021504",
		FileMode:           DefaultFileMode,
		DirMode:            DefaultDirMode,
		MaxBackups:         0,             // disable backup pruning
		RollingPolicy:      1,             // TimeRotate by default
		RollingTimePattern: "0 0 0 * * *", // Rolling at 00:00 AM everyday
		RollingVolumeSize:  "1M",
		BufferSize:         DefaultBufferSize,
		QueueSize:          DefaultQueueSize,
		Compress:           false,
	}
}

// Option defined config option
type Option func(*Config)

// WithTimeTagFormat set the TimeTag format string
func WithTimeTagFormat(format string) Option {
	return func(p *Config) {
		p.TimeTagFormat = format
	}
}

// WithFilePath set the full path of rolling file,
// if dir tree does not exist, it will be created
func WithFilePath(path string) Option {
	return func(p *Config) {
		p.FilePath = path
	}
}

// WithCompress will auto compress auto rotated files with gzip
func WithCompress() Option {
	return func(p *Config) {
		p.Compress = true
	}
}

// WithMaxBackups sets the maximum number of backup files to retain
// 0 will disable pruning backups files, this is the default behaviour
func WithMaxBackups(max int) Option {
	return func(p *Config) {
		p.MaxBackups = max
	}
}

// WithoutRolling set no rolling policy
func WithoutRollingPolicy() Option {
	return func(p *Config) {
		p.RollingPolicy = WithoutRolling
	}
}

// WithRollingTimePattern set the time rolling policy time pattern obey the Corn table style
// visit http://crontab.org/ for details
func WithRollingTimePattern(pattern string) Option {
	return func(p *Config) {
		p.RollingPolicy = TimeRolling
		p.RollingTimePattern = pattern
	}
}

// WithRollingVolumeSize set the rolling file truncation threshold size
func WithRollingVolumeSize(size string) Option {
	return func(p *Config) {
		p.RollingPolicy = VolumeRolling
		p.RollingVolumeSize = size
	}
}
