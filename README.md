# RollingWriter [![Build Status](https://travis-ci.org/arthurkiller/rollingWriter.svg?branch=master)](https://travis-ci.org/arthurkiller/rollingWriter) [![Go Report Card](https://goreportcard.com/badge/github.com/arthurkiller/rollingwriter)](https://goreportcard.com/report/github.com/arthurkiller/rollingwriter) [![GoDoc](https://godoc.org/github.com/arthurkiller/rollingWriter?status.svg)](https://godoc.org/github.com/arthurkiller/rollingWriter) [![codecov](https://codecov.io/gh/arthurkiller/rollingwriter/branch/master/graph/badge.svg)](https://codecov.io/gh/arthurkiller/rollingwriter)
RollingWriter is an auto rotate io.Writer implementation. It always works with logger.

__New Version v2.0 is comming out! Much more Powerfull and Efficient. Try it by follow the demo__

it contains 2 separate patrs:
* Manager: decide when to rotate the file with policy
    RlingPolicy give out the rolling policy
    * WithoutRolling: no rolling will happen
    * TimeRolling: rolling by time
    * VolumeRolling: rolling by file size

* IOWriter: impement the io.Writer and do the io write
    * Writer: not parallel safe writer
    * LockedWriter: parallel safe garented by lock
    * AsyncWtiter: parallel safe async writer

## Features
* Auto rotate
* Parallel safe
* Implement go io.Writer
* Time rotate with corn style task schedual
* Volume rotate
* Max remain rolling files with auto clean

## Quick Start
```golang
	writer, err := rollingwriter.NewWriterFromConfig(&config)
	if err != nil {
		panic(err)
	}

	writer.Write([]byte("hello, world"))
```
Want more? View `demo` for more details.

## TODO
* the Buffered writer needs to be discussed
    ```
        ** Proposal about Buffered Writer **

        I planed to change Buffered Writer into a Batch Writer aimed to merge serval write operations
        into one for better io bandwidth.
    ```
* reconstruct the code
* implement interface

Any suggestion or new feature inneed, please [put up an issue](https://github.com/arthurkiller/rollingWriter/issues/new)
