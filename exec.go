package main

import (
	"log"
	"os"
	"os/exec"
	"runtime"
)

func main() {
	cmd := exec.Command("ls", "-lah")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if runtime.GOOS == "windows" {
		cmd = exec.Command("tasklist")
	}

	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
}
