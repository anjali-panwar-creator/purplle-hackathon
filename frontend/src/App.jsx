import React, { useState, useEffect } from 'react';
import { Activity, Users, ShoppingBag, Clock, AlertTriangle, ArrowRight } from 'lucide-react';
import './index.css';

const Dashboard = () => {
  const [metrics, setMetrics] = useState({
    unique_visitors: 0,
    current_visitors: 0,
    conversion_rate: 0,
    queue_depth: 0,
    avg_dwell_per_zone: {}
  });

  const [funnel, setFunnel] = useState({
    entry_count: 0,
    zone_visit: 0,
    billing_queue: 0,
    drop_off_pct: 0
  });

  const [anomalies, setAnomalies] = useState([]);
  
  // Max possible dwell time for chart normalization (e.g., 5000ms = 100%)
  const MAX_DWELL = 5000; 

  const fetchData = async () => {
    try {
      const [metricsRes, funnelRes, anomaliesRes] = await Promise.all([
        fetch('http://localhost:8080/stores/STORE_BLR_002/metrics'),
        fetch('http://localhost:8080/stores/STORE_BLR_002/funnel'),
        fetch('http://localhost:8080/stores/STORE_BLR_002/anomalies')
      ]);

      const mData = await metricsRes.json();
      const fData = await funnelRes.json();
      const aData = await anomaliesRes.json();

      setMetrics(mData);
      setFunnel(fData);
      setAnomalies(aData.active_anomalies || []);
    } catch (err) {
      console.error("Error fetching data:", err);
    }
  };

  useEffect(() => {
    fetchData(); // initial fetch
    const interval = setInterval(fetchData, 2000); // poll every 2 seconds
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="dashboard-container">
      <h1>Purplle Command Center</h1>

      <div className="grid-layout">
        
        {/* HERO METRICS */}
        <div className="glass-card metric-widget">
          <h2><Users size={20} color="var(--accent-purple)"/> Unique Visitors</h2>
          <div className="metric-value">{metrics.unique_visitors}</div>
          <div className="metric-label">Total Today</div>
        </div>

        <div className="glass-card metric-widget">
          <h2><Activity size={20} color="var(--success-green)"/> Active Visitors</h2>
          <div className="metric-value">{metrics.current_visitors}</div>
          <div className="metric-label">Currently in store</div>
        </div>

        <div className="glass-card metric-widget">
          <h2><ShoppingBag size={20} color="var(--accent-pink)"/> Conversion Rate</h2>
          <div className="metric-value">{metrics.conversion_rate.toFixed(1)}%</div>
          <div className="metric-label">POS Integrated</div>
        </div>

        <div className="glass-card metric-widget">
          <h2><Users size={20} color="#f59e0b"/> Queue Depth</h2>
          <div className="metric-value">{metrics.queue_depth}</div>
          <div className="metric-label">At Billing</div>
        </div>

        {/* FUNNEL AND DWELL CHARTS */}
        <div className="glass-card chart-container">
          <h2><Clock size={20}/> Average Dwell Time by Zone</h2>
          
          {Object.entries(metrics.avg_dwell_per_zone).length === 0 ? (
            <p style={{ color: 'var(--text-secondary)' }}>No zone data yet. Waiting for events...</p>
          ) : (
            Object.entries(metrics.avg_dwell_per_zone).map(([zone, ms]) => {
              const seconds = (ms / 1000).toFixed(1);
              const pct = Math.min((ms / MAX_DWELL) * 100, 100);
              return (
                <div className="bar-row" key={zone}>
                  <div className="bar-label">{zone}</div>
                  <div className="bar-track">
                    <div className="bar-fill" style={{ width: `${pct}%` }}></div>
                  </div>
                  <div className="bar-value">{seconds}s</div>
                </div>
              )
            })
          )}

          <h2 style={{ marginTop: '2rem' }}><ArrowRight size={20}/> Conversion Funnel</h2>
          <div className="bar-row">
            <div className="bar-label">Entered Store</div>
            <div className="bar-track">
              <div className="bar-fill" style={{ width: '100%', background: 'rgba(255,255,255,0.2)' }}></div>
            </div>
            <div className="bar-value">{funnel.entry_count}</div>
          </div>
          <div className="bar-row">
            <div className="bar-label">Visited Zone</div>
            <div className="bar-track">
              <div className="bar-fill" style={{ width: funnel.entry_count ? `${(funnel.zone_visit/funnel.entry_count)*100}%` : '0%' }}></div>
            </div>
            <div className="bar-value">{funnel.zone_visit}</div>
          </div>
          <div className="bar-row">
            <div className="bar-label">Billing Queue</div>
            <div className="bar-track">
              <div className="bar-fill" style={{ width: funnel.entry_count ? `${(funnel.billing_queue/funnel.entry_count)*100}%` : '0%' }}></div>
            </div>
            <div className="bar-value">{funnel.billing_queue}</div>
          </div>
        </div>

        {/* ALERTS FEED */}
        <div className="glass-card alerts-container">
          <h2><AlertTriangle size={20} color="var(--alert-red)"/> Active Anomalies</h2>
          {anomalies.length === 0 ? (
            <p style={{ color: 'var(--text-secondary)' }}>Store operating normally.</p>
          ) : (
            anomalies.map((a, i) => (
              <div className={`alert-item ${a.severity === 'WARN' ? 'warn' : ''}`} key={i}>
                <div className="alert-title">{a.type}</div>
                <div className="alert-desc">{a.message}</div>
                <div className="alert-desc" style={{ fontWeight: 600 }}>Action: {a.suggested_action}</div>
              </div>
            ))
          )}
        </div>

      </div>
    </div>
  );
};

export default Dashboard;
