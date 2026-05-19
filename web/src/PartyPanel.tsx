import type { PlayerSnapshot } from "./types.ts";
import { classAccent, roleKeyOf } from "./format.ts";

interface PartyPanelProps {
	players: PlayerSnapshot[];
	onClose: () => void;
}

// PartyPanel is the WoW-Details-style party-builder view. Lists each
// party member with their IP chip, class accent, name, equipped weapon
// + armor, and bound abilities (Q/W/E/Armor/Head/Shoes/Potion/Food).
//
// "Live" comes for free — every CharacterEquipmentChanged event Albion
// fires for a visible player triggers an agent-side re-snapshot. So if
// a partymate swaps spells / gear, this panel reflects within ~200ms
// of the swap.
export function PartyPanel({ players, onClose }: PartyPanelProps): React.ReactElement {
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
							Party · live loadout
						</span>
						<span className="sk-mono" style={{ color: "var(--sk-fg-3)", fontSize: 11 }}>
							{players.length} member{players.length === 1 ? "" : "s"}
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

				{players.length === 0 && (
					<div className="flex items-center justify-center" style={{ padding: 56, color: "var(--sk-fg-2)", fontSize: 13 }}>
						No party members tracked yet. Re-zone to pick up everyone visible.
					</div>
				)}

				<div style={{ overflowY: "auto" }}>
					{players.map((p) => (
						<PartyRow key={p.userGuid} p={p} />
					))}
				</div>
			</div>
		</div>
	);
}

function PartyRow({ p }: { p: PlayerSnapshot }): React.ReactElement {
	const role = roleKeyOf(p.role);
	const accent = classAccent(p.classCode, role);
	return (
		<div
			style={{
				padding: "12px 16px",
				borderBottom: "1px solid var(--sk-line)",
				background: "var(--sk-bg-1)",
				display: "grid",
				gridTemplateColumns: "auto 1fr",
				gap: 14,
			}}
		>
			<div className="flex flex-col items-center" style={{ gap: 4, minWidth: 80 }}>
				<div
					style={{
						minWidth: 56,
						padding: "4px 10px",
						borderRadius: 4,
						border: `1px solid color-mix(in oklab, ${accent} 55%, var(--sk-line-2))`,
						background: `color-mix(in oklab, ${accent} 12%, var(--sk-bg-2))`,
						color: accent,
						fontFamily: "var(--sk-font-mono)",
						fontWeight: 700,
						fontSize: 15,
						textAlign: "center",
						fontVariantNumeric: "tabular-nums",
					}}
					title={p.itemPower ? `IP ${p.itemPower}` : "IP unknown"}
				>
					{p.itemPower && p.itemPower > 0 ? p.itemPower : "—"}
				</div>
				<span className="sk-upper" style={{ fontSize: 8.5, color: "var(--sk-fg-3)" }}>IP</span>
			</div>
			<div className="flex flex-col" style={{ gap: 6, minWidth: 0 }}>
				<div className="flex items-baseline" style={{ gap: 10 }}>
					<span style={{ fontSize: 15, fontWeight: 600, color: "var(--sk-fg-0)" }}>
						{p.name || "(unknown)"}
					</span>
					{p.roleLabel && (
						<span className="sk-upper" style={{ fontSize: 10, color: accent, fontWeight: 600 }}>
							{p.roleLabel}
						</span>
					)}
				</div>
				{p.activeSpellSlots && p.activeSpellSlots.length > 0 && (
					<div className="flex flex-wrap items-center" style={{ gap: 4, marginTop: 2 }}>
						{p.activeSpellSlots.map((s) => (
							<span
								key={s.slot + s.name}
								title={`${s.slot}: ${s.name}`}
								style={{
									fontSize: 10,
									padding: "2px 6px",
									borderRadius: 3,
									background: "var(--sk-bg-2)",
									border: "1px solid var(--sk-line)",
									color: "var(--sk-fg-1)",
									display: "inline-flex",
									gap: 4,
									alignItems: "baseline",
								}}
							>
								<span className="sk-mono" style={{ color: accent, fontWeight: 700 }}>{s.slot}</span>
								<span>{s.name}</span>
							</span>
						))}
					</div>
				)}
				{p.equipmentSlots && p.equipmentSlots.length > 0 && (
					<div className="flex flex-wrap items-center" style={{ gap: 4 }}>
						{p.equipmentSlots
							.filter((s) => s.slot !== "Bag" && s.slot !== "Mount" && s.name)
							.map((s) => (
								<span
									key={s.slot + (s.name || "")}
									title={s.itemPower ? `${s.slot} · IP ${s.itemPower}` : s.slot}
									style={{
										fontSize: 10,
										padding: "2px 6px",
										borderRadius: 3,
										background: "color-mix(in oklab, var(--sk-bg-2) 70%, transparent)",
										color: "var(--sk-fg-2)",
										display: "inline-flex",
										gap: 4,
									}}
								>
									<span className="sk-mono" style={{ color: "var(--sk-fg-3)", fontSize: 9 }}>{s.slot}</span>
									<span>{s.name}</span>
								</span>
							))}
					</div>
				)}
			</div>
		</div>
	);
}
