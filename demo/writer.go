package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/nipuntalukdar/rollingwriter"
)

func run_writer() {
	// Write implements an interface to io.Writer
	config := rollingwriter.Config{
		LogPath:       "./log",        //Log path
		TimeTagFormat: "060102150405", //Time format string
		FileName:      "test",         //Log file name
		MaxRemain:     5,              //Maximum number of log foles to retain

		// There are currently two rolling strategis:
		// rolling according to time rolling according to size
		// Time rolling: The configuration strategy is like crontable.
		// For example, if it is split at 0:0 every day, then configure 0 0 0 * * *
		// Size rolling: Configure the rolling size threshold of a single
		// log file (uncompressed), such as 1G, 500M, 100K etc.

		RollingPolicy:      rollingwriter.TimeRolling, //Rolling strategy: norolling timerolling volumerolling
		RollingTimePattern: "30 * * * * *",            //Rolling time pattern, roll over every hour
		RollingVolumeSize:  "20k",                     //Minimum size before rolling

		// Compress will compress log file with gzip
		Compress: true,
	}

	//Create a  writer
	writer, err := rollingwriter.NewWriterFromConfig(&config)
	if err != nil {
		// We are just exiting without handling the error
		panic(err)
	}

	// 10 writer concurrenly adding logs
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(w *sync.WaitGroup) {
			for j := 0; j < 1000; j++ {
				fmt.Fprintf(writer, "now the time is given here :%s \n", time.Now())
				time.Sleep(10 * time.Millisecond)
			}
			w.Done()
		}(&wg)
	}
	// Wait for all writers to be done
	wg.Wait()

	//Close the underlying writer
	writer.Close()
}
