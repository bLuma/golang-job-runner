package main

import (
	"log"
	"os/exec"
	"strconv"
)

func testIfProcessIsRunning(pid int) bool {
	//out, err := exec.Command("kill", "-s", "0", strconv.Itoa(pid)).CombinedOutput()
	out, err := exec.Command("ps", "-o", "stat", "h", strconv.Itoa(pid)).CombinedOutput()

	if err != nil {
		// log.Println(err)
		return false
	}

	if string(out) == "" || out[0] == 'Z' || out[0] == 'z' {
		return false
	}

	// if string(out) == "" {
	// 	return true // pid exist
	// }
	return true
}

func killProcess(pid int) {
	_, err := exec.Command("kill", "-s", "9", strconv.Itoa(pid)).CombinedOutput()
	if err != nil {
		log.Println(err)
	}
}
