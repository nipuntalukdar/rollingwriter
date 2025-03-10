package main

import (
	"flag"
	"fmt"
)

func main() {
	var configfile string
	var configfromfile bool

	flag.StringVar(&configfile, "configfile", "config.json", "The config JSON file to use")
	flag.BoolVar(&configfromfile, "configfromfile", false, "Read config from file")

	flag.Parse()

	if configfromfile {
		fmt.Println("Config File:", configfile)
		run_wrtiter_config_from_file(configfile)
	} else {
		run_writer()
	}
}
