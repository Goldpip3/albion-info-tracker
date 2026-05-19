import type { Snapshot } from "./types.ts";

interface FooterProps {
	snapshot: Snapshot | null;
	lastMessageAt: number | null;
	onOpenSessions?: () => void;
	onOpenParty?: () => void;
	onOpenLoot?: () => void;
	sessionsCount?: number;
	lootCount?: number;
}

const BUILD = "v0.6.0";

// Footer is the mono inset chrome strip at the bottom of the meter.
// Houses the tick info on the left, build / status on the right, plus
// quick-access buttons for the modal panels (Sessions, Party).
export function Footer({ snapshot, lastMessageAt, onOpenSessions, onOpenParty, onOpenLoot, sessionsCount = 0, lootCount = 0 }: FooterProps): React.ReactElement {
	const ago = lastMessageAt
		? `${((Date.now() - lastMessageAt) / 1000).toFixed(1)}s ago`
		: "—";
	const status = lastMessageAt && Date.now() - lastMessageAt < 5000 ? "live" : "stale";
	return (
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
			<div className="flex items-center" style={{ gap: 8 }}>
				{onOpenParty && (
					<FooterBtn onClick={onOpenParty} title="Party loadout (live)">
						Party
					</FooterBtn>
				)}
				{onOpenLoot && (
					<FooterBtn onClick={onOpenLoot} title="Loot log + friends' farm rollup">
						Loot{lootCount > 0 ? ` · ${lootCount}` : ""}
					</FooterBtn>
				)}
				{onOpenSessions && (
					<FooterBtn onClick={onOpenSessions} title="Archived sessions on disk">
						Sessions{sessionsCount > 0 ? ` · ${sessionsCount}` : ""}
					</FooterBtn>
				)}
				<span>{BUILD} · {status}</span>
			</div>
		</div>
	);
}

function FooterBtn({ onClick, title, children }: { onClick: () => void; title?: string; children: React.ReactNode }): React.ReactElement {
	return (
		<button
			onClick={onClick}
			title={title}
			style={{
				appearance: "none",
				border: "1px solid var(--sk-line)",
				background: "var(--sk-bg-2)",
				color: "var(--sk-fg-2)",
				padding: "3px 8px",
				borderRadius: 3,
				cursor: "pointer",
				fontFamily: "inherit",
				fontSize: 10,
				letterSpacing: "0.06em",
				textTransform: "uppercase",
				fontWeight: 600,
			}}
		>
			{children}
		</button>
	);
}
