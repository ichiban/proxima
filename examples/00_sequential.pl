% The proxy manager will be available at localhost:8080.
%   curl -x localhost:8080 https://httpbin.org/ip
listen(':8080').

% Tries the proxies localhost:8081, localhost:8082, and localhost:8083 in that order and use the first one working.
tunnel('localhost:8081', _).
tunnel('localhost:8082', _).
tunnel('localhost:8083', _).
