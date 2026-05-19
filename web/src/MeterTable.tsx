import { useState } from "react";
import { createPortal } from "react-dom";
import type { Mode, PlayerSnapshot, Snapshot } from "./types.ts";
import type { Settings } from "./useSettings.ts";
import { ClassChip } from "./ClassChip.tsx";
import { classAccent, fmt, fmtRate, roleKeyOf, type RoleKey } from "./format.ts";

interface MeterTableProps {
	snapshot: Snapshot | null;
	mode: Mode;
	settings: Settings;
	onDrillIn: (p: PlayerSnapshot) => void;
}

export function MeterTable({ snapshot, mode, settings, onDrillIn }: MeterTableProps): React.ReactElement {
	const players = snapshot?.players ?? [];
	const [hovered, setHovered] = useState<PlayerSnapshot | null>(null);

	const primaryFor = (p: PlayerSnapshot): { cur: number; ovr: number; rate: number | null } => {
		switch (mode) {
			case "damage":
				return { cur: p.currentDamage, ovr: p.overallDamage, rate: p.currentDps };
			case "heal":
				return { cur: p.currentHeal, ovr: p.overallHeal, rate: p.currentHps };
			case "taken":
				return { cur: p.currentTaken, ovr: p.overallTaken, rate: null };
			case "mechanics":
				return { cur: p.currentDamage, ovr: p.overallDamage, rate: null };
		}
	};

	// Sort by the active metric, tie-breaking on userGuid so rows don't
	// shuffle each tick when several players share the same value (very
	// common out-of-combat where everyone's at 0). The agent rebuilds the
	// player list from a Go map each snapshot and map iteration order is
	// randomized, so without an explicit tie-break the input order leaks
	// through stable sort and players visibly swap places.
	let sorted = [...players].sort((a, b) => {
		const d = primaryFor(b).cur - primaryFor(a).cur;
		if (d !== 0) return d;
		return a.userGuid < b.userGuid ? -1 : a.userGuid > b.userGuid ? 1 : 0;
	});

	if (settings.pinLocal) {
		const local = sorted.find((p) => p.isLocal);
		if (local) sorted = [local, ...sorted.filter((p) => p !== local)];
	}

	const max = sorted[0] ? primaryFor(sorted[0]).cur : 0;
	// Party total for the active metric — drives the per-row share (%).
	const partyTotal = sorted.reduce((s, p) => s + primaryFor(p).cur, 0);

	if (sorted.length === 0) {
		return (
			<EmptyState />
		);
	}

	const headerLabel: Record<Mode, string> = {
		damage: "Damage · Current / Session",
		heal: "Healing · Current / Session",
		taken: "Damage Taken · Current / Session",
		mechanics: "Mechanics",
	};
	const rateHeader: Record<Mode, string> = {
		damage: "DPS",
		heal: "HPS",
		taken: "",
		mechanics: "",
	};

	return (
		<div className="relative h-full flex flex-col">
			{/* Column header */}
			<div
				className="grid items-center"
				style={{
					gridTemplateColumns: "28px 200px 1fr 90px 70px 70px 70px",
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
				<span>{headerLabel[mode]}</span>
				<span style={{ textAlign: "right" }}>{rateHeader[mode]}</span>
				<span style={{ textAlign: "right" }} title="Damage taken">↓ Taken</span>
				<span style={{ textAlign: "right" }} title="Healing done">+ Heal</span>
				<span style={{ textAlign: "right" }} title="Overheal — heal that hit max-HP targets">~ Over</span>
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
	primary: { cur: number; ovr: number; rate: number | null };
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
				gridTemplateColumns: "28px 200px 1fr 90px 70px 70px 70px",
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

			{/* Class chip + name + role label */}
			<div className="flex items-center min-w-0" style={{ gap: 10 }}>
				<ClassChip code={player.classCode || "—"} roleKey={roleKey} size={Math.min(28, Math.max(22, rowH - 14))} />
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

			{/* Bar + dual value */}
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
					<span className="sk-mono" style={{ fontSize: 11.5, color: "var(--sk-fg-1)" }}>
						↳ {fmt(primary.ovr)} session
					</span>
				</div>
			</div>

			{/* DPS / HPS — tinted to the player's class accent so the rate
			    ties to the bar above it (matches the design's per-row colour
			    coding). Falls back to the muted tone for Taken (no rate). */}
			<div className="flex flex-col items-end" style={{ lineHeight: 1, gap: 2 }}>
				<span
					className="sk-mono"
					style={{
						fontSize: 16,
						fontWeight: 600,
						color: primary.rate != null ? barColor : "var(--sk-fg-3)",
						letterSpacing: "-0.02em",
					}}
				>
					{primary.rate != null ? fmtRate(primary.rate) : "—"}
				</span>
				<span className="sk-upper" style={{ color: "var(--sk-fg-2)", fontSize: 10, fontWeight: 600 }}>
					{mode === "heal" ? "hps" : mode === "damage" ? "dps" : ""}
				</span>
			</div>

			{/* Taken chip */}
			<MiniChip color="var(--sk-taken)" value={fmt(player.currentTaken)} glyph="↓" />
			{/* Healing chip */}
			<MiniChip color="var(--sk-heal)" value={fmt(player.currentHeal)} glyph="+" />
			{/* Overheal chip — dim grey since it's a "waste" stat */}
			<MiniChip color="var(--sk-fg-3)" value={fmt(player.overheal ?? 0)} glyph="~" />
		</div>
	);
}

function MiniChip({ color, value, glyph }: { color: string; value: string; glyph: string }): React.ReactElement {
	return (
		<span
			className="sk-mono inline-flex items-center"
			style={{
				gap: 4,
				fontSize: 12,
				color: "var(--sk-fg-0)",
				fontWeight: 500,
				justifyContent: "flex-end",
				textAlign: "right",
			}}
		>
			<span style={{ color, fontWeight: 700 }}>{glyph}</span>
			{value}
		</span>
	);
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
				<ClassChip code={player.classCode || "—"} roleKey={roleKey} size={22} />
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
