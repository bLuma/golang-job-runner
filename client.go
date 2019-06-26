package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type jobInfo struct {
	Configuration
	PID        int
	StartTime  time.Time
	ExpiryTime time.Time
	Cmd        *exec.Cmd

	DeadCounter int
}

type clientData struct {
	Hostname        string
	MaximumJobCount int
	ServerAddress   string
	SafeExpression  string

	MaxMemoryLimit              int
	MinimumCPUPercent           int
	KillDeadProcessAfterMinutes int

	Jobs map[int]jobInfo

	Lock sync.Mutex
}

func newClientData() *clientData {
	data := new(clientData)
	data.Jobs = make(map[int]jobInfo)
	data.Lock = sync.Mutex{}

	return data
}

var client *clientData

func startClient() {
	client = newClientData()

	flag.Bool("client", true, "Sets client mode")
	flag.StringVar(&client.Hostname, "hostname", "<unk>", "Client hostname")
	flag.IntVar(&client.MaximumJobCount, "maxjobs", 1, "Maximal concurrent job count")
	// TODO: remove HACK for modifying of java memory limit
	flag.IntVar(&client.MaxMemoryLimit, "memory", 32, "Maximum JRE memory for heap in GBs")
	flag.IntVar(&client.MinimumCPUPercent, "minimumcpupercent", 3, "Process must use minimum of X % of CPU to consider as alive")
	flag.IntVar(&client.KillDeadProcessAfterMinutes, "killdeadprocessafterminutes", 5, "Kill process after X minutes of dead state")
	flag.StringVar(&client.ServerAddress, "master", "http://localhost:8088", "Master address")
	flag.StringVar(&client.SafeExpression, "safeexpr", "", "Safe expression") // "java -cp sleepyjava.jar"
	workingFolder := flag.String("workdir", "./", "Sets working folder")

	flag.Parse()

	if client.ServerAddress[len(client.ServerAddress)-1] == '/' {
		client.ServerAddress = client.ServerAddress[:len(client.ServerAddress)-1]
	}

	log.Println("Starting client -", client.ServerAddress)
	log.Println("Hostname:", client.Hostname)
	log.Println("Max jobs:", client.MaximumJobCount)
	log.Println("SafeExpression:", client.SafeExpression)

	if workingFolder != nil {
		log.Println("Changed workdir", *workingFolder)
		if err := os.Chdir(*workingFolder); err != nil {
			log.Println("Cant set working folder of", *workingFolder, err)
		}
	}

	go routineJobRequester()
	go routineJobChecker()
	go routineUsageLogger()

	ch := make(chan struct{})
	<-ch
}

func routineUsageLogger() {
	<-time.After(time.Second * 5)

	for {
		client.Lock.Lock()
		for _, job := range client.Jobs {
			cpu, mem := getJobStats(job)
			go processLogUsageFor(job.Configuration, cpu, mem)
		}
		client.Lock.Unlock()

		<-time.After(time.Second * 60)
	}
}

func routineJobChecker() {
	<-time.After(time.Second * 5)

	for {
		client.Lock.Lock()
		for _, job := range client.Jobs {
			switch {
			case time.Now().After(job.ExpiryTime):
				go processTimeoutedJob(job)

			case !testIfProcessIsRunning(job.PID):
				go processFinishedJob(job)

			case testIfProcessIsDead(&job):
				go processDeadJob(job)
			}
		}
		client.Lock.Unlock()

		<-time.After(time.Second * 15)
	}
}

func processTimeoutedJob(job jobInfo) {
	log.Println("processTimeoutedJob", job.ID, "PID:", job.PID)
	killProcess(job.PID)
	err := sendJobFinishNotification(job.Configuration, false)

	if err == nil {
		client.Lock.Lock()
		delete(client.Jobs, job.PID)
		client.Lock.Unlock()
	} else {
		log.Println(err)
	}
	// else let job stay in map and try to delete it on next cycle
}

func processFinishedJob(job jobInfo) {
	log.Println("processFinishedJob", job.ID, "PID:", job.PID)
	err := sendJobFinishNotification(job.Configuration, true)

	if err == nil {
		client.Lock.Lock()
		delete(client.Jobs, job.PID)
		client.Lock.Unlock()
	}
	// else let job stay in map and try to delete it on next cycle
}

func processDeadJob(job jobInfo) {
	log.Println("processDeadJob", job.ID, "PID:", job.PID)
	killProcess(job.PID)
	err := sendJobDeadNotification(job.Configuration)

	if err == nil {
		client.Lock.Lock()
		delete(client.Jobs, job.PID)
		client.Lock.Unlock()
	}
	// else let job stay in map and try to delete it on next cycle
}

func routineJobRequester() {
	<-time.After(time.Second * 2)

	for {
		client.Lock.Lock()
		shouldGetANewJob := len(client.Jobs) < client.MaximumJobCount
		client.Lock.Unlock()

		if shouldGetANewJob {
			conf, err := requestNewJob()
			if err != nil {
				log.Println("Request job error", err)
				<-time.After(time.Minute * 5)
				continue
			}

			if conf != nil {
				go launchNewJob(conf)
			}
		}

		<-time.After(time.Second * 30)
	}
}

func launchNewJob(conf *Configuration) {
	log.Println("launchNewJob", conf.ID, *conf)

	if !satisfiesSafeExpression(conf) {
		log.Println("Ignoring job", conf.ID, "safe condition not satisfied")
		return
	}

	// TODO: remove this HACK, modifies configuration, changes java memory limit
	for idx, value := range conf.Params {
		if value == "-Xmx32g" {
			conf.Params[idx] = fmt.Sprintf("-Xmx%dg", client.MaxMemoryLimit)
		}
	}

	var job jobInfo
	job.Configuration = *conf
	job.StartTime = time.Now()
	job.ExpiryTime = time.Now().Add(time.Second*time.Duration(conf.Runtime) + time.Minute*2)

	executable := job.Configuration.Params[0]
	parameters := job.Configuration.Params[1:]

	cmd := exec.Command(executable, parameters...)
	err := cmd.Start()
	if err != nil {
		log.Fatalln("Couldnt start job", *conf, err)
		return
	}

	job.PID = cmd.Process.Pid
	job.Cmd = cmd

	client.Lock.Lock()
	client.Jobs[job.PID] = job
	client.Lock.Unlock()
}

func satisfiesSafeExpression(conf *Configuration) bool {
	if len(client.SafeExpression) == 0 {
		return true
	}

	command := strings.Join(conf.Params, " ")
	return strings.HasPrefix(command, client.SafeExpression)
}

func requestNewJob() (*Configuration, error) {
	u, err := url.Parse(fmt.Sprintf("%s/request", client.ServerAddress))
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("hostname", client.Hostname)

	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil // errors.New("No new jobs available")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Invalid response status code")
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var conf Configuration
	if !unmarshalJSON(body, &conf) {
		return nil, err
	}

	return &conf, nil
}

func sendJobFinishNotification(conf Configuration, success bool) error {
	action := "finish"
	if !success {
		action = "timeout"
	}

	// log.Println("job", conf, action)

	u, err := url.Parse(fmt.Sprintf("%s/%s", client.ServerAddress, action))
	if err != nil {
		return err
	}

	q := u.Query()
	q.Set("hostname", client.Hostname)
	q.Set("configuration", marshalJSONAsString(conf))

	u.RawQuery = q.Encode()

	r, err := http.Get(u.String())
	// log.Println("ret:", r.StatusCode, r.Status)

	if err != nil {
		return err
	}

	if r.StatusCode != http.StatusOK {
		return errors.New("Unexpected status code on finish notification")
	}

	return nil
}

func sendJobDeadNotification(conf Configuration) error {
	action := "dead"
	// log.Println("job", conf, action)

	u, err := url.Parse(fmt.Sprintf("%s/%s", client.ServerAddress, action))
	if err != nil {
		return err
	}

	q := u.Query()
	q.Set("hostname", client.Hostname)
	q.Set("configuration", marshalJSONAsString(conf))

	u.RawQuery = q.Encode()

	r, err := http.Get(u.String())
	// log.Println("ret:", r.StatusCode, r.Status)

	if err != nil {
		return err
	}

	if r.StatusCode != http.StatusOK {
		return errors.New("Unexpected status code on dead notification")
	}

	return nil
}

func processLogUsageFor(conf Configuration, cpu int64, mem int64) {
	u, err := url.Parse(fmt.Sprintf("%s/usage", client.ServerAddress))
	if err != nil {
		return
	}

	q := u.Query()
	q.Set("hostname", client.Hostname)
	q.Set("cpu", strconv.FormatInt(cpu, 10))
	q.Set("mem", strconv.FormatInt(mem, 10))
	q.Set("configuration", marshalJSONAsString(conf))

	u.RawQuery = q.Encode()

	r, err := http.Get(u.String())
	// log.Println("ret:", r.StatusCode, r.Status)

	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		return //errors.New("Unexpected status code on finish notification")
	}
}
