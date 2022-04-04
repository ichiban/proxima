% The proxy manager will be available at localhost:8080.
%   curl -x id-foo,pass-bar,port-8082@localhost:8080 https://httpbin.org/ip
listen(':8080').

% Tries the proxy that is specified by the URI template and Key-Value pairs in the proxy URL's userinfo subcomponent.
% The template '{id}:{pass}@localhost:{port}' and `id-foo,pass-bar,port-8082` will make `foo:bar@localhost:8082`.
tunnel(Proxy, Options) :-
	uri_template(Proxy, '{id}:{pass}@localhost:{port}', Options).
