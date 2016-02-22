# tcp-diaper

A simple tcp forwarder for a stupid stupid stupid legacy software

Why?
====

This utility is born out of frustration from a legacy software that nobody
can ever maintain or rewrite that runs on embedded (and pretty important) devices.

This legacy software has a small problem: it does not free the file descriptor
and resources associated with a TCP connection, so the first connection works
as expected, but any following one remains forever stuck, waiting. Only a restart of
the software allows a single new one connection.

What?
=====

This program listens on an address on the host on a number of ports and forwards
any data coming in or out of an incoming connection to a destination address on the same port.
The connection to the destination is cached and is NEVER closed, with all the usual caveats
that this entails.

How?
====

The parameters are:
- m: the minimum port to listen to
- M: the maximum port to listen to
- p: comma separated list of ports to use
- b: the address on which to bind (usually the public interface, usually eth0)
- d: the destination address to send traffic to, usually 127.0.0.1


- k: enable keepalive for the target connection(s)
- K: time between keepalive packets
- C: number of failed probes before dropping the connection
- I: idle time before starting keepalive probes


The program then stops waiting for incoming connections until terminated
