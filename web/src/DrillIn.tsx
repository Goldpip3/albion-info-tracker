import type { PlayerSnapshot, SpellBreakdown } from "./types.ts";
import { ClassChip } from "./ClassChip.tsx";
import { fmt, fmtRate, prettySpell, roleKeyOf } from "./format.ts";

interface DrillInProps {
	player: PlayerSnapshot;
	onClose: () => void;
}

// Per-player drill-in. Matches details.jsx's PlayerDrillIn from the design:
// header with class chip, name + role line, big stats, tab row, then a
// table of abilities with damage-bar fills.
export function DrillIn({ player, onClose }: DrillInProps): React.ReactElement {
	const spells = player.spells ?? [];
	const max = spells[0]?.totalDamage ?? 0;
	const roleKey = roleKeyOf(player.role);

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
				<DrillHeader player={player} roleKey={roleKey} onClose={onClose} />
				<DrillTabs />
				<DrillColumnHeader />
				<DrillRows spells={spells} max={max} />
			</div>
		</div>
	);
}

function DrillHeader({
	player,
	roleKey,
	onClose,
}: {
	player: PlayerSnapshot;
	roleKey: ReturnType<typeof roleKeyOf>;
	onClose: () => void;
}): React.ReactElement {
	const totalHits = (player.spells ?? []).reduce((s, x) => s + x.hits, 0);
	const maxHit = (player.spells ?? []).reduce((m, s) => Math.max(m, s.maxHit), 0);

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
					<div
						style={{
							fontSize: 16,
							fontWeight: 600,
							color: player.isLocal ? "var(--sk-local)" : "var(--sk-fg-0)",
						}}
					>
						{player.name || "(unknown)"}
					</div>
					{player.roleLabel && (
						<div
							className="sk-upper"
							style={{ color: `var(--sk-role-${roleKey})`, fontSize: 9.5 }}
						>
							{player.roleLabel}
						</div>
					)}
				</div>
			</div>
			<div className="flex" style={{ gap: 22 }}>
				<Stat label="Damage" value={fmt(player.currentDamage)} accent="var(--sk-damage)" />
				<Stat label="DPS"    value={fmtRate(player.currentDps)} />
				<Stat label="Hits"   value={totalHits.toString()} />
				<Stat label="Max"    value={fmt(maxHit)} />
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

function DrillTabs(): React.ReactElement {
	const tabs = ["Abilities", "Targets", "Taken", "Healing received", "Timeline"];
	return (
		<div
			className="flex"
			style={{ padding: "0 16px", borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-1)" }}
		>
			{tabs.map((t, i) => (
				<span
					key={t}
					style={{
						padding: "10px 12px",
						fontSize: 11.5,
						color: i === 0 ? "var(--sk-fg-0)" : "var(--sk-fg-2)",
						fontWeight: i === 0 ? 600 : 400,
						borderBottom: i === 0 ? "2px solid var(--sk-damage)" : "2px solid transparent",
						cursor: i === 0 ? "default" : "not-allowed",
						opacity: i === 0 ? 1 : 0.6,
					}}
					title={i === 0 ? "" : "Coming soon"}
				>
					{t}
				</span>
			))}
		</div>
	);
}

function DrillColumnHeader(): React.ReactElement {
	return (
		<div
			className="grid items-center"
			style={{
				gridTemplateColumns: "1fr 70px 70px 70px",
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
			<span style={{ textAlign: "right" }}>Max hit</span>
		</div>
	);
}

function DrillRows({ spells, max }: { spells: SpellBreakdown[]; max: number }): React.ReactElement {
	if (spells.length === 0) {
		return (
			<div
				className="flex-1 flex items-center justify-center"
				style={{ padding: 48, color: "var(--sk-fg-2)", fontSize: 13 }}
			>
				No abilities recorded in the current fight yet.
			</div>
		);
	}
	return (
		<div className="flex-1 overflow-auto">
			{spells.map((s) => {
				const pct = max > 0 ? (s.totalDamage / max) * 100 : 0;
				return (
					<div
						key={s.index}
						className="grid items-center"
						style={{
							gridTemplateColumns: "1fr 70px 70px 70px",
							gap: 12,
							padding: "8px 18px",
							background: "var(--sk-bg-1)",
							borderBottom: "1px solid var(--sk-line)",
							height: 36,
						}}
					>
						<div className="relative flex items-center" style={{ height: 22 }}>
							<div
								style={{
									position: "absolute",
									inset: 0,
									border: "1px solid var(--sk-line)",
									borderRadius: 2,
								}}
							/>
							<div
								style={{
									position: "absolute",
									top: 0,
									bottom: 0,
									left: 0,
									width: `${pct}%`,
									background: "var(--sk-damage-tint)",
									border: "1px solid var(--sk-damage)",
									borderRadius: 2,
								}}
							/>
							<span
								className="relative truncate"
								style={{ marginLeft: 8, zIndex: 1, fontSize: 12, fontWeight: 500, color: "var(--sk-fg-0)" }}
							>
								{prettySpell(s.name) || `#${s.index}`}
							</span>
						</div>
						<span className="sk-mono" style={{ textAlign: "right", fontSize: 12.5, fontWeight: 600, color: "var(--sk-damage)" }}>
							{fmt(s.totalDamage)}
						</span>
						<span className="sk-mono" style={{ textAlign: "right", fontSize: 12, color: "var(--sk-fg-1)" }}>
							{s.hits}
						</span>
						<span className="sk-mono" style={{ textAlign: "right", fontSize: 12, color: "var(--sk-fg-1)" }}>
							{fmt(s.maxHit)}
						</span>
					</div>
				);
			})}
		</div>
	);
}
