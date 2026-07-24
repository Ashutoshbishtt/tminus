# ADR 0003: Separate services, but one repo and one binary

**Status:** Accepted
**Date:** 24 July 2026

## Why

The system has clear parts: taking orders, running dispatch, reading rider locations, sending
notifications, generating load. The question was how far to split them. Full microservices with
their own repos and databases, one single process, or something in between.

## Decision

The services run as separate processes, but they live in one repo, build into one binary, and
share one database with strict rules about who owns what.

**Separate when running.** `api`, `dispatcher`, `locator`, `notifier` and `simulator` each run
as their own process in their own container. They cannot call each other's functions. The only
way they talk is through Kafka and RabbitMQ.

**One repo, one binary.** One Go module. The binary takes a subcommand that decides what it
becomes, like `tminus dispatcher` or `tminus api`. In Docker Compose these are separate
containers running the same image with different subcommands.

**One database, but owned tables.** A single Postgres. Each service owns its own tables and no
service reads another service's tables directly. Anything crossing that line goes through an
event or an API, which is the same rule separate databases would force on me.

## Why this split and not another

I already build and run real microservices at work, with separate repos, separate databases and
service discovery. So that plumbing has little left to teach me. It is also where most side
projects die, because every small change touches three repos and the weekends go to config
instead of to the actual problem.

What I do want to learn sits at the gaps between these processes. Messages arriving twice. A
worker dying halfway through a job. A queue backing up. Following one order across process
lines. Running them as separate processes that only talk over the brokers keeps every one of
those problems real. Nothing is a plain function call pretending to be a service.

Keeping them in one binary takes away the pain without taking away the lesson. One build, one
place for shared types, one version. If I rename a shared field, the compiler tells me every
service that breaks, instead of it blowing up at runtime because one service shipped and another
did not.

## What else I considered

**One single process.** Easiest by far. But then dispatch is just a function call from the order
service, and the queues, retries, duplicate messages and lag all disappear. Those are the whole
point, so no.

**Full microservices from day one.** Separate repos, a database each, service discovery, an API
gateway. This is the version that looks impressive and quietly dies half finished. It also
repeats what I already do at work, so I would learn very little per hour spent.

## The cost

- One shared database means I am trusting myself to follow the no-cross-reading rule, instead of
  it being enforced for me. If I break it, splitting later gets harder.
- It is not a "look, real microservices" showpiece. I am fine with that. I do not need the label
  for interviews when I have the real thing at work.

If a real reason to split the database or pull a service out shows up later, I will write a new
note that replaces this one and explains what forced it. That is a better story than starting
split just for the sake of it.
