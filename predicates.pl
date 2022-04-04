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

:- built_in(mod/3).
mod(N, List, Elem) :-
	length(List, L),
	M is N mod L,
	nth0(M, List, Elem).
