# ADR 0008: Talk to Postgres with pgx

**Status:** Accepted
**Date:** 5 August 2026

## Why

Day 3 is the first time the code opens a connection to anything. Whatever I pick here, every
query from now to day 44 is written against it.

Go gives a choice. There is `database/sql` in the standard library, a generic interface any
database can plug into. Or a driver that talks to one database directly and does not pretend to
be portable.

ADR 0005 already decided Postgres holds the truth and that the correctness rules live in
Postgres transactions. I am not going to swap it out. So the question is not "what if I change
database", it is "what do I lose by pretending I might".

## Decision

`github.com/jackc/pgx/v5`, used natively through `pgxpool`, not through `database/sql`.

Three things I get that I would otherwise not have:

**LISTEN/NOTIFY.** Postgres can tell a program that something changed, instead of the program
asking over and over. Day 13 builds the outbox relay, which reads unsent rows and publishes them
to Kafka. Without LISTEN/NOTIFY that loop polls on a timer, and the timer is a trade between
wasted queries and added latency.

**COPY.** Bulk loading. The simulator on day 36 creates thousands of orders, and seeding by
looping over INSERT is slow enough to change how I work.

**Real types.** jsonb, uuid, timestamptz, arrays and intervals arrive as themselves rather than
being pushed through strings and parsed back. Event payloads on day 12 are jsonb. Every id is a
uuid.

`pgxpool` also has the settings that matter here — minimum connections, maximum lifetime, idle
time — rather than `database/sql`'s thinner set.

I measured the difference before writing this. A single `pgx.Conn` under five concurrent queries
completes one and rejects four with `conn busy`; a pool completes all five in the same wall
time. Notes in `learning/notes/connection-pool.md`.

## What else I considered

- **`database/sql` with pgx underneath.** The middle road, and tempting: the standard interface,
  every tutorial applies, pgx still doing the talking. But it gives up all three things above and
  buys portability I have already decided I do not want. I can still get a `database/sql` handle
  out of the pool when a library needs one, so I am not locked out of that world either.

- **`lib/pq`.** The old default. In maintenance mode for years, strictly worse than pgx as a
  driver, nothing to offer in return. Rejected on the record so I do not reconsider it in six
  months.

- **An ORM.** ADR 0002 said nothing hidden behind a framework, and the point of days 9, 28 and 39
  is knowing exactly what SQL runs and in which transaction. An ORM puts a layer between me and
  the thing I am trying to learn.

## The cost

**The code is Postgres-shaped, not just the schema.** Swapping database later means rewriting
every query, not changing a connection string. That cost is already paid — ADR 0005 tied
correctness to Postgres transactions, so the schema was never portable. This makes the dependency
honest instead of pretending.

**Two APIs, eventually.** Anything that wants `database/sql` — a migration tool, most likely —
needs a bridge over the same pool. One codebase, two ways of talking to Postgres.

**The first dependency in the repo.** The zero-dependency run ends here. It buys enough to be
worth it.
