import type { Snapshot } from "./types.ts";

interface FooterProps {
	snapshot: Snapshot | null;
	lastMessageAt: number | null;
}

const BUILD = "v0.5.0 · build t1f";

export function Footer({ snapshot, lastMessageAt }: FooterProps): React.ReactElement {
	const ago = lastMessageAt
		? `${((Date.now() - lastMessageAt) / 1000).toFixed(1)}s ago`
		: "—";
	const status = lastMessageAt && Date.now() - lastMessageAt < 5000 ? "200 OK" : "stale";
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
			}}
		>
			<span>
				tick · 0.5s &nbsp;·&nbsp; last update {ago}
				{snapshot && (
					<>
						&nbsp;·&nbsp; <span style={{ letterSpacing: 0 }}>
							{new Date(snapshot.generatedAt).toLocaleTimeString()}
						</span>
					</>
				)}
			</span>
			<span>
				{BUILD} · {status}
			</span>
		</div>
	);
}
