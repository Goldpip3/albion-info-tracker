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
	return (
		<div
			className="flex items-center justify-between px-4 py-2"
			style={{
				borderTop: "1px solid var(--sk-line)",
				background: "var(--sk-bg-1)",
			}}
		>
			<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>
				Tick · 0.5s &nbsp;·&nbsp; Last update {ago}
				{snapshot && (
					<>
						&nbsp;·&nbsp; <span className="sk-mono" style={{ textTransform: "none", letterSpacing: 0 }}>
							{new Date(snapshot.generatedAt).toLocaleTimeString()}
						</span>
					</>
				)}
			</span>
			<span className="sk-upper sk-mono" style={{ color: "var(--sk-fg-3)" }}>
				{BUILD}
			</span>
		</div>
	);
}
