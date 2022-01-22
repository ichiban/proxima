:- built_in(probe/2).
probe(Proxy, Target) :-
	catch(http_get(Target, [], Proxy, response(Status, _, _)), error(Type, Detail), (
		log(error, 'probe', [proxy-Proxy, target-Target, type-Type, detail-Detail]),
		false
	)),
	2 is Status // 100.

:- built_in(probe/1).
probe(Proxy) :-
    probe_url(Target),
	probe(Proxy, Target).

:- built_in(round_robin/2).
round_robin(Elem, List) :-
	request_counter(C),
	length(List, N),
	M is (C mod N) + 1,
	nth(M, List, Elem).

:- built_in(nth/3).
nth(1, [Elem|_], Elem) :- !.
nth(N, [_|Rest], Elem) :-
	N > 1,
	M is N - 1,
	nth(M, Rest, Elem).