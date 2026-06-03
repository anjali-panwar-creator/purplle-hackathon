import cv2
import json
import logging
import requests
import uuid
import time
import argparse
from datetime import datetime
from ultralytics import YOLO

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("CV-Pipeline")

class StoreIntelligencePipeline:
    def __init__(self, layout_path, api_url):
        self.api_url = api_url
        with open(layout_path, 'r') as f:
            self.layout = json.load(f)
        
        self.zones = self.layout['zones']
        self.store_id = self.layout['store_id']
        
        # Initialize YOLOv8 model for person detection & tracking
        logger.info("Loading YOLOv8 model...")
        self.model = YOLO("yolov8n.pt") # n is for nano (fastest). Can use 's' or 'm' for better accuracy.
        
        # State tracking: visitor_id -> { zone_id: string, enter_time: float, is_staff: bool, last_seen: float }
        self.visitor_states = {}
        # Cache for Re-ID temporal heuristic
        self.recently_exited = {} 
        self.event_batch = []

    def point_in_polygon(self, x, y, poly):
        n = len(poly)
        inside = False
        p1x, p1y = poly[0]
        for i in range(1, n + 1):
            p2x, p2y = poly[i % n]
            if y > min(p1y, p2y):
                if y <= max(p1y, p2y):
                    if x <= max(p1x, p2x):
                        if p1y != p2y:
                            xinters = (y - p1y) * (p2x - p1x) / (p2y - p1y) + p1x
                        if p1x == p2x or x <= xinters:
                            inside = not inside
            p1x, p1y = p2x, p2y
        return inside

    def get_zone_for_point(self, x, y):
        for zone in self.zones:
            if 'polygon' in zone and self.point_in_polygon(x, y, zone['polygon']):
                return zone['zone_id']
        return None

    def emit_event(self, camera_id, visitor_id, event_type, zone_id=None, dwell_ms=0, queue_depth=None, is_staff=False):
        event = {
            "event_id": str(uuid.uuid4()),
            "store_id": self.store_id,
            "camera_id": camera_id,
            "visitor_id": f"VIS_{visitor_id}",
            "event_type": event_type,
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "zone_id": zone_id,
            "dwell_ms": dwell_ms,
            "is_staff": is_staff,
            "confidence": 0.95,
            "metadata": {
                "queue_depth": queue_depth,
                "session_seq": 1
            }
        }
        self.event_batch.append(event)
        
        if len(self.event_batch) >= 5:
            self.flush_events()

    def flush_events(self):
        if not self.event_batch:
            return
        
        try:
            response = requests.post(f"{self.api_url}/events/ingest", json=self.event_batch)
            if response.status_code == 200:
                logger.info(f"Successfully ingested {len(self.event_batch)} events.")
            else:
                logger.error(f"Failed to ingest events: {response.text}")
        except Exception as e:
            logger.error(f"API unreachable: {e}")
            
        self.event_batch = []

    def process_video(self, video_path, camera_id):
        logger.info(f"Processing video {video_path} for camera {camera_id}...")
        
        # We use Ultralytics built-in tracking (persist=True ensures IDs are tracked across frames)
        # tracker="botsort.yaml" or "bytetrack.yaml" can be used. By default it uses botsort.
        results = self.model.track(source=video_path, show=False, stream=True, classes=[0]) # class 0 is Person
        
        for frame_idx, result in enumerate(results):
            if result.boxes is None or result.boxes.id is None:
                continue
            
            # Extract tracking IDs and bounding boxes
            track_ids = result.boxes.id.int().cpu().tolist()
            boxes = result.boxes.xyxy.cpu().tolist()

            current_frame_visitors = set()

            for track_id, box in zip(track_ids, boxes):
                x1, y1, x2, y2 = box
                current_frame_visitors.add(track_id)
                
                # Use bottom-center of bounding box (the feet)
                feet_x = (x1 + x2) / 2
                feet_y = y2
                
                current_zone = self.get_zone_for_point(feet_x, feet_y)
                
                if track_id not in self.visitor_states:
                    # RE-ID Temporal Heuristic: Check if someone left recently
                    reid_match = None
                    now = time.time()
                    for old_id, exit_time in list(self.recently_exited.items()):
                        if now - exit_time < 30: # 30 second re-entry window
                            reid_match = old_id
                            break
                        else:
                            del self.recently_exited[old_id]
                    
                    if reid_match:
                        actual_id = reid_match
                        del self.recently_exited[reid_match]
                        self.emit_event(camera_id, actual_id, "REENTRY")
                    else:
                        actual_id = track_id
                        self.emit_event(camera_id, actual_id, "ENTRY")

                    self.visitor_states[actual_id] = {
                        "zone_id": current_zone,
                        "enter_time": time.time(),
                        "is_staff": False,
                        "last_seen": time.time()
                    }
                    
                    if current_zone:
                        self.emit_event(camera_id, actual_id, "ZONE_ENTER", zone_id=current_zone)
                        if current_zone == "BILLING_ZONE":
                            self.emit_event(camera_id, actual_id, "BILLING_QUEUE_JOIN", zone_id=current_zone, queue_depth=1)
                else:
                    prev_state = self.visitor_states[track_id]
                    now = time.time()
                    
                    # Staff Exclusion Logic: If in BILLING_ZONE for > 10 mins (600s), classify as staff
                    if prev_state["zone_id"] == "BILLING_ZONE" and (now - prev_state["enter_time"]) > 600:
                        prev_state["is_staff"] = True

                    if prev_state["zone_id"] != current_zone:
                        dwell_ms = int((now - prev_state["enter_time"]) * 1000)
                        
                        if prev_state["zone_id"]:
                            self.emit_event(camera_id, track_id, "ZONE_EXIT", zone_id=prev_state["zone_id"], dwell_ms=dwell_ms, is_staff=prev_state["is_staff"])
                        
                        if current_zone:
                            self.emit_event(camera_id, track_id, "ZONE_ENTER", zone_id=current_zone, is_staff=prev_state["is_staff"])
                            if current_zone == "BILLING_ZONE" and not prev_state["is_staff"]:
                                self.emit_event(camera_id, track_id, "BILLING_QUEUE_JOIN", zone_id=current_zone, queue_depth=1)
                                
                        self.visitor_states[track_id]["zone_id"] = current_zone
                        self.visitor_states[track_id]["enter_time"] = now
                        
                    self.visitor_states[track_id]["last_seen"] = now

            # Handle lost tracks (EXIT)
            now = time.time()
            for tid in list(self.visitor_states.keys()):
                if tid not in current_frame_visitors and (now - self.visitor_states[tid]["last_seen"] > 5):
                    # Person has been gone for 5 seconds
                    state = self.visitor_states[tid]
                    dwell_ms = int((now - state["enter_time"]) * 1000)
                    if state["zone_id"]:
                        self.emit_event(camera_id, tid, "ZONE_EXIT", zone_id=state["zone_id"], dwell_ms=dwell_ms, is_staff=state["is_staff"])
                    self.emit_event(camera_id, tid, "EXIT", is_staff=state["is_staff"])
                    self.recently_exited[tid] = now
                    del self.visitor_states[tid]

            if frame_idx % 100 == 0:
                logger.info(f"Processed {frame_idx} frames...")
                
        # The video has ended, or they walked out of frame. 
        # We must emit EXIT events for everyone remaining in state so they aren't permanently "Active".
        for track_id, state in self.visitor_states.items():
            now = time.time()
            dwell_ms = int((now - state["enter_time"]) * 1000)
            if state["zone_id"]:
                self.emit_event(camera_id, track_id, "ZONE_EXIT", zone_id=state["zone_id"], dwell_ms=dwell_ms)
            self.emit_event(camera_id, track_id, "EXIT")
            
        self.visitor_states = {}

        # Flush any remaining events
        self.flush_events()
        logger.info("Finished processing video.")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Purplle Store Intelligence Detection Pipeline")
    parser.add_argument("--video", type=str, required=True, help="Path to the CCTV .mp4 footage")
    parser.add_argument("--camera", type=str, required=True, help="Camera ID (e.g., CAM_ENTRY_01)")
    parser.add_argument("--layout", type=str, default="../DATA/store_layout.json", help="Path to store_layout.json")
    parser.add_argument("--api", type=str, default="http://localhost:8080", help="API URL")
    
    args = parser.parse_args()

    pipeline = StoreIntelligencePipeline(layout_path=args.layout, api_url=args.api)
    
    # Process the provided video
    pipeline.process_video(args.video, args.camera)
