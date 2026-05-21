import { useMemo, useState } from "react";
import { createPortal } from "react-dom";
import type { Mode, PlayerSnapshot, Snapshot, SubMetric } from "./types.ts";
import type { Settings } from "./useSettings.ts";
import { classAccent, fmt, fmtRate, rankBy, roleKeyOf, type RoleKey } from "./format.ts";

interface MeterTableProps {
	snapshot: Snapshot | null;
	mode: Mode;
	settings: Settings;
	// sub locks this table to one metric+scope. The pane wrapper owns
	// which sub to show, so the same metric can render twice (Current
	// next to Session) as independent panes.
	sub: SubMetric;
	onDrillIn: (p: PlayerSnapshot) => void;
}

export function MeterTable({ snapshot, mode, settings, sub, onDrillIn }: MeterTableProps): React.ReactElement {
	const players = snapshot?.players ?? [];
	const [hovered, setHovered] = useState<PlayerSnapshot | null>(null);

	// primaryFor returns { cur, rate } for a player. `cur` drives the bar
	// fill + sort; `rate` is the right-column DPS/HPS (null = no rate).
	// Each pane shows ONE number now — the Current-vs-Session comparison
	// is done by opening two panes, not a dual number per row.
	const primaryFor = (p: PlayerSnapshot): { cur: number; rate: number | null } => {
		switch (sub) {
			case "damageCurrent": return { cur: p.currentDamage, rate: p.currentDps };
			case "damageTotal":   return { cur: p.overallDamage, rate: p.overallDps };
			case "healCurrent":   return { cur: p.currentHeal,   rate: p.currentHps };
			case "healTotal":     return { cur: p.overallHeal,   rate: p.overallHps };
			case "takenCurrent":  return { cur: p.currentTaken,  rate: null };
			case "takenTotal":    return { cur: p.overallTaken,  rate: null };
		}
	};

	// Sort by the metric, tie-breaking on userGuid so rows don't shuffle
	// each tick when several players tie (common out of combat). Memoised
	// on (generatedAt, sub, count) so unrelated re-renders skip the sort.
	const { sorted, max, partyTotal } = useMemo(() => {
		const arr = [...players].sort(rankBy((p) => primaryFor(p).cur));
		return {
			sorted: arr,
			max: arr[0] ? primaryFor(arr[0]).cur : 0,
			partyTotal: arr.reduce((s, p) => s + primaryFor(p).cur, 0),
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [snapshot?.generatedAt, sub, players.length]);

	if (sorted.length === 0) {
		return <EmptyState />;
	}

	const metricLabel = mode === "heal" ? "Healing" : mode === "taken" ? "Damage Taken" : "Damage";
	const rateLabel = mode === "heal" ? "HPS" : mode === "damage" ? "DPS" : "";

	return (
		<div className="relative h-full flex flex-col">
			{/* Column header */}
			<div
				className="grid items-center"
				style={{
					gridTemplateColumns: "28px minmax(160px, 220px) minmax(0, 1fr) 90px",
					gap: 12,
					padding: "10px 14px",
					borderBottom: "1px solid var(--sk-line)",
					background: "var(--sk-bg-inset)",
					fontSize: 10.5,
					textTransform: "uppercase",
					letterSpacing: "0.1em",
					color: "var(--sk-fg-2)",
					fontWeight: 600,
				}}
			>
				<span style={{ textAlign: "right", paddingRight: 4 }}>#</span>
				<span>Player</span>
				<span>{metricLabel}</span>
				<span style={{ textAlign: "right" }}>{rateLabel}</span>
			</div>

			{/* Rows */}
			<div className="flex-1 overflow-auto" style={{ display: "flex", flexDirection: "column", gap: 2, padding: "2px 0" }}>
				{sorted.map((p, i) => (
					<PlayerRow
						key={p.userGuid}
						player={p}
						rank={i + 1}
						mode={mode}
						max={max}
						partyTotal={partyTotal}
						primary={primaryFor(p)}
						settings={settings}
						onHover={setHovered}
						onDrillIn={onDrillIn}
					/>
				))}
			</div>

			{hovered && <RowTooltip player={hovered} />}
		</div>
	);
}

interface PlayerRowProps {
	player: PlayerSnapshot;
	rank: number;
	mode: Mode;
	max: number;
	partyTotal: number;
	primary: { cur: number; rate: number | null };
	settings: Settings;
	onHover: (p: PlayerSnapshot | null) => void;
	onDrillIn: (p: PlayerSnapshot) => void;
}

function PlayerRow({ player, rank, mode, max, partyTotal, primary, settings, onHover, onDrillIn }: PlayerRowProps): React.ReactElement {
	const isLocal = player.isLocal ?? false;
	const roleKey: RoleKey = roleKeyOf(player.role);
	// Bar / DPS / chip share a per-weapon accent. Holy gold ≠ Nature
	// green ≠ Frost cyan ≠ Daggers red — see classAccent() in format.ts.
	const barColor = classAccent(player.classCode, roleKey);
	const barTint = `color-mix(in oklab, ${barColor} 16%, transparent)`;
	const pct = max > 0 ? Math.min(100, (primary.cur / max) * 100) : 0;
	const rowH = settings.density;

	const barFill = settings.barStyle === "solid" ? barColor : barTint;
	const barBorder = settings.barStyle === "outline" ? `1px solid ${barColor}` : "1px solid transparent";

	return (
		<div
			onMouseEnter={() => onHover(player)}
			onMouseLeave={() => onHover(null)}
			onClick={() => onDrillIn(player)}
			style={{
				position: "relative",
				height: rowH,
				display: "grid",
				gridTemplateColumns: "28px minmax(160px, 220px) minmax(0, 1fr) 90px",
				alignItems: "center",
				gap: 12,
				padding: `0 14px 0 12px`,
				background: isLocal ? "color-mix(in oklab, var(--sk-local) 5%, var(--sk-bg-1))" : "var(--sk-bg-1)",
				borderLeft: isLocal ? "2px solid var(--sk-local)" : "2px solid transparent",
				transition: "background 200ms var(--sk-ease)",
				cursor: "pointer",
			}}
		>
			{/* Rank */}
			<span
				className="sk-mono"
				style={{
					fontSize: 13,
					textAlign: "right",
					paddingRight: 4,
					color: rank === 1 ? "var(--sk-damage)" : "var(--sk-fg-1)",
					fontWeight: rank === 1 ? 600 : 500,
				}}
			>
				{String(rank).padStart(2, "0")}
			</span>

			{/* Name + role label (accent stripe on the left edge carries the
			    class colour now that the IP chip is gone). */}
			<div className="flex items-center min-w-0" style={{ gap: 10 }}>
				<div className="flex flex-col min-w-0" style={{ lineHeight: 1.15, gap: 2 }}>
					<span
						className="truncate"
						style={{
							fontSize: 14.5,
							fontWeight: 600,
							color: "var(--sk-fg-0)",
							letterSpacing: "-0.005em",
						}}
					>
						{player.name || `${player.userGuid.slice(0, 8)}…`}
					</span>
					{player.roleLabel && (
						<span
							className="sk-upper truncate"
							style={{
								fontSize: 10,
								color: barColor,
								fontWeight: 600,
							}}
						>
							{player.roleLabel}
						</span>
					)}
				</div>
			</div>

			{/* Bar + dual value. In the Healing pane, non-healer roles
			    have nothing to plot — render a muted dash instead of a
			    zero-width bar so the eye doesn't waste time on it. */}
			{mode === "heal" && !healsForRole(roleKey) ? (
				<div className="flex items-center" style={{ marginLeft: 8, color: "var(--sk-fg-3)", fontSize: 12 }}>—</div>
			) : (
			<div className="relative h-full flex items-center">
				<div
					style={{
						position: "absolute",
						inset: "auto 0",
						top: "50%",
						transform: "translateY(-50%)",
						height: Math.max(14, rowH - 12),
						background: "transparent",
						borderRadius: 2,
					}}
				>
					<div
						style={{
							position: "absolute",
							inset: 0,
							width: `${pct}%`,
							background: barFill,
							border: barBorder,
							borderRadius: 2,
							transition: "width var(--sk-bar-dur) var(--sk-ease)",
						}}
					/>
					{[0.25, 0.5, 0.75].map((p) => (
						<div
							key={p}
							style={{
								position: "absolute",
								left: `${p * 100}%`,
								top: 0,
								bottom: 0,
								width: 1,
								background: "var(--sk-line)",
							}}
						/>
					))}
				</div>
				<div className="relative z-10 flex items-baseline" style={{ marginLeft: 8, gap: 8 }}>
					<span
						className="sk-mono"
						style={{
							fontSize: 15,
							fontWeight: 600,
							color: "var(--sk-fg-0)",
							letterSpacing: "-0.02em",
						}}
						title={isLocal ? "Local player" : undefined}
					>
						{fmt(primary.cur)}
					</span>
					<span className="sk-mono" style={{ fontSize: 11.5, color: "var(--sk-fg-2)" }}>
						({sharePct(primary.cur, partyTotal)}%)
					</span>
				</div>
			</div>
			)}

			{/* DPS / HPS — tinted to the player's class accent so the rate
			    ties to the bar above it. Falls back to dim "—" for roles
			    that don't produce the active metric (HPS on a DPS row). */}
			<div className="flex flex-col items-end" style={{ lineHeight: 1, gap: 2 }}>
				{(() => {
					const showRate = primary.rate != null && !(mode === "heal" && !healsForRole(roleKey));
					return (
						<>
							<span
								className="sk-mono"
								style={{
									fontSize: 16,
									fontWeight: 600,
									color: showRate ? barColor : "var(--sk-fg-3)",
									letterSpacing: "-0.02em",
								}}
							>
								{showRate ? fmtRate(primary.rate!) : "—"}
							</span>
							<span className="sk-upper" style={{ color: "var(--sk-fg-2)", fontSize: 10, fontWeight: 600 }}>
								{mode === "heal" ? "hps" : mode === "damage" ? "dps" : ""}
							</span>
						</>
					);
				})()}
			</div>

		</div>
	);
}

// healsForRole returns true when a role meaningfully produces heals. Pure
// DPS / tanks / control don't, so their heal/overheal cells should read
// blank to keep the row scannable.
function healsForRole(role: RoleKey): boolean {
	return role === "healer" || role === "support";
}


function RowTooltip({ player }: { player: PlayerSnapshot }): React.ReactElement {
	const roleKey = roleKeyOf(player.role);
	const roleColor = `var(--sk-role-${roleKey})`;
	const lines: Array<{ label: string; value: string; tint: string }> = [
		{ label: "Damage done",   value: fmt(player.overallDamage), tint: "var(--sk-damage)" },
		{ label: "Healing done",  value: fmt(player.overallHeal),   tint: "var(--sk-heal)" },
		{ label: "Damage taken",  value: fmt(player.overallTaken),  tint: "var(--sk-taken)" },
		{ label: "DPS · current", value: fmtRate(player.currentDps),tint: "var(--sk-fg-0)" },
	];
	// Portal'd to document.body so it can render over the multi-pane
	// overflow-hidden containers without getting clipped at the pane edge.
	return createPortal(
		<div
			style={{
				position: "fixed",
				right: 24,
				bottom: 56,
				width: 240,
				background: "var(--sk-bg-2)",
				border: "1px solid var(--sk-line-2)",
				borderRadius: 6,
				padding: 12,
				boxShadow: "0 12px 32px -8px rgba(0,0,0,0.6), 0 2px 0 var(--sk-line)",
				zIndex: 50,
				pointerEvents: "none",
			}}
		>
			<div
				className="flex items-center"
				style={{ gap: 8, marginBottom: 10, paddingBottom: 8, borderBottom: "1px solid var(--sk-line)" }}
			>
				<div className="min-w-0">
					<div
						style={{
							fontSize: 13,
							fontWeight: 600,
							color: "var(--sk-fg-0)",
						}}
					>
						{player.name || "(unknown)"}
					</div>
					{player.roleLabel && (
						<div className="sk-upper" style={{ fontSize: 9.5, color: roleColor }}>
							{player.roleLabel}
						</div>
					)}
				</div>
			</div>
			<div className="flex flex-col" style={{ gap: 6 }}>
				{lines.map((l) => (
					<div key={l.label} className="flex items-baseline justify-between" style={{ gap: 8 }}>
						<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>{l.label}</span>
						<span className="sk-mono" style={{ fontSize: 12, color: l.tint, fontWeight: 600 }}>{l.value}</span>
					</div>
				))}
			</div>
			<div
				className="flex justify-between"
				style={{ marginTop: 10, paddingTop: 8, borderTop: "1px solid var(--sk-line)" }}
			>
				<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>Click row → drill in</span>
			</div>
		</div>,
		document.body,
	);
}

function sharePct(value: number, total: number): string {
	if (total <= 0 || value <= 0) return "0";
	const p = (value / total) * 100;
	if (p >= 99.5) return "100";
	if (p >= 10) return p.toFixed(0);
	return p.toFixed(1);
}

function EmptyState(): React.ReactElement {
	return (
		<div className="flex items-center justify-center" style={{ minHeight: 240, color: "var(--sk-fg-2)" }}>
			<div className="text-center" style={{ padding: 32 }}>
				<div
					className="sk-upper"
					style={{ color: "var(--sk-fg-3)", marginBottom: 8 }}
				>
					Waiting for combat
				</div>
				<div style={{ fontSize: 13, color: "var(--sk-fg-1)" }}>
					Re-zone in Albion (walk through any portal) to populate the meter.
				</div>
			</div>
		</div>
	);
}
