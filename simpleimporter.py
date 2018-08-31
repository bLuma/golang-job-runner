import json
import urllib.request

repls = [169,170,171,173,174,175,176,178,179,180,183,185,186,187,189,195,198,199]

runtime = 60*60*24*3

for repl in repls:
    conf = {
        "Params": ["java", "-jar", "job.jar", str(repl)],
        "Runtime": runtime
    }

    conf = json.dumps(conf)
    print(conf)
    url = "http://localhost:8088/add?"+ urllib.parse.urlencode({"configuration": conf})
    f = urllib.request.urlopen(url)
    print(f.read())


