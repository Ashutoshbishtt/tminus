# ADR 0002: Build this in Go

**Status:** Accepted
**Date:** 23 July 2026

## Why

First thing to pick. The two real options were Go, which I write every day at work, and Java,
which I know but mostly use for design practice and DSA.

## Decision

Go.

**The work fits it.** Most of this system is thousands of connections that spend their time
waiting. Riders sending location pings. Customers watching a live map. Workers sitting on a
queue. In Go I can give each one its own goroutine, which costs about 2 KB to start, and the
code still reads top to bottom instead of turning into a pile of callbacks. Ten thousand riders
pinging every five seconds is around two thousand messages a second, plus a live connection for
every order being watched. Go is comfortable there.

**The simulator needs it.** The simulator has to run thousands of fake riders, each one moving
around and sending pings on its own clock. In Go that is a loop that starts thousands of
goroutines. That is basically the whole design.

**Timeouts run through everything here.** A rider gets 15 seconds to accept. An order has a ten
minute clock. A database call should give up early instead of hanging. Go has `context` for
this, and it works the same way across the standard library, the database driver and the queue
clients. One timeout can cancel a whole chain of work. It is just how Go is written, not
something I have to build myself.

**Short GC pauses.** I plan to publish p99 latency numbers, and Go's garbage collector keeps
pauses well under a millisecond without me tuning anything. The JVM can match that, but tuning
it is a skill of its own, and I do not want to spend this project explaining a GC spike.

**Nothing is hidden from me.** With Spring Boot I would put an annotation on a method and
messages would just show up. That is great at work and useless for learning. In Go I write the
consumer loop myself and decide when to commit the offset. Getting that wrong and watching a
message get handled twice is the point of this project.

**It fits on my laptop.** A Go service starts in milliseconds and uses tens of megabytes. Kafka
and the databases will eat most of the RAM, so the services need to fit in what is left.

## What else I considered

**Java.** The fair argument is that I would learn more simply because I know it less, and plenty
of the product companies I might interview at run on it.

I skipped it because what I want to learn here is not a language. Outbox, idempotency, consumer
groups, dead letter queues, delivery guarantees. Those ideas are the same in any language.
Picking the one I am slower in does not teach me more about them. It just means I reach them
later, and it raises the odds I stall out and never finish.

One thing I checked first: the old line about Go having lightweight threads and Java not having
them is out of date. Java has had virtual threads since Java 21 and they solve the same problem.
So that was never a reason to skip Java. The reasons above are.

## The cost

- Kafka's Java client is the reference one. New features land there first and are documented
  best there. The Go clients are good but they follow behind.
- No Kafka Streams or Flink, so proper stream processing is off the table if I ever want it.
- Java's profiling tools are still ahead of Go's, though pprof is good enough for me.
- For heavy CPU work the JVM would be faster. This system waits on the network far more than it
  does maths, so it does not bite.

Java keeps the place it already has: LLD practice, design patterns, and a separate Spring Boot
project later where the ecosystem is the actual point.
