# RollingWriter 
RollingWriter is an auto rotate `io.Writer` implementation. It can work well with logger.
`This repo is a fork` of [this rollingwriter repo](https://github.com/arthurkiller/rollingwriter). But I made it an independent repo as there were many changes for simplification and performance. Major changes are:
* The logs are always added asynchronouly to the file
* Log messages are always buffered first and hence many logs may be added to the underlying file in one shot
* Writing to the file and roll over of the file is handled by by the same go routine and hence the need for synchronization across different go routines are no longer necessary
* Locks are removed. All communications are through channels without explicit synchronization with locks

RollingWriter contains 2 separate patrs:
* Manager: decide when to rotate the file with policy. RlingPolicy give out the rolling policy
    * WithoutRolling: no rolling will happen
    * TimeRolling: rolling by time
    * VolumeRolling: rolling by file size

* Writer: impement the io.Writer and do the io write
    * Concurrent and safe for adding logs from multiple go routines
    * Logs are added asynchronously to file without blocking the callers
    * Log messages are buffered and write to files may add multiple messages in operation 

## Features
* Auto rotate with multi rotate policies
* Implement parallel and safe io.Writer
* Max remain rolling files with auto cleanup
* Easy for user to implement your manager

## Benchmark
```bash
$ go test -bench=.
goos: linux
goarch: amd64
pkg: github.com/nipuntalukdar/rollingwriter
cpu: Intel(R) Xeon(R) Gold 6338N CPU @ 2.20GHz
BenchmarkWrite-16            	 1000000	      1143 ns/op	       1 B/op	       0 allocs/op
BenchmarkParallelWrite-16    	 1000000	      1362 ns/op	       1 B/op	       0 allocs/op
PASS
ok  	github.com/nipuntalukdar/rollingwriter	2.596s

```

## Quick Start
```golang
	writer, err := rollingwriter.NewWriterFromConfig(&config)
	if err != nil {
		panic(err)
	}

	writer.Write([]byte("hello, world"))
```
or 
```golang
	writer, err := rollingwriter.NewWriterFromConfigFile("/path/to/config.json")
	if err != nil {
		panic(err)
	}

	writer.Write([]byte("hello, world"))
```
For details, check `demo` folder for more details. 
Detailded examples with confifg file are given.
To run the examples:
```bash
cd demo
go build
# We will get the demo executable.
./demo --help
Usage of ./demo:
  -configfile string
    	The config JSON file to use (default "config.json")
  -configfromfile
    	Read config from file
# An example run
./demo --configfromfile --configfile config.json

```
An example for logger integration with [hashicorp logger](https://github.com/hashicorp/go-hclog) is available [here](https://github.com/nipuntalukdar/NipunTalukdarExamples/tree/master/go/logrolling).
Any suggestion or new features needed, please create an [issue](https://github.com/nipuntalukdar/rollingWriter/issues/new) .
