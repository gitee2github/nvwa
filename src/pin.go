package main

import "os"

func pinMemoryClear() error {
	err, _ := runCmd("/usr/bin/nvwa-pin", []string{"--clear-pin-mem"},
		os.Stdin, os.Stdout, os.Stderr)
	return err
}

func pinMemoryFinish() error {
	err, _ := runCmd("/usr/bin/nvwa-pin", []string{"--finish-pin"},
		os.Stdin, os.Stdout, os.Stderr)
	return err
}
