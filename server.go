package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"sync"
	"time"
)

const (
	serverDatabaseFile = "db.json"
	listenAddress      = ":8088"
)

type serverData struct {
	Available map[ConfigurationID]Configuration
	Leased    map[ConfigurationID]LeasedConfiguration
	Finished  map[ConfigurationID]FinishedConfiguration

	lock sync.Mutex
}

func newServerData() *serverData {
	sd := new(serverData)
	sd.Available = make(map[ConfigurationID]Configuration)
	sd.Leased = make(map[ConfigurationID]LeasedConfiguration)
	sd.Finished = make(map[ConfigurationID]FinishedConfiguration)
	sd.lock = sync.Mutex{}

	return sd
}

var server *serverData

func startServer() {
	log.Println("Starting server")

	server = newServerData()

	// server.lock.Lock()
	// tmpid := GetNewID()
	// server.Available[tmpid] = Configuration{tmpid, []string{"aaa", "bbb"}, 10}
	// tmpid = GetNewID()
	// server.Available[tmpid] = Configuration{tmpid, []string{"ccc", "ddd"}, 100}
	// tmpid = GetNewID()
	// server.Leased[tmpid] = LeasedConfiguration{Configuration{tmpid, []string{"ccc", "ddd"}, 1000}, "hostik", time.Now()}
	// server.lock.Unlock()
	tryLoadData()

	go routineLeasedReleaser()
	go routineDatabaseSaver()

	http.HandleFunc("/dump", dumpHandler)
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/request", requestHandler)
	http.HandleFunc("/finish", finishedHandler)
	http.HandleFunc("/timeout", timeoutHandler)

	log.Fatal(http.ListenAndServe(listenAddress, nil))
}

func tryLoadData() {
	data, err := ioutil.ReadFile(serverDatabaseFile)
	if err != nil {
		log.Println(err)
		return
	}

	if !unmarshalJSON(data, server) {
		log.Fatalln(serverDatabaseFile, "loading error (tryLoadData")
	}
}

func routineDatabaseSaver() {
	for {
		<-time.After(time.Second * 15)

		server.lock.Lock()
		f, err := os.OpenFile(serverDatabaseFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalln(err)
			server.lock.Unlock()
			continue
		}

		if !marshalJSONAndWrite(f, server) {
			log.Fatalln("Failed write of db!!!")
			server.lock.Unlock()
			continue
		}

		f.Close()
		server.lock.Unlock()
	}
}

func routineLeasedReleaser() {
	<-time.After(time.Second * 4)

	for {
		server.lock.Lock()
		now := time.Now()

		for k, v := range server.Leased {
			if now.After(v.Expiry) {
				delete(server.Leased, k)
				server.Available[k] = v.Configuration

				log.Println("Job", k, "status=available (processing time expired)")
			}
		}
		server.lock.Unlock()

		<-time.After(time.Second * 30)
	}
}

func dumpHandler(w http.ResponseWriter, req *http.Request) {
	marshalJSONAndWrite(w, server)
}

func requestHandler(w http.ResponseWriter, req *http.Request) {
	hostname := req.URL.Query().Get("hostname")
	if len(hostname) == 0 {
		hostname = "<unknown>"
	}

	server.lock.Lock()
	defer server.lock.Unlock()

	if len(server.Available) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var i ConfigurationID
	for i = range server.Available {
		break
	}

	conf := server.Available[i]
	delete(server.Available, i)
	server.Leased[i] = LeasedConfiguration{conf, hostname, time.Now(), time.Now().Add(time.Second*time.Duration(conf.Runtime) + time.Minute*10)}

	marshalJSONAndWrite(w, conf)

	log.Println("Job", conf.ID, "status=leased (", hostname, ")")
}

func finishedHandler(w http.ResponseWriter, req *http.Request) {
	server.lock.Lock()
	defer server.lock.Unlock()

	hostname := req.URL.Query().Get("hostname")
	if len(hostname) == 0 {
		hostname = "<unknown>"
	}

	configuration := req.URL.Query().Get("configuration")
	if len(configuration) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var conf Configuration
	if !unmarshalJSON([]byte(configuration), &conf) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	knownConfiguration, ok := server.Leased[conf.ID]
	if !ok || !reflect.DeepEqual(knownConfiguration.Configuration, conf) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	delete(server.Leased, conf.ID)
	server.Finished[conf.ID] = FinishedConfiguration{conf, hostname, knownConfiguration.StartTime, time.Now(), "finished"}
	log.Println("Job", conf.ID, "status=finished (", hostname, ")")
}

func timeoutHandler(w http.ResponseWriter, req *http.Request) {
	server.lock.Lock()
	defer server.lock.Unlock()

	hostname := req.URL.Query().Get("hostname")
	if len(hostname) == 0 {
		hostname = "<unknown>"
	}

	configuration := req.URL.Query().Get("configuration")
	if len(configuration) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var conf Configuration
	if !unmarshalJSON([]byte(configuration), &conf) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	knownConfiguration, ok := server.Leased[conf.ID]
	if !ok || !reflect.DeepEqual(knownConfiguration.Configuration, conf) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	delete(server.Leased, conf.ID)
	server.Finished[conf.ID] = FinishedConfiguration{conf, hostname, knownConfiguration.StartTime, time.Now(), "timeout"}
	log.Println("Job", conf.ID, "status=timeout (", hostname, ")")
}

func addHandler(w http.ResponseWriter, req *http.Request) {
	server.lock.Lock()
	defer server.lock.Unlock()

	configuration := req.URL.Query().Get("configuration")
	if len(configuration) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var conf Configuration
	if !unmarshalJSON([]byte(configuration), &conf) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for {
		conf.ID = GetNewID()
		if !isConfigurationIDUsed(conf.ID) {
			break
		}
	}

	server.Available[conf.ID] = conf
	log.Println("New job", conf.ID, "status=available")
}

func isConfigurationIDUsed(cid ConfigurationID) bool {
	if _, ok := server.Available[cid]; ok {
		return true
	}
	if _, ok := server.Leased[cid]; ok {
		return true
	}
	if _, ok := server.Finished[cid]; ok {
		return true
	}

	return false
}
