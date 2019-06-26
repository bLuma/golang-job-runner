# golang-job-runner
Trivial job runner (executor), server-client architecture, inspired by Torque, PBSpro, only basic functionality.

Server acts as a passive agent. Clients get new jobs from server and notify server about job ends.

**Server:**
- contains DB of jobs (saved to json file)
- communication is handled by simple REST API
  - http://server:8088/dump - dumps server database
  - http://server:8088/add?configuration=json - adds new job to queue
  - http://server:8088/request?hostname=requesting_client - requests a new job from queue
  - http://server:8088/finish?hostname=requesting_client&configuration=job_json - client to server signal that job is finished
  - http://server:8088/timeout?hostname=requesting_client&configuration=job_json - client to server signal that job timeouted
  - http://server:8088/dead?hostname=requesting_client&configuration=job_json - client to server signal that job is dead (and was killed)  
  - http://server:8088/usage?cpu=cpu_utilization&mem=mem_utilization&configuration=job_json - client to server signal that logs actual usage of cpu and mem of specified job

**Client:**
- periodically polls server for new jobs
- runs up to X jobs simultaneously (configurable)
- if job exhausted specified time limit, job is killed and server got a "timeout" signal
- if job is sucessfully finished, then client signals a "success" to the server
- if job is using less than x % CPU for specified time then process is marked as dead and killed (optional feature)

**Current project limitations:**
- no authentication/authorization between servers/clients (only "security" feature is ability to limit running of specified jobs - defining job command line prefix)
- no HTTPS
- clients forgot their jobs if restarted (jobs will wait in leased state)
- only windows clients (task operations uses windows "taskkill" and "tasklist" utilities), similiar functionality now for linux platform using ps and kill tools
- only job "walltime" is currently monitored and used for limiting job resources, basic support for killing of dead processeses, very basic support for monitoring of cpu and memory utilization
- no server job queues, jobs are handled in pseudorandom order
