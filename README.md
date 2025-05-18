# cron

# Usage
> sample/scheduler.go

# Usage with webserver
> mock_server/main

```bash
# List added function
curl http://localhost:8000/list

# Update function spec
http://localhost:8000/update?name=health_check&newSpec=@every%2010s

```
