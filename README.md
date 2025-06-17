# Distributed Systems Simulation

This project simulates information dissemination in distributed systems using different communication protocols: **singlecast**, **multicast**, **broadcast**, and **gossip** (with variants like push, pull, and push-pull).  

It models real-world conditions such as **packet loss**, **node failures**, and **network delays**, and collects performance metrics like **time to full propagation** and **coverage percentage** over time.

## Features

- Simulation of 100+ nodes as lightweight goroutines.
- Simulates network delays, message loss, and message corruption
- Supports configurable node counts, domains, fan-out, and more
- Automatically collects metrics and writes to a database
- Outputs CSV logs per node
- Memory usage statistics included
- Full analysis pipeline per simulation type

## Technologies

- **Golang** for simulation engine.
- **Channels** and **goroutines** for concurrency.
- **CSV logging** for easy post-processing and visualization.

## Usage

Run the simulation with custom flags:

```bash
go run main.go \
  -id 1 \
  -nodes 100 \
  -domains 3 \
  -fanout 2 \
  -delay 20 \
  -alive 0.9 \
  -loss 0.05 \
  -corrupt 0.05 \
  -timer 30
```

## Flags

Flag Description
-**id** Experiment ID
-**timer** Max time (sec) allowed for each simulation
-**nodes** Number of nodes in the network
-**domains** Multicast domain count
-**fanout** Gossip fan-out (neighbors per round)
-**delay** Mean delay in message delivery (ms)
-**alive** Probability a node is operational
-**loss** Probability of message loss
-**corrupt** Probability of message corruption
-**remove-db** Delete previous experiment data from DB
-**verbose** Show detailed logs

## Simulations

Each simulation is prepared with:

- Node initialization
- Alive-state randomization
- CSV log creation per node

Then one of the following is run:

- Broadcast — One-to-all send
- Singlecast — One-to-one targeted send
- Multicast — Domain-based multiple group sends
- Gossip Push/Pull/PushPull — Epidemic-style message spread

[Report](/report/report.md)

## Output

- CSV logs per node saved to /metrics
- Aggregated metrics saved to SQLite DB
- Output logs saved per experiment run

```mermaid
graph TD
    A[Start Simulation] --> B[Parse Flags & Init Params]
    B --> C[Set Alive Nodes Mask]
    C --> D[Run Broadcast]
    C --> E[Run Singlecast]
    C --> F[Run Multicast]
    C --> G[Run Gossip Push]
    C --> H[Run Gossip Pull]
    C --> I[Run Gossip PushPull]
    D --> Z[Wait / Timer Check]
    E --> Z
    F --> Z
    G --> Z
    H --> Z
    I --> Z
    Z --> K[Write to DB & Analyze]
    K --> L[Print Memory Stats]
    L --> M[End]
```

---

## Algorithms

```mermaid
flowchart
  subgraph Broadcast
    A1(Sender) --> B1[Node 1]
    A1 --> B2[Node 2]
    A1 --> B3[Node 3]
    A1 --> B4[Node 4]
  end
```

```mermaid
flowchart
  subgraph Multicast
    A2(Sender 0) --> C1[Sender 1]
    A2 --> C2[Node 2]
    A2 --> C3[Node 3]

    C1 --> C4[Node 4]
    C1 --> C5[Node 5]
    %% Multicast targets only a subset
  end
```

```mermaid
flowchart 
  subgraph Singlecast
    A3(Sender) --> D1[Node 1] --> D2[Node 2] --> D3[...]
  end
```

```mermaid
graph
  subgraph GossipPush
    E1(Sender) --> E2[Node A]
    E1 --> E3[Node B]
    E2 --> E4[Node C]
    E3 --> E5[Node D]
    E2 --> E6[Node E]
    E3 --> E7[Node F]
  end
```

```mermaid
flowchart
  subgraph GossipPull
    F2[Node A] --> F1(Reciver)
    F3[Node B] --> F1
    F4[Node C] --> F3
  end
```

```mermaid
flowchart
  subgraph GossipPushPull
    G1(Sender) <--> G2[Node A]
    G2 <--> G3[Node B]
    G4[Node C] <--> G2
    G3 <--> G4
  end
```
