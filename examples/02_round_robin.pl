% The proxy manager will be available at localhost:8080.
%   curl -x localhost:8080 https://httpbin.org/ip
listen(':8080').

% Tries one of the proxies localhost:8081, localhost:8082, and localhost:8083.
tunnel(Proxy, Options) :-
    member(rid(ID), Options),
    mod(ID, ['localhost:8081', 'localhost:8082', 'localhost:8083'], Proxy).
