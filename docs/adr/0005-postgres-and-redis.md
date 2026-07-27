# ADR 0005: Postgres for the truth, Redis for the fast stuff

**Status:** Accepted
**Date:** 27 July 2026

## Why

I had to decide where data lives. The obvious question was "lots of users, should I not be using
NoSQL for scale?" So I had to be honest about what actually decides this.

## Decision

**Postgres holds the truth.** Orders, order state, riders, stock, the outbox. Anything that has
to survive a crash and has to be right. The two hard rules of this project are enforced by
Postgres itself, inside transactions:

- one order can only ever go to one rider, using a uniqueness rule the database enforces,
- the last item in stock cannot be sold twice, because the hold happens in a transaction.

**Redis holds the fast, always-changing, throwaway stuff:**

- the live "who is near the store right now" list, using Redis' built in geo commands,
- a cache for the catalog and for "can we deliver to this address" answers,
- short stock holds, using Redis' built in expiry.

The flood of rider locations comes in from Kafka and updates the Redis list. It never touches
Postgres.

## Why not NoSQL, with lots of users

Because scale is not what decides this, and the question is half myth.

**SQL scales fine.** Instagram, Notion, GitHub and Shopify all run their core data on Postgres at
huge size. You hit plenty of other limits first.

**Users are not writes per second.** Most users are asleep or just browsing. This project is one
city and one dark store, so even at a busy peak it is a few writes a second. Postgres does tens
of thousands a second on a laptop. I am nowhere near its limit and never will be here.

**The real question is the shape of the data.** My order data is deeply connected and has to be
exactly right. Most NoSQL databases give up transactions and strong consistency to get their
scaling. That means I would have to rebuild correctness by hand in my own code, and beat decades
of database work while doing it. I would be giving up the one thing I need, consistency, to buy
the one thing I do not need here, huge write volume.

And I am not avoiding NoSQL out of habit. I am using it where it fits. The rider location flood
goes to Redis and Kafka, both not SQL, exactly because SQL is the wrong tool for a fast stream of
throwaway data. It was never SQL against NoSQL. It is the right store for each kind of data.

## One thing I am not doing

I am not using a Redis lock to enforce "one order, one rider." A Redis lock can quietly let go at
the wrong moment and leave two workers both thinking they hold it. For a rule where breaking it
costs real money, the safe referee is a Postgres transaction. Redis is there for speed, not for
correctness.

## What else I considered for finding nearby riders

- **Redis geo commands.** Picked. Built for exactly this, fast, survives a restart, simple.
- **Keeping the list in the service's memory and rebuilding it from Kafka on startup.** Faster,
  and a nice way to show why Kafka keeping history is useful. But more code, and it lives in one
  process. Saving this as a later upgrade instead of the starting point.
- **PostGIS, the geo add-on for Postgres.** Would point the location flood straight at my durable
  database. Wrong load in the wrong place.

## The cost

- Two datastores to run and keep straight instead of one.
- Some data sits in two places. A rider's real record is in Postgres, their live position is in
  Redis. That is fine as long as I treat Redis as throwaway and can always rebuild it from the
  Kafka stream.
