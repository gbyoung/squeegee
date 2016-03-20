package main

import (
	"flag"
	"fmt"
	"github.com/gbyoung/squeegee"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
	// Use all the cores on this box, which is not default on older Go
	runtime.GOMAXPROCS(runtime.NumCPU())

	var confFile = "squeegee.conf"
	flag.StringVar(&confFile, "confFile", confFile, "The TOML configuration file")
	var clear = false
	flag.BoolVar(&clear, "clear", clear, "Clear the Cache")
	var debug = false
	flag.BoolVar(&debug, "debug", debug, "Log additional debug information")
	var help = false
	flag.BoolVar(&help, "help", help, "Print extended help")
    flag.Parse()

	squeegee := squeegee.Squeegee{}

	if err := squeegee.Init(confFile, debug); err != nil {
		fmt.Println("\nConfiguration:  " + err.Error() + "\n")
		return
	}

	if clear {
		squeegee.ClearCache()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT)

    squeegee.Start()

	// Block until a signal is received or we receive something on stopChan
	var msg string
	select {
	case s := <-c:
		msg = "Received " + s.String() + " signal.  Cleaning up and exiting...\n"
	case msg = <-squeegee.StopChan:
	}
	fmt.Println(msg)
}
