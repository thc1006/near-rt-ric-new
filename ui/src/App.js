import React, { useState, useEffect } from 'react';
import './App.css';

function App() {
  const [networkFunctions, setNetworkFunctions] = useState([]);
  const [kpis, setKpis] = useState({});
  const [alarms, setAlarms] = useState([]);
  const [logs, setLogs] = useState([]);

  useEffect(() => {
    // Auto-discover network functions
    // This would typically be a call to an SMO API endpoint
    const discoveredFunctions = [
      { id: 'ric-1', name: 'near-RT RIC', type: 'RIC' },
      { id: 'ocu-1', name: 'O-CU Simulator', type: 'CU' },
      { id: 'odu-1', name: 'O-DU Simulator', type: 'DU' },
      { id: 'xapp-hello', name: 'Hello World xApp', type: 'xApp' },
    ];
    setNetworkFunctions(discoveredFunctions);

    // Fetch KPIs, alarms, and logs
    // This would be replaced with real-time data fetching from Kafka/Elasticsearch
    const interval = setInterval(() => {
      setKpis({
        'ric-1': { throughput: Math.random() * 100, latency: Math.random() * 10 },
        'xapp-hello': { subscriptions: Math.floor(Math.random() * 10) },
      });
      setAlarms([
        { id: 1, severity: 'critical', message: 'Connection to O-CU lost' },
        { id: 2, severity: 'warning', message: 'High latency on e2node-1' },
      ]);
      setLogs([
        { timestamp: new Date().toISOString(), service: 'ric-1', message: 'E2 Setup Request received' },
        { timestamp: new Date().toISOString(), service: 'xapp-hello', message: 'Subscription request sent' },
      ]);
    }, 2000);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="App">
      <header className="App-header">
        <h1>O-RAN Interactive Operations Console</h1>
      </header>
      <main>
        <div className="dashboard-container">
          <div className="panel network-functions">
            <h2>Network Functions</h2>
            <ul>
              {networkFunctions.map(nf => (
                <li key={nf.id}><strong>{nf.name}</strong> ({nf.type})</li>
              ))}
            </ul>
          </div>
          <div className="panel kpis">
            <h2>Real-Time KPIs</h2>
            {Object.entries(kpis).map(([nf, data]) => (
              <div key={nf}>
                <h3>{nf}</h3>
                <ul>
                  {Object.entries(data).map(([key, value]) => (
                    <li key={key}>{key}: {value.toFixed(2)}</li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
          <div className="panel alarms">
            <h2>Alarms</h2>
            <ul>
              {alarms.map(alarm => (
                <li key={alarm.id} className={`alarm-${alarm.severity}`}>
                  <strong>{alarm.severity.toUpperCase()}:</strong> {alarm.message}
                </li>
              ))}
            </ul>
          </div>
          <div className="panel logs">
            <h2>Logs</h2>
            <pre>
              {logs.map(log => `[${log.timestamp}] [${log.service}] ${log.message}\n`).join('')}
            </pre>
          </div>
        </div>
      </main>
    </div>
  );
}

export default App;