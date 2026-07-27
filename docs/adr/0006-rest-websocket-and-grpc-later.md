# ADR 0006: REST for the APIs, WebSocket for live updates, gRPC later

**Status:** Accepted
**Date:** 27 July 2026

## Why

Three questions to answer here. How do the apps talk to my API? How do live updates reach a
phone? And how do my own services talk to each other?

## Decision

**Apps to my API: REST with JSON, using `chi` as the router.**

Nothing fancy needed. Every client knows REST, I can test it with curl, and I already use chi at
work. Keeping the boring part boring means I can spend my time on dispatch, which is the part I
actually want to think about.

**Live updates: WebSocket**, for the customer watching the map and for the rider's connection.

**Between my own services: mostly nothing.**

They talk through Kafka and RabbitMQ. That was the whole reason for splitting them that way. For
the few spots where one service has to ask another something and wait for the answer, plain REST
is fine for now.

**gRPC: saving it for v2.**

I will build the REST version first, then move one internal call over to gRPC and write down
what changed. Moving something that already works teaches me more than starting with it. It also
makes a better story than "I used gRPC because it is fast."

## Why WebSocket, for real this time

My first reason for picking WebSocket was wrong, so I want the honest one written down.

I said the rider side needs two-way talk because the rider gets an offer and sends back an
accept. But accepting is just a normal endpoint. `POST /offers/501/accept` works fine over REST.
That was never the problem.

The real problem comes one step earlier: **how does the offer even reach the rider?**

With plain HTTP the server cannot start the conversation. The phone has to ask first. So the
rider's app would sit there asking "anything for me? anything for me?" over and over. That hurts
here because an offer only lasts 15 seconds. If the app checks every 5 seconds, the rider loses
a third of the window just finding out an offer existed. I need the server to push the offer
down the second it exists.

If I split the rider's app into its three jobs:

| What it does | Which way | Needs a push? |
| --- | --- | --- |
| send location every few seconds | phone to server | no |
| **get an offer** | **server to phone** | **yes** |
| accept the offer | phone to server | no |

Only one of the three needs a push. So SSE for the offers (a one-way stream from server to
phone) plus plain REST for the rest is a working design, and I would not call it wrong.

I went with WebSocket anyway, for two reasons:

1. **Location updates get costly over plain HTTP.** Every request carries headers, an auth token
   and connection setup. That is a few hundred bytes of wrapper around a tiny "I am here"
   message. Every few seconds, times thousands of riders, it adds up fast. On a WebSocket the
   connection is already open, so each ping is just the data.
2. **One connection instead of two things to look after.** With SSE plus REST the app keeps a
   stream open *and* keeps firing separate requests. With WebSocket there is one pipe doing all
   three jobs. One thing to connect, one thing to reconnect when the network drops, one thing to
   debug.

There is also a reason specific to this project. My "rider app" is really the simulator, one Go
program pretending to be thousands of riders. Giving each fake rider one open connection is much
lighter on my laptop than each one firing off HTTP requests on a timer.

## What else I looked at

- **Polling.** Simplest, but wasteful, and too slow for a 15-second offer.
- **SSE plus REST.** Honestly fine, and simpler than WebSocket for the customer side, since the
  map only needs data going one way. I skipped it because it means running two different
  real-time setups instead of one, plus the location cost above.
- **gRPC now.** Faster, and the contracts are typed, but it needs code generation and setup for
  the handful of internal calls I have. Better as a v2 move.

## The cost

- WebSocket is more work than SSE. I have to deal with reconnects, heartbeats and dropped
  connections myself.
- Worth being honest: real delivery apps mostly send offers as push notifications through
  Firebase or APNs, because a rider's phone app might be closed or in the background, and a
  WebSocket dies when that happens. My riders are simulated and always running, so WebSocket
  works here. It would not be the right answer for a real phone.
