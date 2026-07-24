# ADR 0003: Separate services, but one repo and one binary

**Status:** Accepted
**Date:** 24 July 2026

## Why this came up

The system has clear parts: taking orders, running dispatch, reading the rider location
stream, sending notifications, generating load. The question was how far to split them.
Fully separate microservices with their own repos and databases, or one process, or
something in between.

## Decision

The services run as separate processes, but they live in one repo, build into one binary,
and share one database with strict ownership rules.

- **Separate at runtime.** `api`, `dispatcher`, `locator`, `notifier` and `simulator` each
  run as their own process in their own container. They cannot call each other's functions.
  The only way they talk is through Kafka and RabbitMQ.
- **One repo, one binary.** One Go module. The binary takes a subcommand that decides which
  service it becomes (`tminus dispatcher`, `tminus api`, and so on). In Docker Compose these
  are separate containers running the same image with different subcommands.
- **One database, owned tables.** A single Postgres, but each service owns its own tables and
  no service reads another service's tables directly. Cross-service data goes through events
  or an API, the same rule as separate databases would enforce.

## Why this split and not another

I already build and run true microservices at work, with separate repos, separate databases
and service discovery. So that plumbing has little left to teach me, and it is where most
side projects stall: every small change touches three repos, and the weekends go to config
instead of to the actual problem.

What I do want to learn lives at the boundaries between these processes: duplicate messages,
a consumer dying halfway through a job, queues backing up, tracing one order across process
lines. Running the services as separate processes that only talk over the brokers keeps every
one of those problems real. Nothing is a plain function call pretending to be a service.

Keeping them in one binary removes the pain without removing the lesson. One build, one place
for shared types, one version. Rename a shared field and the compiler lists every service that
breaks, instead of it blowing up at runtime because one service shipped and another did not.

## What I looked at instead

- **One single process.** Easiest by far, but then dispatch is just a function call from the
  order service, and the queues, retries, duplicate messages and consumer lag all vanish. That
  is the whole point of the project, so no.
- **Full microservices from day one.** Separate repos, a database each, service discovery, an
  API gateway. This is the version that looks impressive and quietly dies half built. It also
  repeats what I already do at work, so the learning per hour is low.

## What this costs me

- One shared database means I am trusting myself to honour the no-cross-reading rule rather
  than having it enforced by separate connections. If I break the rule, splitting later gets
  harder.
- It is not a "look, real microservices" showpiece. That is fine. I do not need that label for
  interviews, because I have the real thing at work.

If a real reason to split the database or pull a service into its own repo shows up later, I
will write a new note that replaces this one and explains what forced it. That is a better
story than starting split for the sake of it.
