% The proxy manager will be available at localhost:8080.
%   curl -x one,two,three@localhost:8080 https://httpbin.org/ip
listen(':8080').

% Similar to 00_sequential.pl, but filters out proxies by tags.
% localhost:8081 is only available if you supply `one` in proxy URL's userinfo subcomponent
%   curl -x one@localhost:8080 https://httpbin.org/ip
% If you supply multiple tags, it tries the proxies that match any of the tags.
% For example, if you supply `one,three` (no spaces allowed), it tries localhost:8081 and localhost:8083.
%   curl -x one,three@localhost:8080 https://httpbin.org/ip
tunnel('localhost:8081', Options) :- member(one, Options).
tunnel('localhost:8082', Options) :- member(two, Options).
tunnel('localhost:8083', Options) :- member(three, Options).
