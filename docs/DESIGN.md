# DESIGN.md

## Architecture Overview
The Purplle Store Intelligence System is an end-to-end edge analytics pipeline designed to operate inside physical retail locations. It converts unstructured CCTV video feeds into structured, actionable business intelligence.

The architecture is divided into three highly decoupled layers:

### 1. The Detection Layer (Python, Ultralytics)
The "Eyes" of the system. We use a Python pipeline with Ultralytics YOLOv8 for Object Detection and built-in tracking (BoT-SORT).
- **Spatial Mapping:** A mathematical Ray-Casting algorithm continuously evaluates the `(x, y)` "feet" coordinates of a visitor's bounding box against the geometric zone polygons defined in `store_layout.json`.
- **State Machine:** To prevent network flooding, the pipeline uses an internal state machine to track visitor dwell times locally. It only emits `ZONE_EXIT` HTTP POST requests when a visitor physically leaves a zone.
- **Time-Window Re-ID Heuristic:** To handle Cross-Camera deduplication without a heavy Re-ID neural network, we implement a temporal cache. If a tracking ID disappears from a camera exit threshold and a new ID appears at an entry threshold within 30 seconds, the IDs are merged, and a `REENTRY` event is emitted.

### 2. The Intelligence API (Golang, Gin, SQLite)
The "Brain" of the system. 
- **High Throughput:** Built in Golang to leverage lightweight Goroutines, allowing the server to asynchronously process thousands of concurrent event ingestions from dozens of cameras.
- **Edge Durability:** Uses a serverless, Pure-Go SQLite database (`glebarez/sqlite`) designed for edge node environments where cloud connectivity is unreliable.
- **Idempotency:** The `/events/ingest` endpoint is natively idempotent via SQLite `ON CONFLICT DO NOTHING` clauses keyed on UUIDs, protecting against duplicate network retries.
- **On-the-fly Computation:** Endpoints like `/funnel` and `/heatmap` compute analytics on-the-fly using SQL aggregations (`COUNT(DISTINCT visitor_id)`), ensuring metrics are perfectly real-time and never cached from stale data.

### 3. The Command Center (React, Vite)
A real-time visual dashboard (Part E Bonus) built with Vite and React. It polls the Golang API and visualizes queue depths, active anomalies, and conversion funnels using a premium Glassmorphism Dark Mode aesthetic.

---

## AI-Assisted Decisions
This system was pair-programmed with an AI Assistant (Gemini). Below are three critical places where the LLM shaped the design:

### 1. The Pure-Go SQLite Pivot (Agreed & Adopted)
**The Scenario:** Initially, we set up `gorm.io/driver/sqlite`. When running the server on a local Windows machine, it completely failed due to missing `CGO` and `GCC` compilers required by the standard SQLite C-bindings.
**The AI Input:** The AI diagnosed the CGO compiler issue and suggested completely overriding the standard driver with `github.com/glebarez/sqlite`, a CGO-free, pure-Go port of SQLite.
**The Verdict:** We strongly agreed. By adopting the pure-Go driver, we eliminated all cross-compilation errors, ensuring the code could be instantly run or containerized on any reviewer's machine without installing build-essential tools.

### 2. The Bounding Box "Feet" Logic (Agreed & Adopted)
**The Scenario:** When mapping a YOLO bounding box `[x1, y1, x2, y2]` to a physical floor zone, we initially considered checking if the center of the box was inside the polygon. 
**The AI Input:** The AI pointed out a physical edge case: if a customer leans over a Skincare display to test a product, the center of their bounding box (their torso/head) will register in the Skincare zone, but their feet are in the aisle. The AI suggested extracting the bottom-center coordinate `feet_x = (x1 + x2)/2, feet_y = y2` for ray-casting.
**The Verdict:** We agreed. Using the feet coordinates drastically improved the physical accuracy of the spatial mapping.

### 3. Event Batching Thresholds (Overrode)
**The Scenario:** The AI initially designed the Python detection pipeline to batch 50 events before flushing the HTTP POST request to the API to save network bandwidth.
**The AI Input:** The AI assumed a high-FPS, high-traffic scenario where 50 events would be generated every few seconds.
**The Verdict:** We **overrode** this decision during testing. Because we were running CPU inference (processing ~2 frames per second), it took extremely long to generate 50 events, causing the Dashboard to look frozen. We overrode the AI's logic and reduced the batch threshold to `5` for testing, enabling real-time visual feedback on the Dashboard.
