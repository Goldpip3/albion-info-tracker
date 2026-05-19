import { useMemo, useState } from "react";
import type { LootEntry, LooterTotals, PlayerSnapshot, Session } from "./types.ts";
import { fmt, roleKeyOf } from "./format.ts";
import { IPChip } from "./IPChip.tsx";

// Silver-pile pickups arrive on the wire as FixPoint copper (1 silver
// = 10_000 copper) to mirror Albion's internal accounting. AODP item
// values come back already in silver. These helpers keep the two
// scales straight at the display boundary so callers never have to
// remember which way a particular field is denominated.
const toSilver = (copper: number): number => Math.floor(copper / 10_000);
const totalSilverOf = (l: LooterTotals): number => toSilver(l.silverPicked) + l.silverValueLoot;

export interface LootBodyProps {
	loot: LootEntry[];
	looterTotals: LooterTotals[];
	players: PlayerSnapshot[];
	session?: Session | null;
	generatedAt?: string;
	// Modal-only affordances. LootPage leaves these undefined so the
	// extra buttons disappear cleanly.
	onOpenAsPage?: () => void;
}

type Tab = "perPlayer" | "items";

// LootBody is the part of the loot screen that's shared between the
// modal (LootPanel) and the fullscreen route (LootPage). The wrapping
// chrome differs — backdrop, max-width, close button — but the rows
// and the per-player drill-in live here so the two surfaces never
// drift.
export function LootBody({ loot, looterTotals, players, session, generatedAt, onOpenAsPage }: LootBodyProps): React.ReactElement {
	const [tab, setTab] = useState<Tab>("perPlayer");
	const [drillName, setDrillName] = useState<string | null>(null);
	const [toast, setToast] = useState<string | null>(null);

	// Sort + grand-total + top-value memo. Recomputes when the agent
	// posts a new snapshot (generatedAt flips) — not on every internal
	// state change like opening a tooltip.
	const { sorted, topValue, grandTotal } = useMemo(() => {
		const arr = [...looterTotals].sort((a, b) => {
			const d = totalSilverOf(b) - totalSilverOf(a);
			if (d !== 0) return d;
			return a.name.localeCompare(b.name);
		});
		const top = arr.length > 0 ? totalSilverOf(arr[0]) : 0;
		const grand = arr.reduce((s, l) => s + totalSilverOf(l), 0);
		return { sorted: arr, topValue: top, grandTotal: grand };
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [generatedAt, looterTotals.length]);

	// O(n) lookup into players by name for the IPChip on each row.
	const playerByName = useMemo(() => {
		const m = new Map<string, PlayerSnapshot>();
		for (const p of players) {
			if (p.name) m.set(p.name, p);
		}
		return m;
	}, [players]);

	const drillEntries = useMemo(() => {
		if (!drillName) return [];
		return loot.filter((l) => l.looter === drillName).sort((a, b) => +new Date(b.at) - +new Date(a.at));
	}, [drillName, loot]);

	const totalPickups = sorted.reduce((s, l) => s + l.pickups, 0);
	const sessionMin = Math.max(0, Math.floor((session?.elapsedMs ?? 0) / 60_000));

	const fireToast = (msg: string): void => {
		setToast(msg);
		window.setTimeout(() => setToast(null), 1500);
	};

	const onCopySummary = (): void => {
		const lines = [`Session loot · ${sessionMin}m`];
		sorted.forEach((l, i) => {
			const top = l.topItemName ? `  ·  top: ${l.topItemName} (${fmt(l.topItemValue ?? 0)})` : "";
			lines.push(`${i + 1}. ${l.name}  ${fmt(totalSilverOf(l))} silver  ·  ${l.pickups} pickups${top}`);
		});
		navigator.clipboard.writeText(lines.join("\n")).then(
			() => fireToast("Copied"),
			() => fireToast("Copy failed"),
		);
	};

	const onCopyTable = (): void => {
		const rows = sorted.map((l, i) => {
			const total = fmt(totalSilverOf(l));
			const top = l.topItemName ? `${l.topItemName} (${fmt(l.topItemValue ?? 0)})` : "—";
			return [String(i + 1), l.name, total, String(l.pickups), top];
		});
		const headers = ["#", "Looter", "Silver", "Picks", "Top item"];
		const widths = headers.map((h, i) => Math.max(h.length, ...rows.map((r) => r[i].length)));
		const fmtRow = (r: string[]): string => r.map((c, i) => c.padEnd(widths[i])).join("  ");
		const lines = ["```", `Session loot · ${sessionMin}m`, fmtRow(headers), ...rows.map(fmtRow), "```"];
		navigator.clipboard.writeText(lines.join("\n")).then(
			() => fireToast("Copied"),
			() => fireToast("Copy failed"),
		);
	};

	return (
		<div className="flex flex-col" style={{ minHeight: 0, flex: 1, position: "relative" }}>
			{/* Header strip: tabs (or back), meta counters, copy + open-as-page buttons */}
			<div
				className="flex items-center"
				style={{
					gap: 12,
					padding: "10px 16px",
					borderBottom: "1px solid var(--sk-line)",
					background: "var(--sk-bg-inset)",
				}}
			>
				{drillName ? (
					<button
						onClick={() => setDrillName(null)}
						className="sk-upper"
						style={ghostBtn}
					>
						← Back
					</button>
				) : (
					<div className="flex items-center" style={{ gap: 2 }}>
						<TabBtn active={tab === "perPlayer"} onClick={() => setTab("perPlayer")}>Per-Player</TabBtn>
						<TabBtn active={tab === "items"} onClick={() => setTab("items")}>Items</TabBtn>
					</div>
				)}
				<span
					className="sk-mono"
					style={{ fontSize: 11, color: "var(--sk-fg-3)", letterSpacing: "0.04em", marginLeft: 8 }}
				>
					{drillName
						? `${drillName} · ${drillEntries.length} pickups`
						: `SESSION LOOT · ${sorted.length} looter${sorted.length === 1 ? "" : "s"} · ${totalPickups} pickups · ${sessionMin}m elapsed`}
				</span>
				<div className="flex items-center" style={{ gap: 6, marginLeft: "auto" }}>
					<button onClick={onCopySummary} className="sk-upper" style={ghostBtn} title="Copy plain-text summary">Copy Summary</button>
					<button onClick={onCopyTable} className="sk-upper" style={ghostBtn} title="Copy as Discord codeblock">Copy as table</button>
					{onOpenAsPage && (
						<button onClick={onOpenAsPage} className="sk-upper" style={ghostBtn} title="Open this view in a new tab">Open as page</button>
					)}
				</div>
			</div>

			{/* Body */}
			{drillName ? (
				<DrillView
					name={drillName}
					entries={drillEntries}
					row={sorted.find((l) => l.name === drillName)}
				/>
			) : tab === "perPlayer" ? (
				<PerPlayerTab
					sorted={sorted}
					topValue={topValue}
					grandTotal={grandTotal}
					playerByName={playerByName}
					onOpenLooter={setDrillName}
				/>
			) : (
				<ItemsTab loot={loot} />
			)}

			{toast && <Toast text={toast} />}
		</div>
	);
}

function PerPlayerTab({
	sorted,
	topValue,
	grandTotal,
	playerByName,
	onOpenLooter,
}: {
	sorted: LooterTotals[];
	topValue: number;
	grandTotal: number;
	playerByName: Map<string, PlayerSnapshot>;
	onOpenLooter: (name: string) => void;
}): React.ReactElement {
	if (sorted.length === 0) {
		return (
			<Empty>
				No loot yet this session. Kills and chest opens will populate
				this once you start farming.
			</Empty>
		);
	}
	const top = sorted[0];
	const rest = sorted.slice(1);
	return (
		<div style={{ overflowY: "auto", flex: 1 }}>
			<TopFarmerCard l={top} playerByName={playerByName} />
			<LooterHeader />
			<div>
				{rest.map((l) => (
					<LooterRow
						key={l.name}
						l={l}
						topValue={topValue}
						player={playerByName.get(l.name)}
						onClick={() => onOpenLooter(l.name)}
					/>
				))}
			</div>
			<GrandTotal grandTotal={grandTotal} />
		</div>
	);
}

// LooterHeader is the sticky column-label row that sits between the
// Top Farmer card and the per-player list. Without it the bare `34`
// and `473K` cells read as "what does this mean" — the user explicitly
// asked for headers so the units travel with the numbers.
function LooterHeader(): React.ReactElement {
	return (
		<div
			className="grid items-center sk-upper"
			style={{
				gridTemplateColumns: "44px minmax(180px, 1.4fr) minmax(0, 2fr) 70px 100px 16px",
				gap: 12,
				padding: "8px 16px",
				position: "sticky",
				top: 0,
				background: "var(--sk-bg-inset)",
				borderBottom: "1px solid var(--sk-line)",
				fontSize: 10,
				color: "var(--sk-fg-3)",
				letterSpacing: "0.08em",
				fontWeight: 600,
				zIndex: 1,
			}}
		>
			<span />
			<span>Player</span>
			<span>Relative to top</span>
			<span style={{ textAlign: "right" }}>Pickups</span>
			<span style={{ textAlign: "right" }}>Silver</span>
			<span />
		</div>
	);
}

function TopFarmerCard({ l, playerByName }: { l: LooterTotals; playerByName: Map<string, PlayerSnapshot> }): React.ReactElement {
	const total = totalSilverOf(l);
	const player = playerByName.get(l.name);
	const roleKey = roleKeyOf(player?.role);
	return (
		<div
			style={{
				display: "flex",
				alignItems: "center",
				gap: 16,
				padding: "18px 20px",
				background: l.isLocal
					? "color-mix(in oklab, var(--sk-local) 8%, var(--sk-bg-2))"
					: "color-mix(in oklab, var(--sk-card-fame, #d4af37) 6%, var(--sk-bg-2))",
				borderBottom: "1px solid var(--sk-line-2)",
				borderLeft: l.isLocal ? "3px solid var(--sk-local)" : "3px solid var(--sk-card-fame, #d4af37)",
			}}
		>
			{player ? (
				<IPChip
					itemPower={player.itemPower}
					classCode={player.classCode}
					roleKey={roleKey}
					slots={player.equipmentSlots}
					size={36}
				/>
			) : (
				<Placeholder size={36} />
			)}
			<div className="flex flex-col" style={{ gap: 3, minWidth: 0, flex: 1 }}>
				<div className="flex items-baseline" style={{ gap: 8 }}>
					<span className="sk-upper" style={{ fontSize: 9.5, color: "var(--sk-fg-3)" }}>Top farmer</span>
					<span
						style={{
							fontSize: 17,
							fontWeight: 600,
							color: l.isLocal ? "var(--sk-local)" : "var(--sk-fg-0)",
						}}
					>
						{l.name}
					</span>
					{l.isLocal && <span className="sk-upper" style={{ fontSize: 9, color: "var(--sk-local)" }}>You</span>}
					{!l.isLocal && <SourceBadge source={l.source} />}
				</div>
				<div style={{ fontSize: 12, color: "var(--sk-fg-2)" }}>
					<LooterSubline l={l} />
				</div>
			</div>
			<div className="flex flex-col items-end" style={{ gap: 2 }}>
				<span
					className="sk-mono"
					style={{ fontSize: 22, fontWeight: 700, color: "var(--sk-card-fame, #d4af37)", letterSpacing: "-0.02em" }}
				>
					{fmt(total)}
				</span>
				<span className="sk-upper" style={{ fontSize: 9.5, color: "var(--sk-fg-3)" }}>
					{l.pickups} pickups
				</span>
			</div>
		</div>
	);
}

function LooterRow({
	l,
	topValue,
	player,
	onClick,
}: {
	l: LooterTotals;
	topValue: number;
	player?: PlayerSnapshot;
	onClick: () => void;
}): React.ReactElement {
	const total = totalSilverOf(l);
	const pct = topValue > 0 ? Math.min(100, (total / topValue) * 100) : 0;
	const roleKey = roleKeyOf(player?.role);
	const goldFill = "var(--sk-card-fame, #d4af37)";
	const lastMs = l.lastPickupAt ? Date.now() - +new Date(l.lastPickupAt) : Number.POSITIVE_INFINITY;
	const activeRatio = Math.max(0, Math.min(1, 1 - lastMs / 30_000));
	return (
		<div
			onClick={onClick}
			className="grid items-center"
			style={{
				gridTemplateColumns: "44px minmax(180px, 1.4fr) minmax(0, 2fr) 70px 100px 16px",
				gap: 12,
				padding: "11px 16px",
				borderBottom: "1px solid var(--sk-line)",
				background: l.isLocal ? "color-mix(in oklab, var(--sk-local) 5%, var(--sk-bg-1))" : "var(--sk-bg-1)",
				borderLeft: l.isLocal ? "2px solid var(--sk-local)" : "2px solid transparent",
				cursor: "pointer",
				transition: "background 160ms var(--sk-ease)",
			}}
		>
			{player ? (
				<IPChip itemPower={player.itemPower} classCode={player.classCode} roleKey={roleKey} slots={player.equipmentSlots} size={30} />
			) : (
				<Placeholder size={30} />
			)}

			<div className="flex flex-col min-w-0" style={{ gap: 2, lineHeight: 1.2 }}>
				<span
					className="truncate"
					style={{
						fontSize: 14,
						fontWeight: 600,
						color: l.isLocal ? "var(--sk-local)" : "var(--sk-fg-0)",
					}}
				>
					{l.name}
					{l.isLocal && <span className="sk-upper" style={{ fontSize: 9, color: "var(--sk-local)", marginLeft: 6 }}>You</span>}
					{!l.isLocal && <SourceBadge source={l.source} inline />}
				</span>
				<span className="truncate" style={{ fontSize: 11, color: "var(--sk-fg-3)" }}>
					<LooterSubline l={l} />
				</span>
			</div>

			<div style={{ position: "relative", height: 18 }}>
				<div
					style={{
						position: "absolute",
						inset: 0,
						background: "var(--sk-bg-2)",
						border: "1px solid var(--sk-line)",
						borderRadius: 2,
					}}
				/>
				<div
					style={{
						position: "absolute",
						left: 0,
						top: 0,
						bottom: 0,
						width: `${pct}%`,
						background: `color-mix(in oklab, ${goldFill} 35%, transparent)`,
						border: `1px solid color-mix(in oklab, ${goldFill} 55%, var(--sk-line-2))`,
						borderRadius: 2,
						transition: "width var(--sk-bar-dur) var(--sk-ease)",
					}}
				/>
			</div>

			<span className="sk-mono" style={{ textAlign: "right", fontSize: 12, color: "var(--sk-fg-2)" }}>
				{l.pickups}
			</span>
			<span className="sk-mono" style={{ textAlign: "right", fontSize: 14, fontWeight: 700, color: goldFill }}>
				{fmt(total)}
			</span>

			{/* Activity dot: green when fresh, fading to grey beyond 30s. */}
			<span
				title={l.lastPickupAt ? `Last pickup ${relTime(l.lastPickupAt)}` : "no pickups yet"}
				style={{
					width: 8,
					height: 8,
					borderRadius: 99,
					background: `color-mix(in oklab, var(--sk-card-heal, #5cf0a4) ${Math.round(activeRatio * 100)}%, var(--sk-fg-3))`,
					boxShadow: activeRatio > 0.5 ? `0 0 6px color-mix(in oklab, var(--sk-card-heal, #5cf0a4) ${Math.round(activeRatio * 60)}%, transparent)` : "none",
					justifySelf: "end",
				}}
			/>
		</div>
	);
}

function DrillView({
	name,
	entries,
	row,
}: {
	name: string;
	entries: LootEntry[];
	row?: LooterTotals;
}): React.ReactElement {
	const limit = 200;
	const shown = entries.slice(0, limit);
	const overflow = entries.length - shown.length;
	const total = row ? totalSilverOf(row) : 0;
	return (
		<div style={{ overflowY: "auto", flex: 1 }}>
			<div
				className="flex items-baseline"
				style={{
					gap: 14,
					padding: "16px 20px",
					borderBottom: "1px solid var(--sk-line)",
					background: row?.isLocal
						? "color-mix(in oklab, var(--sk-local) 6%, var(--sk-bg-2))"
						: "var(--sk-bg-inset)",
				}}
			>
				<span style={{ fontSize: 17, fontWeight: 600, color: row?.isLocal ? "var(--sk-local)" : "var(--sk-fg-0)" }}>
					{name}
				</span>
				<span className="sk-mono" style={{ fontSize: 13, fontWeight: 700, color: "var(--sk-card-fame, #d4af37)", marginLeft: "auto" }}>
					{fmt(total)} silver
				</span>
				<span className="sk-mono" style={{ fontSize: 11, color: "var(--sk-fg-3)" }}>
					{row?.pickups ?? entries.length} pickups
				</span>
			</div>
			{shown.length === 0 ? (
				<Empty>No pickups recorded for {name} this session.</Empty>
			) : (
				<>
					{shown.map((e, i) => (
						<div
							key={`${e.at}-${i}`}
							className="grid items-center"
							style={{
								gridTemplateColumns: "60px 14px minmax(0, 1fr) 50px 110px",
								gap: 10,
								padding: "8px 16px",
								borderBottom: "1px solid var(--sk-line)",
							}}
						>
							<span className="sk-mono" style={{ fontSize: 10, color: "var(--sk-fg-3)" }}>
								{relTime(e.at)}
							</span>
							{/* Icon placeholder — empty 14px square until items.bin icons land. */}
							<span style={{ width: 14, height: 14, borderRadius: 2, background: "var(--sk-bg-2)", border: "1px solid var(--sk-line)" }} />
							<div className="flex flex-col min-w-0" style={{ lineHeight: 1.2, gap: 1 }}>
								<span className="truncate" style={{ fontSize: 12, color: "var(--sk-fg-0)" }}>
									{e.isSilver ? "Silver pile" : (e.displayName || e.uniqueName || `#${e.itemIndex}`)}
								</span>
								{e.lootedFrom && (
									<span className="truncate" style={{ fontSize: 10, color: "var(--sk-fg-3)" }}>
										from {e.lootedFrom}
									</span>
								)}
							</div>
							<span className="sk-mono" style={{ textAlign: "right", fontSize: 11, color: "var(--sk-fg-1)" }}>
								{e.isSilver ? "—" : e.quantity}
							</span>
							<span
								className="sk-mono"
								style={{
									textAlign: "right",
									fontSize: 12,
									fontWeight: 600,
									color: e.isSilver
										? "var(--sk-card-silver, #d4af37)"
										: (e.silverValue && e.silverValue > 0 ? "var(--sk-card-fame, #d4af37)" : "var(--sk-fg-3)"),
								}}
							>
								{e.isSilver
									? fmt(toSilver(e.quantity))
									: (e.silverValue && e.silverValue > 0 ? fmt(e.silverValue) : "—")}
							</span>
						</div>
					))}
					{overflow > 0 && (
						<div style={{ padding: "10px 16px", color: "var(--sk-fg-3)", fontSize: 11, textAlign: "center" }}>
							{overflow} more entries… (truncated for sanity)
						</div>
					)}
				</>
			)}
		</div>
	);
}

function ItemsTab({ loot }: { loot: LootEntry[] }): React.ReactElement {
	if (loot.length === 0) {
		return <Empty>No loot events captured yet.</Empty>;
	}
	const ordered = [...loot].reverse();
	return (
		<div style={{ overflowY: "auto", flex: 1 }}>
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
						{l.isSilver ? "—" : l.quantity}
					</span>
					<span className="sk-mono" style={{ textAlign: "right", fontSize: 11, color: l.isSilver ? "var(--sk-card-silver)" : "var(--sk-fg-3)" }}>
						{l.isSilver ? fmt(toSilver(l.quantity)) : "—"}
					</span>
					<span className="sk-mono" style={{ textAlign: "right", fontSize: 11, color: l.silverValue && l.silverValue > 0 ? "var(--sk-card-fame)" : "var(--sk-fg-3)" }}>
						{l.silverValue && l.silverValue > 0 ? fmt(l.silverValue) : "—"}
					</span>
				</div>
			))}
		</div>
	);
}

function GrandTotal({ grandTotal }: { grandTotal: number }): React.ReactElement {
	return (
		<div
			className="flex items-baseline"
			style={{
				gap: 12,
				padding: "12px 16px",
				background: "var(--sk-bg-inset)",
				borderTop: "1px solid var(--sk-line-2)",
			}}
		>
			<span className="sk-upper" style={{ fontSize: 10, color: "var(--sk-fg-2)" }}>Party total</span>
			<span className="sk-mono" style={{ fontSize: 14, fontWeight: 700, color: "var(--sk-card-fame, #d4af37)", marginLeft: "auto" }}>
				{fmt(grandTotal)}
			</span>
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

function LooterSubline({ l }: { l: LooterTotals }): React.ReactElement {
	// Three-way fallback so the row never reads as a bare em-dash:
	//   1. Priced top item → "top: <name> · <value>"
	//   2. Item-but-unpriced fall-back via RecentItemName
	//   3. Only silver pickups → "silver only · <amount>"
	if (l.topItemName && (l.topItemValue ?? 0) > 0) {
		return (
			<>
				top: <span style={{ color: "var(--sk-fg-2)" }}>{l.topItemName}</span>
				{" · "}
				<span className="sk-mono">{fmt(l.topItemValue ?? 0)}</span>
			</>
		);
	}
	if (l.onlySilver) {
		return (
			<>
				silver only · <span className="sk-mono">{fmt(toSilver(l.silverPicked))} K</span>
			</>
		);
	}
	if (l.recentItemName) {
		return (
			<>
				recent: <span style={{ color: "var(--sk-fg-2)" }}>{l.recentItemName}</span>
				{" "}
				<span className="sk-mono" style={{ color: "var(--sk-fg-3)" }}>(unpriced)</span>
			</>
		);
	}
	return <span style={{ color: "var(--sk-fg-3)" }}>no pickups yet</span>;
}

// SourceBadge colors why a non-local row is on screen. "party" is
// orange to ride the damage accent (active fight context), "guild" is
// violet (membership context), "friend" is cyan (manual allowlist).
function SourceBadge({ source, inline = false }: { source?: string; inline?: boolean }): React.ReactElement | null {
	if (!source || source === "local") return null;
	const map: Record<string, { label: string; fg: string; bg: string; border: string }> = {
		party:  { label: "PARTY",  fg: "var(--sk-damage)",   bg: "color-mix(in oklab, var(--sk-damage) 12%, var(--sk-bg-2))",     border: "color-mix(in oklab, var(--sk-damage) 45%, var(--sk-line))" },
		guild:  { label: "GUILD",  fg: "#c08cff",            bg: "color-mix(in oklab, #c08cff 12%, var(--sk-bg-2))",              border: "color-mix(in oklab, #c08cff 45%, var(--sk-line))" },
		friend: { label: "FRIEND", fg: "#6fd8ff",            bg: "color-mix(in oklab, #6fd8ff 12%, var(--sk-bg-2))",              border: "color-mix(in oklab, #6fd8ff 45%, var(--sk-line))" },
	};
	const entry = map[source];
	if (!entry) return null;
	return (
		<span
			className="sk-upper"
			style={{
				fontSize: 8.5,
				letterSpacing: "0.08em",
				fontWeight: 700,
				padding: "1px 6px",
				borderRadius: 99,
				color: entry.fg,
				background: entry.bg,
				border: `1px solid ${entry.border}`,
				marginLeft: inline ? 6 : 0,
				lineHeight: 1.4,
			}}
			title={`Visible because: ${source}`}
		>
			{entry.label}
		</span>
	);
}

function Placeholder({ size }: { size: number }): React.ReactElement {
	return (
		<div
			title="IP unknown — agent hasn't seen this looter's equipment yet"
			style={{
				width: size,
				height: size,
				borderRadius: 4,
				background: "transparent",
				border: "1px solid var(--sk-line)",
				color: "var(--sk-fg-3)",
				display: "inline-flex",
				alignItems: "center",
				justifyContent: "center",
				fontFamily: "var(--sk-font-mono)",
				fontSize: Math.max(11, Math.floor(size * 0.45)),
				lineHeight: 1,
			}}
		>
			—
		</div>
	);
}

function Empty({ children }: { children: React.ReactNode }): React.ReactElement {
	return (
		<div className="flex items-center justify-center" style={{ padding: 56, color: "var(--sk-fg-2)", fontSize: 13, textAlign: "center", flex: 1 }}>
			{children}
		</div>
	);
}

function Toast({ text }: { text: string }): React.ReactElement {
	return (
		<div
			style={{
				position: "absolute",
				top: 12,
				right: 12,
				padding: "6px 12px",
				background: "var(--sk-bg-3)",
				border: "1px solid var(--sk-line-2)",
				borderRadius: 4,
				fontSize: 11,
				color: "var(--sk-fg-0)",
				fontWeight: 600,
				letterSpacing: "0.04em",
				boxShadow: "0 4px 12px rgba(0,0,0,0.4)",
				pointerEvents: "none",
				zIndex: 60,
			}}
		>
			{text}
		</div>
	);
}

function relTime(iso: string): string {
	const ms = Date.now() - +new Date(iso);
	if (ms < 5_000) return "just now";
	if (ms < 60_000) return `${Math.floor(ms / 1_000)}s ago`;
	if (ms < 3_600_000) return `${Math.floor(ms / 60_000)}m ago`;
	return `${Math.floor(ms / 3_600_000)}h ago`;
}

const ghostBtn: React.CSSProperties = {
	appearance: "none",
	border: "1px solid var(--sk-line)",
	background: "var(--sk-bg-2)",
	color: "var(--sk-fg-1)",
	padding: "4px 9px",
	borderRadius: 4,
	cursor: "pointer",
	fontSize: 10,
	letterSpacing: "0.06em",
	fontWeight: 600,
};
