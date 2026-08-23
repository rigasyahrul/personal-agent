---
title: Hub startSession is not create-and-send atomic
date: 2026-08-20
tags:
  - hub
  - sessions
  - compounding
---

# Hub startSession is not create-and-send atomic

## Memory

After `createProjectSession` succeeds, always put the session in the list and
open it (`activeSession`) before / even if `sendMessage` fails. Never leave a
created session invisible so a second Send creates another. Keep the draft on
send fail so the open chat can retry.

See [[source/intro|Intro]] for the product framing, [[AGENTS]] for the standing
rule, and [[memory/lessons|lessons]] for the index.

## What went wrong

Treating create+send as one success path left the new session off the list.
The user retried and created a second orphan session.
