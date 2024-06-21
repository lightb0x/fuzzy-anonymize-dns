# fuzzy-anonymize-dns
This repository provide proof-of-concept of fuzzy-anonymize DNS

## outline
DNS works in hierarchical way as follows:
* client
* recursive resolver
* iterative resolver (omitted afterwards for simplicity)

Current works on anonymizing DNS propose `relay` in between client and recursive resolver:
* client
* relay
* recursive resolver

`relay` is contradictory in that:
* protocol do not trust `relay`
* protocol ensures anonymity as much as `relay` (does not collude with recursive resolver)

`relay` also add latency in that:
* additional network hop is required to DNS query
* recursive resolver provide target address geographically adjacent to `relay` rather than to client

On the other hand, [`dnscrypt-proxy`](https://github.com/DNSCrypt/dnscrypt-proxy) propose local DNS proxy (or `proxy` in short). `proxy` achieves:
* easy setup for client (e.g., query 127.0.0.1 without encryption)
* ensure encryption on communication from local to recursive resolver (e.g., dnscrypt)
* cache DNS queries locally
* queries from clients are grouped as one

Idea is to extend `proxy` to make groups of `proxy`. As group size increase, it is harder to distinguish client to client and `proxy` to `proxy`.

Once cache miss occurs, it will be broadcasted to random number of random other `proxy` to update cache and to replicate query to recursive resolver.

Proposal achieves follows:
* DNS queries are fuzzily anonymized
  * recursive resolver cannot easily distinguish distribution of DNS queries of individual client
  * `proxy` cannot easily distinguish distribution of DNS queries of other `proxy` following randomness
* less cold cache misses
  * cache miss of other `proxy` randomly updates other `proxy`s
  * especially beneficial to not-so-frequently-used `proxy` and often-used domain name
* no latency added
  * no additional network hop is required
    * do not wait for response of other `proxy`
    * call to recursive resolver is handled in parallel
  * every `proxy` maintain geographically adjacent target addresses for domain names

