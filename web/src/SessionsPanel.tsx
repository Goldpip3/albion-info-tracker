import { useState } from "react";
import type { ArchivedSession } from "./types.ts";
import { fmt, fmtDuration } from "./format.ts";

interface SessionsPanelProps {
	sessions: ArchivedSession[];
	onDelete: (id: string) => void;
	onClose: () => void;
}

// SessionsPanel lists every archived session the agent has on disk and
// lets the user delete any of them. Sessions are written when the user
// hits "New session" — the prior run gets a JSON file under
// %LocalAppData%\GDA\sessions on Windows. The list ships in every
// snapshot's `sessions` field; deletion goes through the existing
// command channel as { action: "deleteSession", arg: <id> }.
export function SessionsPanel({ sessions, onDelete, onClose }: SessionsPanelProps): React.ReactElement {
	const [confirming, setConfirming] = useState<string | null>(null);

	return (
		<div
			className="fixed inset-0 z-50 flex items-center justify-center p-4"
			style={{ background: "rgba(0,0,0,0.6)", backdropFilter: "blur(4px)" }}
			onClick={onClose}
		>
			<div
				className="w-full max-w-2xl flex flex-col"
				style={{
					background: "var(--sk-bg-1)",
					border: "1px solid var(--sk-line)",
					borderRadius: 8,
					overflow: "hidden",
					maxHeight: "85vh",
				}}
				onClick={(e) => e.stopPropagation()}
			>
				<div
					className="flex items-center justify-between px-4 py-3"
					style={{ borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-inset)" }}
				>
					<div className="flex items-center" style={{ gap: 12 }}>
						<span className="sk-upper" style={{ color: "var(--sk-fg-0)", fontWeight: 600, fontSize: 12 }}>
							Sessions on disk
						</span>
						<span className="sk-mono" style={{ color: "var(--sk-fg-3)", fontSize: 11 }}>
							{sessions.length} archived
						</span>
					</div>
					<button
						onClick={onClose}
						className="sk-upper"
						style={{
							appearance: "none",
							border: "1px solid var(--sk-line-2)",
							background: "var(--sk-bg-3)",
							color: "var(--sk-fg-1)",
							padding: "4px 9px",
							borderRadius: 4,
							cursor: "pointer",
							fontSize: 10,
							letterSpacing: "0.08em",
							fontWeight: 600,
						}}
					>
						Close
					</button>
				</div>

				{sessions.length === 0 && (
					<div className="flex items-center justify-center" style={{ padding: 56, color: "var(--sk-fg-2)", fontSize: 13, textAlign: "center", lineHeight: 1.5 }}>
						No archived sessions yet.<br/>
						<span style={{ fontSize: 11, color: "var(--sk-fg-3)" }}>
							Hit "New session" at the end of a run — it'll be saved here for review.
						</span>
					</div>
				)}

				<div style={{ overflowY: "auto" }}>
					{sessions.map((s) => (
						<Row
							key={s.id}
							s={s}
							confirming={confirming === s.id}
							onConfirmStart={() => setConfirming(s.id)}
							onConfirmCancel={() => setConfirming(null)}
							onConfirmDelete={() => { onDelete(s.id); setConfirming(null); }}
						/>
					))}
				</div>
			</div>
		</div>
	);
}

function Row({
	s, confirming, onConfirmStart, onConfirmCancel, onConfirmDelete,
}: {
	s: ArchivedSession;
	confirming: boolean;
	onConfirmStart: () => void;
	onConfirmCancel: () => void;
	onConfirmDelete: () => void;
}): React.ReactElement {
	const started = new Date(s.startedAt);
	const dateLabel = started.toLocaleString(undefined, {
		month: "short", day: "numeric", hour: "numeric", minute: "2-digit",
	});
	const seconds = Math.floor(s.durationMs / 1000);
	const fame = s.fameTotal / 10_000;
	const silver = s.silverTotal / 10_000;
	const respec = s.respecTotal / 10_000;
	const might = s.mightTotal / 10_000;

	return (
		<div
			className="grid items-center"
			style={{
				gridTemplateColumns: "auto 1fr auto auto",
				gap: 14,
				padding: "10px 16px",
				borderBottom: "1px solid var(--sk-line)",
				background: "var(--sk-bg-1)",
			}}
		>
			<div className="flex flex-col" style={{ gap: 2, minWidth: 120 }}>
				<span style={{ fontSize: 13, fontWeight: 600, color: "var(--sk-fg-0)" }}>
					{s.localName || dateLabel}
				</span>
				<span className="sk-upper" style={{ fontSize: 9.5, color: "var(--sk-fg-3)" }}>
					{s.localName ? dateLabel : (s.zone || "—")}
				</span>
			</div>
			<div className="flex items-baseline" style={{ gap: 14 }}>
				<EconStat label="Fame"   value={fmt(fame)}   accent="var(--sk-card-fame)" />
				<EconStat label="Silver" value={fmt(silver)} accent="var(--sk-card-silver)" />
				<EconStat label="Respec" value={kFmt(respec)} accent="var(--sk-card-respec)" />
				<EconStat label="Might"  value={kFmt(might)}  accent="var(--sk-card-might)" />
			</div>
			<span className="sk-mono" style={{ fontSize: 11, color: "var(--sk-fg-2)", textAlign: "right" }}>
				{fmtDuration(seconds)}
			</span>
			{confirming ? (
				<div className="flex items-center" style={{ gap: 4 }}>
					<button
						onClick={onConfirmDelete}
						className="sk-upper"
						style={{
							appearance: "none",
							border: "1px solid var(--sk-err)",
							background: "color-mix(in oklab, var(--sk-err) 18%, transparent)",
							color: "var(--sk-err)",
							padding: "3px 8px",
							borderRadius: 4,
							cursor: "pointer",
							fontSize: 10,
							fontWeight: 600,
							letterSpacing: "0.08em",
						}}
					>
						Delete
					</button>
					<button
						onClick={onConfirmCancel}
						className="sk-upper"
						style={{
							appearance: "none",
							border: "1px solid var(--sk-line-2)",
							background: "var(--sk-bg-3)",
							color: "var(--sk-fg-2)",
							padding: "3px 8px",
							borderRadius: 4,
							cursor: "pointer",
							fontSize: 10,
							fontWeight: 600,
							letterSpacing: "0.08em",
						}}
					>
						Cancel
					</button>
				</div>
			) : (
				<button
					onClick={onConfirmStart}
					title="Delete this archived session"
					style={{
						appearance: "none",
						border: "1px solid var(--sk-line)",
						background: "var(--sk-bg-2)",
						color: "var(--sk-fg-3)",
						padding: "3px 8px",
						borderRadius: 4,
						cursor: "pointer",
						fontSize: 12,
						lineHeight: 1,
					}}
				>
					✕
				</button>
			)}
		</div>
	);
}

function EconStat({ label, value, accent }: { label: string; value: string; accent: string }): React.ReactElement {
	return (
		<div className="flex flex-col items-start" style={{ gap: 1 }}>
			<span className="sk-upper" style={{ fontSize: 8.5, color: "var(--sk-fg-3)" }}>{label}</span>
			<span className="sk-mono" style={{ fontSize: 12, color: accent, fontWeight: 600 }}>{value}</span>
		</div>
	);
}

function kFmt(n: number): string {
	if (n <= 0) return "0";
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
	if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
	return n.toFixed(0);
}
