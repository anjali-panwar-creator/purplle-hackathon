# CHOICES.md

This document outlines the three primary architectural decisions made while building the Store Intelligence Pipeline, and the AI-assisted reasoning behind them.

## 1. Detection Model Choice: YOLOv8 Native Tracking vs Standalone ByteTrack
**Options Considered:**
1. Standalone `bytetrack` + Custom Object Detector (PyTorch).
2. Ultralytics YOLOv8 with built-in `model.track()`.

**What AI Suggested:**
The AI suggested using the standalone `bytetrack` library initially, but during the implementation phase, it recognized severe dependency conflicts on Windows involving C++ compilation and Torch extensions. It then recommended pivoting to the Ultralytics built-in tracker.

**What We Chose and Why:**
We chose **Ultralytics YOLOv8 Built-in Tracking (`model.track(persist=True)`)**.
We chose this because it natively incorporates BoT-SORT/ByteTrack without the massive overhead of compiling custom C++ extensions on edge devices. Retail edge environments (like a physical Purplle store backroom) often lack full DevOps support, so a Python library that reliably installs via `pip` and handles both detection and tracking in a single pass is highly preferable to a fragile custom C++ pipeline.

## 2. Event Schema Design Rationale
**Options Considered:**
1. **State-based polling:** Continually writing a visitor's `(x, y)` coordinate to the database every 100ms.
2. **Event-driven architecture:** Only writing to the database when a state change occurs (e.g., `ZONE_ENTER`, `ZONE_EXIT`).

**What AI Suggested:**
The AI strongly advocated for an Event-Driven Architecture, warning that writing `(x,y)` coordinates for 50 shoppers at 15 FPS would result in 750 database writes per second, crashing a lightweight SQLite edge database. 

**What We Chose and Why:**
We chose the **Event-Driven Architecture** and designed the schema to only emit state transitions. The Python pipeline maintains an internal state machine (tracking `enter_time` and `zone_id`), and only emits a `ZONE_EXIT` event containing the aggregated `dwell_ms` when the visitor physically leaves the zone polygon. This reduces network overhead by 99% and allows the Golang API to instantly compute average dwell times without complex time-series joins.

## 3. API Architecture Choice: Golang + SQLite
**Options Considered:**
1. FastAPI (Python) + PostgreSQL.
2. Gin (Golang) + SQLite (Pure Go).

**What AI Suggested:**
The AI suggested Golang + SQLite because the physical retail environment acts as an "Edge Node" where cloud databases may become unreachable during internet outages.

**What We Chose and Why:**
We chose **Golang + SQLite using the `glebarez/sqlite` pure-Go driver**.
A retail store requires a lightweight, zero-configuration database that doesn't need a dedicated DBA to manage. SQLite is perfect for this. We chose Golang because it can handle thousands of concurrent ingestion requests via Goroutines effortlessly. Crucially, we chose a *pure-Go* SQLite driver to completely eliminate the need for CGO and GCC compilers, ensuring that our `docker-compose up` command runs flawlessly on any OS without missing C-dependencies.
