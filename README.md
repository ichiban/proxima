# Proxima: a Proxy Manager built in Go, configured in Prolog

## What is this?

**Proxima** is an intelligent proxy manager that lets you get the most out of HTTP proxies and proxy providers.

- Rule-based proxy selection
- Filters out unavailable / malicious proxies
- Rotates proxies
- Works with any HTTP tunneling proxies
- Supports a variety of proxy providers such as Bright Data (formerly known as Luminati), Oxylabs, GeoSurf, Smartproxy, and so on

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

## How Proxima queries the configuration file

### On startup 

Proxima queries the configuration file with `listen(Addr).` for the address to wait for incoming `CONNECT` requests.

### On each `CONNECT` request

Proxima queries the configuration file with `tunnel(Proxy, Options).` to filter out proxies and use the first one to which Proxima actually succeeds on connecting.

`Options` is a list of:
- `rid(ID)`: `ID` is an integer ID for the `CONNECT` request
- `remote(Addr)`: `Addr` is an atom that represents the address of the client 
- `target(Addr)`: `Addr` is an atom that represents the address of the server
- anything passed in the userinfo subcomponent

## Built-in predicates

The Prolog processor is based on [`ichiban/prolog`](https://github.com/ichiban/prolog) extended by the custom built-in predicates listed below.

### `host_port/3`

`host_port(HostPort, Host, Port)` succeeds iff the atom `HostPort` is the concatenation of the atom `Host` and the integer `Port`.
It can be used either to break down `HostPort` into `Host` and `Port` or to construct `HostPort` out of `Host` and `Port`.

### `uri_template/3` 

`uri_template(Template, Pairs, URI)` applies `Key-Value` pairs in the list `Pairs` to the atom `Template` which is a [URI Template described in RFC6570](https://datatracker.ietf.org/doc/html/rfc6570/) and unifies the result with `URI`.

This is useful especially when you're working with a proxy provider that has session ID functionality. See `examples/03_uri_template.pl`. 

### `probe/4` 

`probe(Proxy, Target, Options, Status)` probes the availability of `Proxy` by making an HTTP GET request to the URL `target` and succeeds if the resulting status code unifies with `Status`.

`Options` is a proper list containing:
- `Header-Values` where `Header` is an atom and `Values` is a list of atoms

### `probe/3` 

`probe(Proxy, Target, Options)` is similar to `proby(Proxy, Target, Options, Status)` but succeeds only if `Status` is a successful status code 2XX.

### `probe/2`

`probe(Proxy, Target)` is same as `probe(Proxy, Target, [])`. See `examples/05_probe.pl`.

### `log/3`

`log(Level, Message, Pairs)` outputs a structured log to stderr. `Level` must be one of the log levels listed below. `Message` is the message portion of the log. `Pairs` is the list of `Key-Value` pairs in the structured log.

Log levels are:
- `debug`
- `info`
- `warn`
- `error`

### `mod/3`

`mod(N, List, Elem)` is similar to `nth0(N, List, Elem)` but `N` can be greater than the length of `List`. In that case, `N` will be replaced by the remainder of the division of `N` by the length of `List`.

Combined with `member(rid(N), Options)`, you can implement a round-robin scheduler for a list of proxies. See `examples/02_round_robin.pl`.


