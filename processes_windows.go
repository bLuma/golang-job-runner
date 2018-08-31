package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

func testIfProcessIsRunning(pid int) bool {
	// tasklist /fi "PID eq 86483" /fo csv /nh
	args := []string{
		"/fi", // filter
		fmt.Sprintf("PID eq %d", pid),
		"/fo", // output format
		"csv", // CSV
		"/nh", // do not include table header
	}
	cmd := exec.Command("tasklist", args...)
	var b bytes.Buffer
	cmd.Stdout = &b

	err := cmd.Run()
	if err != nil {
		log.Println("testIfProcessIsRunning", err)
		return false
	}

	bs := b.String()
	// log.Println(bs)

	if bs[0:5] == "INFO:" {
		// log.Println("info - not running!")
		return false
	}

	return true
}

func killProcess(pid int) {
	args := []string{
		"/fi", // filter
		fmt.Sprintf("PID eq %d", pid),
		"/t", // and children processes
		"/f", // forcefully
	}
	cmd := exec.Command("taskkill", args...)
	// var b bytes.Buffer
	// cmd.Stdout = &b

	err := cmd.Run()
	if err != nil {
		log.Println("killProcess", err)
	}
}
