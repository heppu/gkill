package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/heppu/gkill/killer"
)

func main() {
	var filter string
	if len(os.Args) > 1 {
		filter = strings.Join(os.Args[1:], " ")
	}

	k, err := killer.NewKiller(filter)
	if err != nil {
		fmt.Print(err)
	}

	if err = k.Start(); err != nil {
		fmt.Print(err)
	}
}
