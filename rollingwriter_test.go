package rollingwriter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	options := []Option{
		WithTimeTagFormat("200601021504"), WithLogPath("./"), WithFileName("foo"),
        WithCompress(),
		WithMaxRemain(3), WithRollingVolumeSize("100mb"), WithRollingTimePattern("0 0 0 * * *"),
	}
	cfg := NewDefaultConfig()
	for _, opt := range options {
		opt(&cfg)
	}

	destcfg := Config{
		LogPath:                "./",
		TimeTagFormat:          "200601021504",
		FileName:               "foo",
		FileExtension:          "log",
		MaxRemain:              3,             // disable auto delete
		RollingPolicy:          TimeRolling,   // TimeRotate by default
		RollingTimePattern:     "0 0 0 * * *", // Rolling at 00:00 AM everyday
		RollingVolumeSize:      "100mb",
		Compress:               true,
		BufferSize:             DefaultBufferSize,
		QueueSize:              DefaultQueueSize,
	}
	assert.Equal(t, cfg, destcfg)
}
