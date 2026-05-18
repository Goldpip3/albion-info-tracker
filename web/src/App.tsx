import { useEffect, useRef, useState } from 'react';
import './App.css';

const WS_URL = 'ws://127.0.0.1:9696';
const RECONNECT_DELAY_MS = 2000;
const RENDER_BASE = 'https://render.albiononline.com/v1/item';

type PlayerSnapshot = {
  playerId: string;
  name: string;
  weaponItemId: string | null;
  totalDamage: number;
  totalHeal: number;
  totalTaken: number;
  dps: number;
  hps: number;
  fame: number;
  silver: number;
};

type ServerMessage =
  | { type: 'hello'; ts: number; sessionId: string; players: PlayerSnapshot[] }
  | { type: 'playersUpdate'; ts: number; players: PlayerSnapshot[] }
  | { type: 'playerJoined'; ts: number; playerId: string; name: string; weaponItemId: string | null }
  | { type: 'playerLeft'; ts: number; playerId: string }
  | { type: 'weaponEquipped'; ts: number; playerId: string; weaponItemId: string | null }
  | { type: 'fameUpdate'; ts: number; fame: number; combatFame: number; silver: number }
  | { type: 'sessionReset'; ts: number; sessionId: string };

type Status = 'connecting' | 'connected' | 'disconnected';

const fmt = (n: number) => n.toLocaleString('en-US');
const fmtRate = (n: number) => {
  if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M';
  if (n >= 1e4) return (n / 1e3).toFixed(0) + 'k';
  if (n >= 1e3) return (n / 1e3).toFixed(1) + 'k';
  return Math.round(n).toString();
};

function App() {
  const [status, setStatus] = useState<Status>('connecting');
  const [sessionId, setSessionId] = useState<string>('');
  const [players, setPlayers] = useState<PlayerSnapshot[]>([]);
  const [fame, setFame] = useState(0);
  const [combatFame, setCombatFame] = useState(0);
  const [silver, setSilver] = useState(0);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    let cancelled = false;
    let reconnectHandle: number | null = null;

    const connect = () => {
      if (cancelled) return;
      setStatus('connecting');
      const ws = new WebSocket(WS_URL);
      wsRef.current = ws;

      ws.onopen = () => setStatus('connected');

      ws.onmessage = (ev) => {
        let msg: ServerMessage;
        try {
          msg = JSON.parse(ev.data) as ServerMessage;
        } catch {
          return;
        }
        switch (msg.type) {
          case 'hello':
            setSessionId(msg.sessionId);
            setPlayers(msg.players);
            break;
          case 'playersUpdate':
            setPlayers(msg.players);
            break;
          case 'playerJoined':
            setPlayers((prev) =>
              prev.some((p) => p.playerId === msg.playerId)
                ? prev
                : [
                    ...prev,
                    {
                      playerId: msg.playerId,
                      name: msg.name,
                      weaponItemId: msg.weaponItemId,
                      totalDamage: 0,
                      totalHeal: 0,
                      totalTaken: 0,
                      dps: 0,
                      hps: 0,
                      fame: 0,
                      silver: 0,
                    },
                  ],
            );
            break;
          case 'playerLeft':
            setPlayers((prev) => prev.filter((p) => p.playerId !== msg.playerId));
            break;
          case 'weaponEquipped':
            setPlayers((prev) =>
              prev.map((p) =>
                p.playerId === msg.playerId ? { ...p, weaponItemId: msg.weaponItemId } : p,
              ),
            );
            break;
          case 'fameUpdate':
            setFame(msg.fame);
            setCombatFame(msg.combatFame);
            setSilver(msg.silver);
            break;
          case 'sessionReset':
            setSessionId(msg.sessionId);
            setPlayers((prev) =>
              prev.map((p) => ({ ...p, totalDamage: 0, totalHeal: 0, totalTaken: 0, dps: 0, hps: 0, fame: 0, silver: 0 })),
            );
            setFame(0);
            setCombatFame(0);
            setSilver(0);
            break;
        }
      };

      ws.onclose = () => {
        wsRef.current = null;
        if (!cancelled) {
          setStatus('disconnected');
          reconnectHandle = window.setTimeout(connect, RECONNECT_DELAY_MS);
        }
      };

      ws.onerror = () => {
        // onclose will fire next; just let it handle reconnection.
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (reconnectHandle !== null) window.clearTimeout(reconnectHandle);
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, []);

  const sendReset = () => {
    wsRef.current?.send(JSON.stringify({ type: 'reset' }));
  };

  const sorted = [...players].sort((a, b) => b.totalDamage - a.totalDamage);
  const maxDamage = Math.max(1, ...sorted.map((p) => p.totalDamage));

  return (
    <div className="app">
      <header className="topbar">
        <div className="title">
          <h1>Albion Info Tracker</h1>
          <span className={`status status-${status}`}>
            <span className="dot" />
            {status === 'connected' ? `connected · session ${sessionId.slice(0, 8)}` : status}
          </span>
        </div>
        <button className="reset" onClick={sendReset} disabled={status !== 'connected'}>
          Reset
        </button>
      </header>

      <section className="totals">
        <div><label>Fame</label><span>{fmt(fame)}</span></div>
        <div><label>Combat Fame</label><span>{fmt(combatFame)}</span></div>
        <div><label>Silver</label><span>{fmt(silver)}</span></div>
        <div><label>Party</label><span>{players.length}</span></div>
      </section>

      <section className="meter">
        {sorted.length === 0 ? (
          <p className="empty">
            {status === 'connected'
              ? 'Waiting for combat events… make sure the service is running, you are in a party, and you zoned in after starting it.'
              : 'Not connected to the tracker service. Start AlbionInfoTracker.dll, then this page will auto-reconnect.'}
          </p>
        ) : (
          <ul className="rows">
            {sorted.map((p) => {
              const widthPct = (p.totalDamage / maxDamage) * 100;
              return (
                <li key={p.playerId} className="row">
                  <div className="bar" style={{ width: `${widthPct}%` }} />
                  <div className="content">
                    {p.weaponItemId ? (
                      <img
                        className="icon"
                        src={`${RENDER_BASE}/${encodeURIComponent(p.weaponItemId)}.png?size=32`}
                        alt=""
                        loading="lazy"
                      />
                    ) : (
                      <span className="icon icon-empty" />
                    )}
                    <span className="name">{p.name || '(unknown)'}</span>
                    <span className="metric metric-dmg">
                      <span className="value">{fmt(p.totalDamage)}</span>
                      <span className="rate">{fmtRate(p.dps)}/s</span>
                    </span>
                    <span className="metric metric-heal">
                      <span className="value">{fmt(p.totalHeal)}</span>
                      <span className="rate">{fmtRate(p.hps)}/s</span>
                    </span>
                    <span className="metric metric-taken">
                      <span className="value">{fmt(p.totalTaken)}</span>
                    </span>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
        {sorted.length > 0 && (
          <div className="legend">
            <span className="legend-dmg">damage</span>
            <span className="legend-heal">heal</span>
            <span className="legend-taken">taken</span>
          </div>
        )}
      </section>
    </div>
  );
}

export default App;
