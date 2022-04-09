:- built_in(probe/3).
probe(Proxy, Target, Options) :-
    probe(Proxy, Target, Options, Status),
    Status // 100 =:= 2.

:- built_in(probe/2).
probe(Proxy, Target) :-
	probe(Proxy, Target, []).

:- built_in(mod/3).
mod(N, List, Elem) :-
	length(List, L),
	M is N mod L,
	nth0(M, List, Elem).
