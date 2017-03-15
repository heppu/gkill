package main

import (
	"fmt"

	"github.com/heppu/gkill/killer"
)

func main() {
	k, err := killer.NewKiller()
	if err != nil {
		fmt.Print(err)
	}

	if err = k.Start(); err != nil {
		fmt.Print(err)
	}
}
