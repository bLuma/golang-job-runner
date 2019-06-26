package main

import (
	"log"

	pp "github.com/shirou/gopsutil/process"
)

func testIfProcessIsDead(job *jobInfo) bool {
	p, err := pp.NewProcess(int32(job.PID))
	if err != nil {
		log.Println("Unable to create process for -", job.PID)
		return false
	}

	cpu, err := p.CPUPercent()
	if err != nil {
		log.Println("Unable to query cpupercent for -", job.PID)
		return false
	}

	if cpu < float64(client.MinimumCPUPercent) {
		job.DeadCounter++
	} else {
		job.DeadCounter = 0
	}

	// processMemory, err := p.MemoryInfo()
	// if err != nil {
	// 	log.Println("Unable to query memoryinfo for -", job.PID)
	// 	return false
	// }
	//used := processMemory.RSS

	return job.DeadCounter > 4*client.KillDeadProcessAfterMinutes // 4 - minute, 4*5 = 20
}

func getJobStats(job jobInfo) (int64, int64) {
	p, err := pp.NewProcess(int32(job.PID))
	if err != nil {
		log.Println("Unable to create process for -", job.PID)
		return 0, 0
	}

	cpu, err := p.CPUPercent()
	if err != nil {
		log.Println("Unable to query cpupercent for -", job.PID)
		return 0, 0
	}

	processMemory, err := p.MemoryInfo()
	if err != nil {
		log.Println("Unable to query memoryinfo for -", job.PID)
		return 0, 0
	}
	used := processMemory.RSS

	return int64(cpu * 100), int64(used)
}
