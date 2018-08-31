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

**Client:**
- periodically polls server for new jobs
- runs up to X jobs simultaneously (configurable)
- if job exhausted specified time limit, job is killed and server got a "timeout" signal
- if job is sucessfully finished, then client signals a "success" to the server

**Current project limitations:**
- no authentication/authorization between servers/clients (only "security" feature is ability to limit running of specified jobs - defining job command line prefix)
- no HTTPS
- clients forgot their jobs if restarted (jobs will wait in leased state)
- only windows clients (task operations uses windows "taskkill" and "tasklist" utilities)
- only job "walltime" is currently monitored and used for limiting job resources
- no server job queues, jobs are handled in pseudorandom order
