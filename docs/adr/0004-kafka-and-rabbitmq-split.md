# ADR 0004: What goes on Kafka and what goes on RabbitMQ

**Status:** Accepted
**Date:** 27 July 2026

## Why

I want both Kafka and RabbitMQ in this project, and I want each one doing the job it is actually
good at. Not one of them bolted on so I can put it on a list. So I had to place every flow on
purpose and be able to say why it is there and not on the other one.

The rule I used to decide:

- **Kafka is a notice board.** You pin up things that happened. They stay pinned, in order. Many
  different readers can walk past and read the same notes at their own speed, and a new reader
  can show up later and read everything from the start.
- **RabbitMQ is a to-do list.** Each message is a job. One worker picks it up, does it, and it is
  gone. RabbitMQ gives me the useful bits around a single job: retry after a delay, priority, a
  timeout on the message, and a box to park the ones that keep failing.

## Decision

**Kafka carries the streams of facts.** Things that happened, that lots of parts of the app want
to read and keep.

- `rider.location` — every rider's location pings. Thousands a second. Read by tracking for the
  live map, by the part that keeps the "who is nearby" list fresh, and later by analytics. Kept
  in order per rider.
- `order.events` — the life story of each order: created, accepted, packed, ready, assigned,
  picked up, delivered. Read by dispatch, tracking, notifications and history. Kept in order per
  order, and kept around so a new reader can replay it.

**RabbitMQ carries the jobs.** Work one worker should do once, where I care about retry,
priority, timeouts, or parking the failures.

- `dispatch.jobs` — "find and lock a rider for this order." One worker per order. Orders close to
  breaking the ten minute promise jump the line using priority. Orders that cannot be placed go
  to the dead letter box instead of blocking everyone behind them.
- `rider.offers` — "offer this order to this rider, they have 15 seconds." The message carries
  its own timeout. If the rider does not accept in time, RabbitMQ sends it back to dispatch on
  its own, which means "try the next rider."
- `notifications` — "send this push, SMS or email." Sorted to the right worker by type, failed
  sends retried after a delay, hopeless ones parked.

## The two calls that could have gone either way

The obvious flows place themselves. These two are worth explaining.

**Why dispatch jobs are on RabbitMQ and not Kafka.** In Kafka, messages in one partition are read
strictly in order. So one bad order that keeps failing would block every order behind it until
someone deals with it. One weird order could freeze a whole slice of the city. RabbitMQ handles
each message on its own, so a bad order drops into the dead letter box and everyone else keeps
moving. Kafka also has no real per-message priority, and I need urgent orders to cut the line.

**Why order events are on Kafka and not RabbitMQ.** Order events have many readers and I want to
keep the history. On RabbitMQ a message is gone once it is consumed, and adding a new reader
later means redoing the plumbing. On Kafka a new reader, say analytics added next month, can show
up and read all of history from the start without touching anything that already works.

The pattern I like: each broker sits exactly where the other one is weak.

## One choice I made inside this

The 15 second offer timeout could have been a plain timer inside the dispatcher, with no broker
involved. That would be simpler. I picked the RabbitMQ way on purpose, because learning that
pattern, a timeout on a message that sends it back when it expires, is a big part of why I am
using RabbitMQ at all. If it causes more trouble than it teaches, I will write a new note and
move it in memory.

## The cost

- The order service cannot just save to the database and be done. To get an event onto Kafka
  without ever losing one, it writes the order and the event in the same database transaction,
  and a small relay reads that table and publishes to Kafka. That is more machinery, but it is
  the only way to close the gap between "saved to the database" and "told Kafka." It is what
  protects the rule that no order quietly disappears.
- Two brokers to run and understand instead of one. Worth it here, since learning both is the
  point.
