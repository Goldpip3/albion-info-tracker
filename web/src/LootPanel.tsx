import { LootBody } from "./LootBody.tsx";
import type { LootEntry, LooterTotals, PlayerSnapshot, Session } from "./types.ts";

interface LootPanelProps {
	loot: LootEntry[];
	looterTotals: LooterTotals[];
	players: PlayerSnapshot[];
	session?: Session | null;
	generatedAt?: string;
	onClose: () => void;
	onOpenAsPage?: () => void;
}

// LootPanel is the modal wrapper. The actual content lives in LootBody
// so the fullscreen /loot route can render the same view without modal
// chrome — backdrop, max-width, Close button.
export function LootPanel({ loot, looterTotals, players, session, generatedAt, onClose, onOpenAsPage }: LootPanelProps): React.ReactElement {
	return (
		<div
			className="fixed inset-0 z-50 flex items-center justify-center p-4"
			style={{ background: "rgba(0,0,0,0.6)", backdropFilter: "blur(4px)" }}
			onClick={onClose}
		>
			<div
				className="w-full max-w-3xl flex flex-col"
				style={{
					background: "var(--sk-bg-1)",
					border: "1px solid var(--sk-line)",
					borderRadius: 8,
					overflow: "hidden",
					maxHeight: "90vh",
				}}
				onClick={(e) => e.stopPropagation()}
			>
				<div
					className="flex items-center justify-between px-4 py-3"
					style={{ borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-inset)" }}
				>
					<span className="sk-upper" style={{ color: "var(--sk-fg-0)", fontWeight: 600, fontSize: 12 }}>
						Loot · who farmed what
					</span>
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
				<LootBody
					loot={loot}
					looterTotals={looterTotals}
					players={players}
					session={session ?? null}
					generatedAt={generatedAt}
					onOpenAsPage={onOpenAsPage}
				/>
			</div>
		</div>
	);
}
