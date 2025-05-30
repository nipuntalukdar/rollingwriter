package rollingwriter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	options := []Option{
		WithTimeTagFormat("200601021504"), WithFilePath("./log.log"),
		WithCompress(),
		WithMaxBackups(3), WithRollingVolumeSize("100mb"), WithRollingTimePattern("0 0 0 * * *"),
	}
	cfg := NewDefaultConfig()
	for _, opt := range options {
		opt(&cfg)
	}

	destcfg := Config{
		FilePath:           "./log.log",
		TimeTagFormat:      "200601021504",
		MaxBackups:         3,
		RollingPolicy:      TimeRolling,   // TimeRotate by default
		RollingTimePattern: "0 0 0 * * *", // Rolling at 00:00 AM everyday
		RollingVolumeSize:  "100mb",
		Compress:           true,
		BufferSize:         DefaultBufferSize,
		QueueSize:          DefaultQueueSize,
	}

	sanitizeConfig(&destcfg)

	assert.Equal(t, cfg, destcfg)
}
