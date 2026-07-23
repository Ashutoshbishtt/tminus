# ADR 0001: Write down the decisions

**Status:** Accepted
**Date:** 23 July 2026

## Why

I am building this project to actually learn the hard parts, not just to have something that
runs. The thinking behind a choice matters as much as the code. If I do not write the reasons
down, then in three months I will not remember what I chose on purpose and what I just did
without thinking.

## Decision

Every real decision gets a short file in `docs/adr/`, numbered in order. Each file says:
what the situation was, what I picked, what else I looked at, and what it costs me.

I do not go back and edit an old decision. If I change my mind later, I write a new file and
mark the old one as replaced by it. That way the history stays honest.

## What else I considered

- **Not writing anything down.** Easiest option, and it throws away the part of this project
  that is worth the most.
- **One big design doc.** Simpler to keep, but it only shows where I ended up. It hides the
  paths I did not take, which are usually the interesting bit.

## The cost

A bit of extra writing every time I decide something. In return the repo explains itself,
and anyone reading it can see why it looks the way it does.
