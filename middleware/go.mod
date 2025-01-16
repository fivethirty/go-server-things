module github.com/fivethirty/go-server-things/middleware

go 1.23.1

require (
	github.com/fivethirty/go-server-things/logs v0.0.1
	github.com/google/uuid v1.6.0
)

replace github.com/fivethirty/go-server-things/logs => ../logs
