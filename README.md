# Proxima: a Proxy Manager built in Go, configured in Prolog

## What is this?

**Proxima** is the only reasonable proxy manager.

- Works with any HTTP proxies
- Supports a variety of HTTP proxy providers such as Bright Data (formerly known as Luminati), Oxylabs, GeoSurf, Smartproxy, and so on
- Incredibly flexible

## Usage

### Install the latest version

You can install it with `go install`. Make sure you have Go 1.7 or above installed.

```console
go install github.com/ichiban/proxima/cmd/proxima@latest
```

### Prepare a config file

You can find more examples in `examples/`.

```console
$ cat << EOF > config.pl
listen(':8080').
tunnel('localhost:8081', Options) :- member(one, Options).
tunnel('localhost:8082', Options) :- member(two, Options).
tunnel('localhost:8083', Options) :- member(three, Options).
EOF
```

### Run the proxy manager

```console
$ $(go env GOPATH)/bin/proxima config.pl
8:42PM INF Start addr=:8080
```

### Make an HTTP request via the proxy manager

Assuming you're running a proxy at 'localhost:8083', a request via the proxy manager results in success.

```console
$ curl -I -x one,three@localhost:8080 https://httpbin.org/status/200
HTTP/1.1 200 OK
Date: Mon, 04 Apr 2022 11:52:52 GMT
Content-Length: 0

HTTP/2 200 
date: Mon, 04 Apr 2022 11:52:53 GMT
content-type: text/html; charset=utf-8
content-length: 0
server: gunicorn/19.9.0
access-control-allow-origin: *
access-control-allow-credentials: true

```

The log tells the first trial via `localhost:8081` failed since it's not running a proxy, but the second trial via `localhost:8083` succeeded since it's running.
It didn't try `localhost:8082` since the given options `one,three` didn't match the tag `two`.

```console
$ go run cmd/proxima/main.go config.pl
8:52PM INF Start addr=:8080
8:52PM WRN net.Dial() failed error="dial tcp [::1]:8081: connect: connection refused" proxy=localhost:8081 remote=[::1]:52888 rid=c95do52s1s437fk32dpg ua=curl/7.77.0
8:52PM INF tunnel start proxy=localhost:8083 remote=[::1]:52888 rid=c95do52s1s437fk32dpg ua=curl/7.77.0
8:52PM INF tunnel finish proxy=localhost:8083 remote=[::1]:52888 rid=c95do52s1s437fk32dpg ua=curl/7.77.0
```

## Built-in predicates

