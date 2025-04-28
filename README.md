# Distributed Systems Simulation

This project simulates information dissemination in distributed systems using different communication protocols: **singlecast**, **multicast**, **broadcast**, and **gossip** (with variants like push, pull, and push-pull).  

It models real-world conditions such as **packet loss**, **node failures**, and **network delays**, and collects performance metrics like **time to full propagation** and **coverage percentage** over time.

## Features

- Simulation of 100+ nodes as lightweight goroutines.
- Support for different dissemination strategies:
  - Singlecast
  - Multicast
  - Broadcast
  - Gossip (Push, Pull, Push-Pull)
- Emulation of:
  - Random packet loss
  - Random node failures
  - Network latency
- Metric collection (CSV output) for performance analysis.
- Flexible configuration (number of nodes, failure rates, loss probability, etc.)

## Technologies

- **Golang** for simulation engine.
- **Channels** and **goroutines** for concurrency.
- **CSV logging** for easy post-processing and visualization.
