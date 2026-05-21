import type { Snapshot } from "./types.ts";

interface FooterProps {
	snapshot: Snapshot | null;
	lastMessageAt: number | null;
}

const BUILD = "v0.6.0";

// Footer is the mono inset chrome strip at the bottom of the meter.
// Quick-access shortcuts moved to the TabBar at the top, so this
// surface is back to being purely informational: tick rate, last update
// timestamp, build version, live/stale indicator.
//
// A 2-px combat band sits above the chrome row — transparent when out
// of combat, animated orange when the agent reports fight.inCombat ===
// true. At-a-glance "are we fighting" without having to read the
// header timer. Inspired by the in-combat band common to MMO raid
// frames.
export function Footer({ snapshot, lastMessageAt }: FooterProps): React.ReactElement {
	const ago = lastMessageAt
		? `${((Date.now() - lastMessageAt) / 1000).toFixed(1)}s ago`
		: "—";
	// 20s grace: the agent's idle heartbeat is 15s, so a tighter window would
	// falsely flag a healthy-but-idle agent as stale.
	const status = lastMessageAt && Date.now() - lastMessageAt < 20000 ? "live" : "stale";
	const inCombat = snapshot?.fight?.inCombat ?? false;
	return (
		<div style={{ flexShrink: 0 }}>
			<div
				aria-hidden
				style={{
					height: 2,
					background: inCombat ? "var(--sk-damage)" : "transparent",
					boxShadow: inCombat
						? "0 0 12px color-mix(in oklab, var(--sk-damage) 60%, transparent)"
						: "none",
					transition: "background 220ms var(--sk-ease), box-shadow 220ms var(--sk-ease)",
				}}
			/>
			<div
				className="flex items-center justify-between"
				style={{
					padding: "8px 16px",
					borderTop: "1px solid var(--sk-line)",
					background: "var(--sk-bg-inset)",
					fontFamily: "var(--sk-font-mono)",
					fontSize: 10,
					color: "var(--sk-fg-3)",
					letterSpacing: "0.06em",
					gap: 12,
				}}
			>
				<span>
					push 0.2s &nbsp;·&nbsp; spark 2s &nbsp;·&nbsp; last update {ago}
					{snapshot && (
						<>
							&nbsp;·&nbsp; <span style={{ letterSpacing: 0 }}>
								{new Date(snapshot.generatedAt).toLocaleTimeString()}
							</span>
						</>
					)}
				</span>
				<span>{BUILD} · {status}</span>
			</div>
		</div>
	);
}
