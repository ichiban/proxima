/*
 * Proxima Configuration
 */

%% Server Settings

% a TCP address to listen to.
listen(':8080').

%% Probing Settings

probe_url('https://probe.example.com/ok.txt').

%% Routing Rules

tunnel('fast-proxy.example.com:8080', Options) :-
	member(fast, Options).

tunnel('limited-target-proxy.example.com:8080', Options) :-
	member(target('target.example.com:443'), Options).

tunnel(Proxy, Options) :-
	uri_template('{id}:{pass}@auth-proxy.example.com:8080', Options, Proxy).

tunnel(Proxy, _) :-
	round_robin(Proxy, ['localhost:8081', 'localhost:8082', 'localhost:8083']),
    probe(Proxy).
