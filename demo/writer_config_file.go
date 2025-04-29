package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/nipuntalukdar/rollingwriter"
)

func run_wrtiter_config_from_file(configfile string) {
	// Writer implements an interface to io.Writer
	// Create a  writer where configs are in a file
	writer, err := rollingwriter.NewWriterFromConfigFile(configfile)
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

	// Close the underlying writer
	writer.Close()
}
