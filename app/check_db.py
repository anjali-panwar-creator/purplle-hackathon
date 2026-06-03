import sqlite3
conn=sqlite3.connect('store_intelligence.db')
c=conn.cursor()
c.execute("SELECT zone_id, dwell_ms FROM events WHERE event_type='ZONE_EXIT'")
print(c.fetchall())
