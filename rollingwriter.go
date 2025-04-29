package rollingwriter

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
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
	DefaultQueueSize =  8 * 1024
	// MinBufferSize define the minimum buffer size for log messages, 1 MB
	DefaultBufferSize = 1024 * 1024

	// MinQueueSize define the minimum queue size for asynchronize write
	MinQueueSize =  64
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

// Manager used to trigger rolling event.
type Manager interface {
	// Fire will return a string channel
	// while the rolling event occurred, new file name will generate
	Fire() chan string
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
	// LogPath defined the full path of log file directory.
	// there comes out 2 different log file:
	//
	// 1. the current log
	//	log file path is located here:
	//	[LogPath]/[FileName].[FileExtension]
	//
	// 2. the truncated log file
	//	the truncated log file is backup here:
	//	[LogPath]/[FileName].[FileExtension].[TimeTag]
	//  if compressed true
	//	[LogPath]/[FileName].[FileExtension].gz.[TimeTag]
	//
	// NOTICE: blank field will be ignored
	// By default we using '-' as separator, you can set it yourself
	TimeTagFormat string `json:"time_tag_format,omitempty"`
	LogPath       string `json:"log_path,omitempty"`
	FileName      string `json:"file_name,omitempty"`
	// FileExtension defines the log file extension. By default, it's 'log'
	FileExtension string `json:"file_extension,omitempty"`
	// FileFormatter log file path formatter for the file start write
	// By default, append '.gz' suffix when Compress is true
	FileFormatter LogFileFormatter `json:"-"`

	// Mode of log files created
	FileMode os.FileMode `json:"file_mode,omitempty"`

	// Directory mode, mode of directory created
	DirMode os.FileMode `json:"dir_mode,omitempty"`

	// MaxRemain will auto clear the roling file list, set 0 will disable auto clean
	MaxRemain int `json:"max_remain,omitempty"`

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

func (c *Config) fileFormat(start time.Time) (filename string) {
	if c.FileFormatter != nil {
		filename = c.FileFormatter(start)
		if c.Compress && filepath.Ext(filename) != ".gz" {
			filename += ".gz"
		}
	} else {
		// [path-to-log]/filename.[FileExtension].2007010215041517
		timeTag := start.Format(c.TimeTagFormat)
		if c.Compress {
			filename = path.Join(c.LogPath, c.FileName+"."+c.FileExtension+".gz."+timeTag)
		} else {
			filename = path.Join(c.LogPath, c.FileName+"."+c.FileExtension+"."+timeTag)
		}
	}
	return
}

// NewDefaultConfig return the default config
func NewDefaultConfig() Config {
	return Config{
		LogPath:            "./log",
		TimeTagFormat:      "200601021504",
		FileName:           "log",
		FileExtension:      "log",
		FileMode:           DefaultFileMode,
		DirMode:            DefaultDirMode,
		MaxRemain:          -1,            // disable auto delete
		RollingPolicy:      1,             // TimeRotate by default
		RollingTimePattern: "0 0 0 * * *", // Rolling at 00:00 AM everyday
		RollingVolumeSize:  "1G",
		BufferSize:         DefaultBufferSize,
		QueueSize:          DefaultQueueSize,
		Compress:           false,
	}
}

// LogFilePath return the absolute path on log file
func LogFilePath(c *Config) (filepath string) {
	filepath = path.Join(c.LogPath, c.FileName) + "." + c.FileExtension
	return
}

// Option defined config option
type Option func(*Config)

// WithTimeTagFormat set the TimeTag format string
func WithTimeTagFormat(format string) Option {
	return func(p *Config) {
		p.TimeTagFormat = format
	}
}

// WithLogPath set the log dir and auto create dir tree
// if the dir/path is not exist
func WithLogPath(path string) Option {
	return func(p *Config) {
		p.LogPath = path
	}
}

// WithFileName set the log file name
func WithFileName(name string) Option {
	return func(p *Config) {
		p.FileName = name
	}
}

// WithFileExtension set the log file extension
func WithFileExtension(ext string) Option {
	return func(p *Config) {
		p.FileExtension = ext
	}
}

// WithFileFormatter set the log file formatter
func WithFileFormatter(formatter LogFileFormatter) Option {
	return func(p *Config) {
		p.FileFormatter = formatter
	}
}

// WithCompress will auto compress the truncated log file with gzip
func WithCompress() Option {
	return func(p *Config) {
		p.Compress = true
	}
}

// WithMaxRemain enable the auto deletion for old file when exceed the given max value
// Bydefault -1 will disable the auto deletion
func WithMaxRemain(max int) Option {
	return func(p *Config) {
		p.MaxRemain = max
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
