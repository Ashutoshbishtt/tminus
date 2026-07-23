# tminus

**A quick-commerce delivery backend that tries to get every order out in under 10 minutes.**

> Status: just started. I am still deciding the tech stack. Nothing here is ready to use yet.

---

## The problem

For a 10-minute delivery app, the hard part is not the ordering API. Anyone can build that.
The hard part is **dispatch**: orders keep coming in, riders keep moving around the city, and
something has to keep deciding who picks up which order, right now, fast enough that the food
still gets there in time.

And that decision has to be correct. If two riders get the same order, you pay twice. If no
rider gets it, you lose the customer. This has to keep working even when a rider goes offline
in the middle of a trip, or a queue restarts, or a service falls behind.

That is what this project is about.

## What I am building

One city, one dark store. Small on purpose, so I can go deep instead of wide.

- **Orders** — place an order without creating duplicates on retry, hold the stock for a short
  time, move the order through a fixed set of states, and clean up properly if it gets cancelled.
- **Riders** — go online and offline, send location updates constantly, get offered an order,
  accept it or let it time out.
- **Dispatch engine** — find nearby riders, score them, pick one, and make sure an order can
  never end up with two riders. If something goes wrong, hand the order to someone else.
- **Live tracking** — the customer watches the rider move on a map in real time.
- **Notifications** — send updates over different channels, retry the ones that fail, and park
  the ones that keep failing.

Two more pieces, and these are the ones I actually care about most:

- **A simulator** — spawns thousands of fake riders driving around and creates fake order
  traffic (normal load, lunch rush, dinner rush, sudden spikes). Same seed gives the same run,
  so I can compare results properly.
- **A test harness that breaks things on purpose** — kills a broker, kills a worker halfway
  through, adds network delay, slows down the database. Then checks whether the system still
  behaved correctly.

## The rules the system must never break

This is the whole point. The harness exists to check these:

1. **One order never goes to two riders.** Not even if a message gets delivered twice or a
   service restarts at the wrong moment.
2. **No order quietly disappears.** Every order ends up either delivered or cancelled.
3. **Stock is never oversold.** Two people cannot both buy the last item.
4. **Nothing happens twice** when it should only happen once.

## What I am not building

No real payment gateway, no reviews, no coupons, no chat, no admin panel, no mobile app, no
machine learning. If it does not put pressure on dispatch, messaging, or correctness, it stays out.

## Plan

**First version** — catalog and stock, order flow, rider flow, dispatch engine, live tracking,
notifications, the simulator, the test harness, one command to run everything locally, and
basic dashboards.

**Later** — grouping multiple orders into one trip, trying different dispatch strategies and
publishing which one wins, payments (fake provider, just to learn the rollback pattern), an ops
dashboard, tracing, and Kubernetes.

## How it fits together

```
                        ┌──────────────┐
  Customer / Rider ───► │ API Gateway  │  REST + WebSocket
                        └──────┬───────┘
                               │
      ┌────────────┬───────────┼────────────┬───────────────┐
      ▼            ▼           ▼            ▼               ▼
  ┌────────┐  ┌─────────┐ ┌─────────┐ ┌──────────┐  ┌──────────────┐
  │ Order  │  │  Stock  │ │  Rider  │ │ Tracking │  │ Notification │
  └───┬────┘  └────┬────┘ └────┬────┘ └────▲─────┘  └──────▲───────┘
      │            │           │           │               │
      └──────┬─────┴───────────┘           │               │
             ▼                             │               │
      ┌──────────────┐                     │               │
      │  event log   │  rider locations    │               │
      │              │  order events ─────►│               │
      └──────┬───────┘                     │               │
             ▼                             │               │
      ┌──────────────┐   dispatch jobs ┌───────────┐       │
      │   DISPATCH   │◄───────────────►│ job queue │───────┘
      │    ENGINE    │   offers, retry └───────────┘
      └──────────────┘

  Simulator ──drives──► everything ──checked by──► test harness
```

I have left the boxes generic on purpose. The real choices (which streaming platform, which
queue, which database, which language) are being decided now, and each one gets written down
in [`docs/adr/`](docs/adr/) with the reason and what it costs.

## Notes on decisions

Every real decision goes in [`docs/adr/`](docs/adr/): what I picked, what I did not pick, and
what I am giving up by picking it.

## License

MIT, see [LICENSE](LICENSE).
