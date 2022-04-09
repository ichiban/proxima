% The proxy manager will be available at localhost:8080.
%   curl -x localhost:8080 https://httpbin.org/ip
listen(':8080').

% Similar to 00_sequential.pl, but filters out proxies by their probing results.
tunnel(Proxy, _) :-
    Proxy = 'localhost:8081',
    probe(Proxy, 'https://httpbin.org/status/404').

tunnel(Proxy, _) :-
    Proxy = 'localhost:8082',
    probe(Proxy, 'https://httpbin.org/status/404').

tunnel(Proxy, _) :-
    Proxy = 'localhost:8083',
    probe(Proxy, 'https://httpbin.org/status/200').
