import type { Snapshot } from "./types.ts";

interface FooterProps {
	snapshot: Snapshot | null;
	lastMessageAt: number | null;
}

const BUILD = "V0.1 · BUILD T1A"; // bumped per release

export function Footer({ snapshot, lastMessageAt }: FooterProps): React.ReactElement {
	const ago = lastMessageAt ? ((Date.now() - lastMessageAt) / 1000).toFixed(1) : "—";
	return (
		<footer className="border-t border-skirmish-line px-4 py-1.5 flex items-center text-[10px] tracking-[0.12em] uppercase text-skirmish-muted">
			<span>Tick · 0.5s</span>
			<span className="mx-3 text-skirmish-line">|</span>
			<span>Last update {ago}s ago</span>
			<span className="ml-auto">{BUILD}</span>
			{snapshot && (
				<>
					<span className="mx-3 text-skirmish-line">|</span>
					<span>{new Date(snapshot.generatedAt).toLocaleTimeString()}</span>
				</>
			)}
		</footer>
	);
}
