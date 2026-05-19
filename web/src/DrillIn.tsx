import { useState } from "react";
import type { AssistBreakdown, PlayerSnapshot, SpellBreakdown, TargetBreakdown } from "./types.ts";
import { ClassChip } from "./ClassChip.tsx";
import { classAccent, fmt, fmtRate, prettySpell, roleKeyOf } from "./format.ts";

interface DrillInProps {
	player: PlayerSnapshot;
	onClose: () => void;
}

type DrillTab = "fight" | "session" | "targets" | "assists";

// Per-player drill-in. Tabs let you flip between:
//   - Fight     — spells used in the current fight (resets between pulls)
//   - Session   — spells cast since the last "New session" (cast count
//                 column matters here: "how many ult casts did I get off")
//   - Targets   — top recipients of this player's damage
//   - Assists   — debuff windows this player kept up + damage that flowed
//                 under them ("Level 2" attribution)
export function DrillIn({ player, onClose }: DrillInProps): React.ReactElement {
	const [tab, setTab] = useState<DrillTab>("fight");
	const roleKey = roleKeyOf(player.role);
	const accent = classAccent(player.classCode, roleKey);

	return (
		<div
			className="fixed inset-0 z-40 flex items-stretch justify-center p-2 md:p-6"
			style={{ background: "rgba(0,0,0,0.7)", backdropFilter: "blur(6px)" }}
			onClick={onClose}
		>
			<div
				className="w-full max-w-3xl flex flex-col"
				style={{
					background: "var(--sk-bg-0)",
					border: "1px solid var(--sk-line)",
					borderRadius: 8,
					overflow: "hidden",
				}}
				onClick={(e) => e.stopPropagation()}
			>
				<DrillHeader player={player} roleKey={roleKey} accent={accent} onClose={onClose} />
				<DrillTabs tab={tab} setTab={setTab} player={player} />
				{tab === "fight"   && <SpellTable spells={player.spells ?? []} emptyText="No abilities recorded in the current fight yet." accent={accent} />}
				{tab === "session" && <SpellTable spells={player.sessionSpells ?? []} emptyText="No spells cast yet this session." accent={accent} showCasts />}
				{tab === "targets" && <TargetTable targets={player.targets ?? []} accent={accent} />}
				{tab === "assists" && <AssistTable assists={player.assists ?? []} accent={accent} />}
			</div>
		</div>
	);
}

function DrillHeader({
	player, roleKey, accent, onClose,
}: {
	player: PlayerSnapshot;
	roleKey: ReturnType<typeof roleKeyOf>;
	accent: string;
	onClose: () => void;
}): React.ReactElement {
	const totalHits = (player.spells ?? []).reduce((s, x) => s + x.hits, 0);
	const totalCasts = (player.sessionSpells ?? []).reduce((s, x) => s + (x.casts ?? 0), 0);

	return (
		<div
			className="flex items-center justify-between px-4 py-3"
			style={{ borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-1)" }}
		>
			<div className="flex items-center" style={{ gap: 12 }}>
				<button
					onClick={onClose}
					aria-label="Close"
					style={{
						appearance: "none",
						border: "1px solid var(--sk-line)",
						background: "var(--sk-bg-2)",
						color: "var(--sk-fg-1)",
						width: 24,
						height: 24,
						borderRadius: 4,
						cursor: "pointer",
						fontSize: 12,
					}}
				>
					←
				</button>
				<ClassChip code={player.classCode || "—"} roleKey={roleKey} size={28} />
				<div>
					<div style={{ fontSize: 16, fontWeight: 600, color: "var(--sk-fg-0)" }}>
						{player.name || "(unknown)"}
					</div>
					{player.roleLabel && (
						<div className="sk-upper" style={{ color: accent, fontSize: 9.5 }}>
							{player.roleLabel}
						</div>
					)}
				</div>
			</div>
			<div className="flex" style={{ gap: 22 }}>
				<Stat label="Damage" value={fmt(player.currentDamage)} accent="var(--sk-damage)" />
				<Stat label="DPS"    value={fmtRate(player.currentDps)} />
				<Stat label="Hits"   value={totalHits.toString()} />
				<Stat label="Casts"  value={totalCasts.toString()} />
			</div>
		</div>
	);
}

function Stat({ label, value, accent }: { label: string; value: string; accent?: string }): React.ReactElement {
	return (
		<div className="flex flex-col items-end" style={{ gap: 2 }}>
			<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>{label}</span>
			<span className="sk-mono" style={{ fontSize: 15, fontWeight: 600, color: accent ?? "var(--sk-fg-0)" }}>
				{value}
			</span>
		</div>
	);
}

function DrillTabs({ tab, setTab, player }: { tab: DrillTab; setTab: (t: DrillTab) => void; player: PlayerSnapshot }): React.ReactElement {
	const items: Array<{ id: DrillTab; label: string; badge?: string }> = [
		{ id: "fight",   label: "Fight",   badge: (player.spells ?? []).length.toString() },
		{ id: "session", label: "Session", badge: (player.sessionSpells ?? []).length.toString() },
		{ id: "targets", label: "Targets", badge: (player.targets ?? []).length.toString() },
		{ id: "assists", label: "Assists", badge: (player.assists ?? []).length.toString() },
	];
	return (
		<div
			className="flex"
			style={{ padding: "0 16px", borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-1)" }}
		>
			{items.map((it) => {
				const active = it.id === tab;
				return (
					<button
						key={it.id}
						onClick={() => setTab(it.id)}
						style={{
							padding: "10px 12px",
							fontSize: 11.5,
							color: active ? "var(--sk-fg-0)" : "var(--sk-fg-2)",
							fontWeight: active ? 600 : 400,
							borderBottom: active ? "2px solid var(--sk-damage)" : "2px solid transparent",
							borderTop: "0",
							borderLeft: "0",
							borderRight: "0",
							background: "transparent",
							cursor: "pointer",
							display: "inline-flex",
							alignItems: "center",
							gap: 6,
						}}
					>
						{it.label}
						{it.badge !== undefined && it.badge !== "0" && (
							<span
								className="sk-mono"
								style={{ fontSize: 9, color: "var(--sk-fg-3)", padding: "1px 5px", border: "1px solid var(--sk-line)", borderRadius: 99 }}
							>
								{it.badge}
							</span>
						)}
					</button>
				);
			})}
		</div>
	);
}

function SpellTable({
	spells, emptyText, accent, showCasts,
}: {
	spells: SpellBreakdown[];
	emptyText: string;
	accent: string;
	showCasts?: boolean;
}): React.ReactElement {
	if (spells.length === 0) {
		return (
			<div className="flex-1 flex items-center justify-center" style={{ padding: 48, color: "var(--sk-fg-2)", fontSize: 13 }}>
				{emptyText}
			</div>
		);
	}
	const cols = showCasts
		? "1fr 60px 60px 60px 60px"
		: "1fr 70px 70px 70px";
	const max = spells.reduce((m, s) => Math.max(m, s.totalDamage), 0);
	return (
		<>
			<div
				className="grid items-center"
				style={{
					gridTemplateColumns: cols,
					gap: 12,
					padding: "10px 18px",
					fontSize: 10,
					color: "var(--sk-fg-3)",
					textTransform: "uppercase",
					letterSpacing: "0.08em",
					borderBottom: "1px solid var(--sk-line)",
				}}
			>
				<span>Ability</span>
				<span style={{ textAlign: "right" }}>Total</span>
				<span style={{ textAlign: "right" }}>Hits</span>
				{showCasts && <span style={{ textAlign: "right" }}>Casts</span>}
				<span style={{ textAlign: "right" }}>Max hit</span>
			</div>
			<div className="flex-1 overflow-auto">
				{spells.map((s) => {
					const pct = max > 0 ? (s.totalDamage / max) * 100 : 0;
					return (
						<div
							key={s.index}
							className="grid items-center"
							style={{
								gridTemplateColumns: cols,
								gap: 12,
								padding: "8px 18px",
								background: "var(--sk-bg-1)",
								borderBottom: "1px solid var(--sk-line)",
								height: 36,
							}}
						>
							<div className="relative flex items-center" style={{ height: 22 }}>
								<div style={{ position: "absolute", inset: 0, border: "1px solid var(--sk-line)", borderRadius: 2 }} />
								<div
									style={{
										position: "absolute", top: 0, bottom: 0, left: 0,
										width: `${pct}%`,
										background: `color-mix(in oklab, ${accent} 16%, transparent)`,
										border: `1px solid ${accent}`,
										borderRadius: 2,
									}}
								/>
								<span className="relative truncate" style={{ marginLeft: 8, zIndex: 1, fontSize: 12, fontWeight: 500, color: "var(--sk-fg-0)" }}>
									{prettySpell(s.name) || `#${s.index}`}
								</span>
							</div>
							<span className="sk-mono" style={{ textAlign: "right", fontSize: 12.5, fontWeight: 600, color: accent }}>
								{fmt(s.totalDamage)}
							</span>
							<span className="sk-mono" style={{ textAlign: "right", fontSize: 12, color: "var(--sk-fg-1)" }}>{s.hits}</span>
							{showCasts && (
								<span className="sk-mono" style={{ textAlign: "right", fontSize: 12, color: "var(--sk-fg-1)" }}>{s.casts ?? 0}</span>
							)}
							<span className="sk-mono" style={{ textAlign: "right", fontSize: 12, color: "var(--sk-fg-1)" }}>{fmt(s.maxHit)}</span>
						</div>
					);
				})}
			</div>
		</>
	);
}

function TargetTable({ targets, accent }: { targets: TargetBreakdown[]; accent: string }): React.ReactElement {
	if (targets.length === 0) {
		return (
			<div className="flex-1 flex items-center justify-center" style={{ padding: 48, color: "var(--sk-fg-2)", fontSize: 13 }}>
				No targets recorded — once you hit something this will populate.
			</div>
		);
	}
	const max = targets[0].damage;
	return (
		<>
			<div
				className="grid items-center"
				style={{
					gridTemplateColumns: "1fr 100px",
					gap: 12,
					padding: "10px 18px",
					fontSize: 10,
					color: "var(--sk-fg-3)",
					textTransform: "uppercase",
					letterSpacing: "0.08em",
					borderBottom: "1px solid var(--sk-line)",
				}}
			>
				<span>Target</span>
				<span style={{ textAlign: "right" }}>Damage</span>
			</div>
			<div className="flex-1 overflow-auto">
				{targets.map((t) => {
					const pct = max > 0 ? (t.damage / max) * 100 : 0;
					return (
						<div
							key={t.objectId}
							className="grid items-center"
							style={{
								gridTemplateColumns: "1fr 100px",
								gap: 12,
								padding: "8px 18px",
								background: "var(--sk-bg-1)",
								borderBottom: "1px solid var(--sk-line)",
								height: 36,
							}}
						>
							<div className="relative flex items-center" style={{ height: 22 }}>
								<div style={{ position: "absolute", inset: 0, border: "1px solid var(--sk-line)", borderRadius: 2 }} />
								<div
									style={{
										position: "absolute", top: 0, bottom: 0, left: 0,
										width: `${pct}%`,
										background: `color-mix(in oklab, ${accent} 16%, transparent)`,
										border: `1px solid ${accent}`,
										borderRadius: 2,
									}}
								/>
								<span className="relative truncate" style={{ marginLeft: 8, zIndex: 1, fontSize: 12, fontWeight: 500, color: "var(--sk-fg-0)" }}>
									{t.name || `#${t.objectId}`}
								</span>
							</div>
							<span className="sk-mono" style={{ textAlign: "right", fontSize: 12.5, fontWeight: 600, color: accent }}>
								{fmt(t.damage)}
							</span>
						</div>
					);
				})}
			</div>
		</>
	);
}

function AssistTable({ assists, accent }: { assists: AssistBreakdown[]; accent: string }): React.ReactElement {
	if (assists.length === 0) {
		return (
			<div className="flex-1 flex items-center justify-center" style={{ padding: "32px 48px", color: "var(--sk-fg-2)", fontSize: 13, textAlign: "center", lineHeight: 1.5 }}>
				No assist windows recorded.<br/>
				<span style={{ fontSize: 11, color: "var(--sk-fg-3)" }}>
					Land a debuff (Frazzle, Sunder, Cursed Beam, etc.) on a target —
					damage the party deals during its uptime will accrue here.
				</span>
			</div>
		);
	}
	const max = assists[0].damageDuring;
	return (
		<>
			<div
				className="grid items-center"
				style={{
					gridTemplateColumns: "1fr 80px 100px",
					gap: 12,
					padding: "10px 18px",
					fontSize: 10,
					color: "var(--sk-fg-3)",
					textTransform: "uppercase",
					letterSpacing: "0.08em",
					borderBottom: "1px solid var(--sk-line)",
				}}
			>
				<span>Debuff</span>
				<span style={{ textAlign: "right" }} title="How long the debuff was up">Uptime</span>
				<span style={{ textAlign: "right" }} title="Damage the party dealt to targets while the debuff was up">Dmg under</span>
			</div>
			<div className="flex-1 overflow-auto">
				{assists.map((a) => {
					const pct = max > 0 ? (a.damageDuring / max) * 100 : 0;
					const secs = a.uptimeMs / 1000;
					return (
						<div
							key={a.index}
							className="grid items-center"
							style={{
								gridTemplateColumns: "1fr 80px 100px",
								gap: 12,
								padding: "8px 18px",
								background: "var(--sk-bg-1)",
								borderBottom: "1px solid var(--sk-line)",
								height: 36,
							}}
						>
							<div className="relative flex items-center" style={{ height: 22 }}>
								<div style={{ position: "absolute", inset: 0, border: "1px solid var(--sk-line)", borderRadius: 2 }} />
								<div
									style={{
										position: "absolute", top: 0, bottom: 0, left: 0,
										width: `${pct}%`,
										background: `color-mix(in oklab, ${accent} 16%, transparent)`,
										border: `1px solid ${accent}`,
										borderRadius: 2,
									}}
								/>
								<span className="relative truncate" style={{ marginLeft: 8, zIndex: 1, fontSize: 12, fontWeight: 500, color: "var(--sk-fg-0)" }}>
									{prettySpell(a.name) || `#${a.index}`}
								</span>
							</div>
							<span className="sk-mono" style={{ textAlign: "right", fontSize: 12, color: "var(--sk-fg-1)" }}>
								{secs >= 60 ? `${Math.floor(secs / 60)}m ${Math.round(secs % 60)}s` : `${secs.toFixed(1)}s`}
							</span>
							<span className="sk-mono" style={{ textAlign: "right", fontSize: 12.5, fontWeight: 600, color: accent }}>
								{fmt(a.damageDuring)}
							</span>
						</div>
					);
				})}
			</div>
		</>
	);
}
