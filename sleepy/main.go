package main

import (
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	val, err := strconv.ParseInt(os.Args[1], 10, 32)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	log.Println("Sleepy for", val, "seconds")

	end := time.Now().Add(time.Second * time.Duration(val))
	i := 0
	for {
		i++
		if time.Now().After(end) {
			break
		}
	}

	// <-time.After(time.Second * time.Duration(val))
	log.Println("Woke up!", i)
}
