import { useState } from "react";
import type { LootEntry, LooterTotals } from "./types.ts";
import { fmt } from "./format.ts";

interface LootPanelProps {
	loot: LootEntry[];
	looterTotals: LooterTotals[];
	onClose: () => void;
}

// LootPanel is the "who farmed what" view. Two tabs:
//   Totals — per-looter rollup with item count + silver + estimated
//            value (AODP prices). Sorted: local first, then by value.
//   Items  — chronological loot log, most-recent first, with each
//            line showing the looter / item / qty / source / value.
//
// AODP prices update asynchronously in the background; the agent
// refreshes prices for newly-looted items within ~1s and the next
// snapshot reflects the value. Until then SilverValue is 0 (the
// "—" cell).
export function LootPanel({ loot, looterTotals, onClose }: LootPanelProps): React.ReactElement {
	const [tab, setTab] = useState<"totals" | "items">("totals");

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
					<div className="flex items-center" style={{ gap: 12 }}>
						<span className="sk-upper" style={{ color: "var(--sk-fg-0)", fontWeight: 600, fontSize: 12 }}>
							Loot · friends' farming
						</span>
						<div className="flex items-center" style={{ gap: 2, marginLeft: 12 }}>
							<TabBtn active={tab === "totals"} onClick={() => setTab("totals")}>Totals</TabBtn>
							<TabBtn active={tab === "items"} onClick={() => setTab("items")}>Items</TabBtn>
						</div>
						<span className="sk-mono" style={{ color: "var(--sk-fg-3)", fontSize: 11 }}>
							{loot.length} entr{loot.length === 1 ? "y" : "ies"}
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

				{tab === "totals" ? <TotalsTab looterTotals={looterTotals} /> : <ItemsTab loot={loot} />}
			</div>
		</div>
	);
}

function TabBtn({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }): React.ReactElement {
	return (
		<button
			onClick={onClick}
			style={{
				appearance: "none",
				border: 0,
				background: active ? "var(--sk-bg-3)" : "transparent",
				color: active ? "var(--sk-fg-0)" : "var(--sk-fg-2)",
				padding: "4px 10px",
				borderRadius: 4,
				cursor: "pointer",
				fontSize: 11,
				fontWeight: 600,
				boxShadow: active ? "inset 0 0 0 1px var(--sk-line-2)" : "none",
			}}
		>
			{children}
		</button>
	);
}

function TotalsTab({ looterTotals }: { looterTotals: LooterTotals[] }): React.ReactElement {
	if (looterTotals.length === 0) {
		return <Empty>No loot tracked yet — kill / loot something with friends nearby to populate this.</Empty>;
	}
	const grandTotal = looterTotals.reduce((s, l) => s + l.valueTotal, 0);
	return (
		<div style={{ overflowY: "auto" }}>
			<div
				className="grid items-center"
				style={{
					gridTemplateColumns: "1fr 70px 100px 120px",
					gap: 12,
					padding: "10px 16px",
					fontSize: 10,
					color: "var(--sk-fg-3)",
					textTransform: "uppercase",
					letterSpacing: "0.08em",
					borderBottom: "1px solid var(--sk-line)",
					background: "var(--sk-bg-inset)",
				}}
			>
				<span>Looter</span>
				<span style={{ textAlign: "right" }}>Items</span>
				<span style={{ textAlign: "right" }}>Silver</span>
				<span style={{ textAlign: "right" }}>Est. value</span>
			</div>
			{looterTotals.map((l) => (
				<div
					key={l.name}
					className="grid items-center"
					style={{
						gridTemplateColumns: "1fr 70px 100px 120px",
						gap: 12,
						padding: "10px 16px",
						borderBottom: "1px solid var(--sk-line)",
						background: l.isLocal ? "color-mix(in oklab, var(--sk-local) 5%, var(--sk-bg-1))" : "var(--sk-bg-1)",
						borderLeft: l.isLocal ? "2px solid var(--sk-local)" : "2px solid transparent",
					}}
				>
					<div className="flex items-center" style={{ gap: 8 }}>
						<span style={{
							fontSize: 14,
							fontWeight: 600,
							color: l.isLocal ? "var(--sk-local)" : "var(--sk-fg-0)",
						}}>{l.name}</span>
						{l.isLocal && (
							<span className="sk-upper" style={{ fontSize: 9, color: "var(--sk-local)" }}>You</span>
						)}
					</div>
					<span className="sk-mono" style={{ textAlign: "right", fontSize: 12, color: "var(--sk-fg-1)" }}>
						{l.itemCount}
					</span>
					<span className="sk-mono" style={{ textAlign: "right", fontSize: 12, color: "var(--sk-card-silver)" }}>
						{fmt(l.silverTotal)}
					</span>
					<span className="sk-mono" style={{ textAlign: "right", fontSize: 13, color: "var(--sk-card-fame)", fontWeight: 600 }}>
						{fmt(l.valueTotal)}
					</span>
				</div>
			))}
			<div
				className="grid items-center"
				style={{
					gridTemplateColumns: "1fr 120px",
					gap: 12,
					padding: "10px 16px",
					background: "var(--sk-bg-inset)",
					borderTop: "1px solid var(--sk-line-2)",
				}}
			>
				<span className="sk-upper" style={{ fontSize: 10, color: "var(--sk-fg-2)" }}>Party total</span>
				<span className="sk-mono" style={{ textAlign: "right", fontSize: 14, color: "var(--sk-card-fame)", fontWeight: 700 }}>
					{fmt(grandTotal)}
				</span>
			</div>
		</div>
	);
}

function ItemsTab({ loot }: { loot: LootEntry[] }): React.ReactElement {
	if (loot.length === 0) {
		return <Empty>No loot events captured yet.</Empty>;
	}
	const ordered = [...loot].reverse();
	return (
		<div style={{ overflowY: "auto" }}>
			<div
				className="grid items-center"
				style={{
					gridTemplateColumns: "60px 1fr 50px 100px 100px",
					gap: 10,
					padding: "10px 16px",
					fontSize: 10,
					color: "var(--sk-fg-3)",
					textTransform: "uppercase",
					letterSpacing: "0.08em",
					borderBottom: "1px solid var(--sk-line)",
					background: "var(--sk-bg-inset)",
				}}
			>
				<span>Time</span>
				<span>Item · Looter</span>
				<span style={{ textAlign: "right" }}>Qty</span>
				<span style={{ textAlign: "right" }}>Silver</span>
				<span style={{ textAlign: "right" }}>Value</span>
			</div>
			{ordered.map((l, i) => (
				<div
					key={`${l.at}-${i}`}
					className="grid items-center"
					style={{
						gridTemplateColumns: "60px 1fr 50px 100px 100px",
						gap: 10,
						padding: "8px 16px",
						borderBottom: "1px solid var(--sk-line)",
						background: l.looterIsLocal ? "color-mix(in oklab, var(--sk-local) 4%, var(--sk-bg-1))" : "var(--sk-bg-1)",
					}}
				>
					<span className="sk-mono" style={{ fontSize: 10, color: "var(--sk-fg-3)" }}>
						{new Date(l.at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
					</span>
					<div className="flex flex-col" style={{ gap: 1, minWidth: 0 }}>
						<span style={{ fontSize: 12, color: "var(--sk-fg-0)" }}>
							{l.isSilver ? "Silver pile" : (l.displayName || l.uniqueName || `#${l.itemIndex}`)}
						</span>
						<span style={{ fontSize: 10, color: "var(--sk-fg-3)" }}>
							{l.looter}{l.looterIsLocal ? " (you)" : ""}{l.lootedFrom ? ` · from ${l.lootedFrom}` : ""}
						</span>
					</div>
					<span className="sk-mono" style={{ textAlign: "right", fontSize: 11, color: "var(--sk-fg-1)" }}>
						{l.quantity}
					</span>
					<span className="sk-mono" style={{ textAlign: "right", fontSize: 11, color: l.isSilver ? "var(--sk-card-silver)" : "var(--sk-fg-3)" }}>
						{l.isSilver ? fmt(l.quantity) : "—"}
					</span>
					<span className="sk-mono" style={{ textAlign: "right", fontSize: 11, color: l.silverValue && l.silverValue > 0 ? "var(--sk-card-fame)" : "var(--sk-fg-3)" }}>
						{l.silverValue && l.silverValue > 0 ? fmt(l.silverValue) : "—"}
					</span>
				</div>
			))}
		</div>
	);
}

function Empty({ children }: { children: React.ReactNode }): React.ReactElement {
	return (
		<div className="flex items-center justify-center" style={{ padding: 56, color: "var(--sk-fg-2)", fontSize: 13, textAlign: "center" }}>
			{children}
		</div>
	);
}
