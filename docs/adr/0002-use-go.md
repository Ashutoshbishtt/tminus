# ADR 0002: Build tminus in Go

**Status:** Accepted
**Date:** 23 July 2026

## Why this came up

Before anything else I had to pick a language. The two real options were Go, which I write
every day at work, and Java, which I know but use mostly for design and DSA practice.

## Decision

Go.

## The reasons

**The workload fits.** Most of this system is thousands of connections that spend their time
waiting: riders sending location pings, customers watching a live map, workers sitting on a
queue. A goroutine costs about 2 KB to start, so one goroutine per connection is fine, and the
code still reads top to bottom instead of turning into callbacks. Ten thousand riders pinging
every five seconds is roughly two thousand messages a second plus a live socket per tracked
order. That is comfortable territory for Go.

**The simulator needs it.** The load simulator has to run thousands of fake riders, each one
moving and sending pings on its own clock. In Go that is a loop that starts thousands of
goroutines. That is the whole design.

**Deadlines run through everything.** A rider gets a few seconds to accept an offer, an order
has a ten minute clock, a database call should give up early. Go's `context` carries deadlines
through the standard library, the database driver and the queue clients, so one timeout can
cancel a whole chain of work. It is the normal way to write Go, not something I have to build.

**Short GC pauses.** I am going to publish p99 latency numbers, and Go's collector keeps pauses
well under a millisecond without tuning. The JVM can match this but tuning it is its own skill,
and I do not want to spend the project explaining a garbage collection spike.

**Nothing is hidden.** With Spring Boot I would annotate a method and messages would arrive,
and I would learn nothing about consumer groups or offset commits. In Go I write the consumer
loop myself and decide when to commit the offset. Feeling a message get processed twice because
I committed in the wrong place is the point of this project.

**It runs on my laptop.** A Go service starts in milliseconds and uses tens of megabytes. Kafka
and the databases will take most of the RAM, and I need the services to fit in what is left.

## What I looked at instead

**Java.** The honest argument for it is that I would learn more simply because I know it less,
and a lot of the product companies I might interview at run on it. I did not pick it because
what I actually want to learn here is not a language. Outbox, idempotency, consumer groups,
dead letter queues, delivery guarantees: those are the same ideas in any language. Choosing the
slower option does not teach me more about them, it just means I get to them later, and it
raises the chance I stall and never finish.

One thing I checked before writing this off: the old line about Go having lightweight threads
and Java not is out of date. Java has had virtual threads since Java 21 and they solve the same
problem. So that was not a reason to skip Java. The reasons above are.

## What this costs me

- Kafka's Java client is the reference one. Every feature appears there first and is documented
  best there. The Go clients are solid but they follow behind.
- No Kafka Streams or Flink, so real stream processing is off the table if I ever want it.
- Java's profiling tools are still ahead of Go's, though pprof is good enough.
- For heavy CPU work the JVM would be faster. This system is I/O bound, so it does not bite.

Java stays where it already earns its place: LLD practice, design patterns, and a separate
Spring Boot project later where the ecosystem is the actual point.
