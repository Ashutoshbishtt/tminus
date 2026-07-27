# ADR 0007: The local stack and the versions I pinned

**Status:** Accepted
**Date:** 27 July 2026

## Why

Everything has to run on one laptop, and that laptop has 8 GB of RAM. So the question was not
just which pieces to run, but whether real Kafka fits next to Postgres, Redis, RabbitMQ and my
own services, or whether I need a lighter stand-in.

## Decision

One `docker-compose.yml` at the root, started with `make up`. Four containers, all pinned:

| What | Image | Why this one |
| --- | --- | --- |
| Postgres | `postgres:17-alpine` | 17 is mature and well proven. Alpine keeps it small. |
| Redis | `redis:7.4-alpine` | Has the geo commands I need. Stable and boring, which is what I want from a cache. |
| Kafka | `apache/kafka:4.0.0` | Kafka 4 runs in KRaft mode, so no ZooKeeper. One container instead of two. |
| RabbitMQ | `rabbitmq:4-management-alpine` | 4.x is current, and the management image gives me the UI on 15672 for free. |

Go is pinned at 1.24, which is what is already on the machine.

Everything is pinned to a real version, never `latest`. I want the stack to be the same next
month as it is today, and to upgrade on purpose rather than by surprise.

Other things I set on purpose:

- **Memory limits on every container**, and Kafka's JVM heap capped at 512 MB. Left alone the JVM
  will take a gigabyte just because it can.
- **Health checks everywhere**, and `make up` waits for them. Nothing reports ready before it is.
- **Three partitions by default** on new topics, so consumer groups and rebalancing are actually
  visible once I run more than one consumer.
- **Auto topic creation off.** I want a typo to fail loudly, not quietly make a topic.
- **24 hour retention.** Location data is a firehose and is only interesting for a short while.
- **Named volumes**, so a restart does not wipe the data, and `make reset` when I do want it gone.
- **The management UIs sit behind a profile** (`make tools`), because on 8 GB every container
  counts and those are only for looking at things.

## Does real Kafka actually fit

Yes, with room to spare. Measured on the machine, sitting idle:

| Container | Memory | Limit |
| --- | --- | --- |
| Kafka | 347 MB | 1200 MB |
| RabbitMQ | 106 MB | 512 MB |
| Postgres | 23 MB | 512 MB |
| Redis | 10 MB | 256 MB |
| **Total** | **~486 MB** | |

The whole stack comes up healthy in about 36 seconds. My own services are Go binaries at a few
tens of megabytes each, so they barely move the number.

## What else I considered

**Redpanda instead of Kafka.** It speaks the Kafka protocol, so my client code would be
identical, and it uses noticeably less memory because there is no JVM. That was a real option
given 8 GB. I measured real Kafka first, saw it sitting at 347 MB, and decided the memory saving
was not worth it. Learning Kafka is one of the reasons this project exists, and I would rather
hit the real thing's quirks than a compatible reimplementation's.

**Kafka with ZooKeeper.** That is the setup I would have had a couple of years ago. Kafka 4
removed ZooKeeper entirely, so KRaft is not just lighter, it is the only way forward.

**Confluent's images instead of the Apache ones.** More features, but heavier and further from
plain Kafka. I want the plain one.

**Running the databases straight on the laptop instead of in containers.** Faster, but then the
setup lives in my head instead of in a file, and nobody else can run it.

## The cost

- Four containers means about half a gigabyte gone before I write a line of code. Fine on this
  machine, and the limits stop any one of them running away.
- One Kafka node means no replication, so I cannot play with broker failure the way a three node
  cluster would let me. If I want that for the chaos work later, I can add nodes then.
- Pinned versions go stale. That is a trade I am happy with: I would rather upgrade deliberately
  than have a rebuild change something under me.
