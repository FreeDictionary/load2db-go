package main

import (
	"log"

	"github.com/FreeDictionary/load2db-go/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalln("Failed to parse command line arguments: ", err)
	}
}
