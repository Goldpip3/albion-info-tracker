import { useMemo } from "react";
import type { PlayerSnapshot, VisiblePlayer } from "./types.ts";
import { classAccent, rankBy, roleKeyOf } from "./format.ts";
import { EmptyState } from "./EmptyState.tsx";

interface PartyBodyProps {
	players: PlayerSnapshot[];
	generatedAt?: string;
	visiblePlayers?: VisiblePlayer[];
	sendCommand?: (action: string, arg?: string) => boolean;
}

// PartyBody is the shared body for the Party view. Rows are sorted by
// IP desc with userGuid as a stable tiebreak (Go's randomised map
// iteration would otherwise shuffle rows each tick). Below the roster
// it lists tracked-but-unpartied players as one-tap "add to party"
// candidates — the deterministic fix for mixed-guild parties the agent
// never saw form. Adds/removes route through the agent command channel
// and persist to disk, so they survive an agent restart.
export function PartyBody({ players, generatedAt, visiblePlayers = [], sendCommand }: PartyBodyProps): React.ReactElement {
	const sorted = useMemo(() => {
		return [...players].sort(rankBy((p) => p.itemPower ?? 0));
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [generatedAt, players.length]);

	const canEdit = !!sendCommand;
	const onRemove = (guid: string): void => { sendCommand?.("removePartyMember", guid); };
	const onAdd = (guid: string): void => { sendCommand?.("addPartyMember", guid); };
	const onClear = (): void => {
		if (window.confirm("Clear the whole party roster? Use this if you're solo but a stale group is showing.")) {
			sendCommand?.("clearParty");
		}
	};

	const localGuildKnown = players.some((p) => p.isLocal && !!p.guild);
	const addList = (
		<AddInRange players={visiblePlayers} localGuildKnown={localGuildKnown} onAdd={canEdit ? onAdd : undefined} />
	);

	// Header bar with a Clear-party action — the escape hatch for a
	// stale roster restored from disk while you're actually solo.
	const bar = (canEdit && sorted.length > 0) ? (
		<div
			className="flex items-center justify-between"
			style={{ padding: "8px 16px", borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-inset)" }}
		>
			<span className="sk-upper" style={{ fontSize: 10, color: "var(--sk-fg-3)", letterSpacing: "0.1em", fontWeight: 700 }}>
				{sorted.length} in party
			</span>
			<button
				onClick={onClear}
				className="sk-upper"
				style={{
					appearance: "none",
					border: "1px solid color-mix(in oklab, var(--sk-err) 45%, var(--sk-line))",
					background: "color-mix(in oklab, var(--sk-err) 10%, var(--sk-bg-2))",
					color: "var(--sk-err)",
					padding: "3px 10px", borderRadius: 4, cursor: "pointer",
					fontSize: 9.5, fontWeight: 700, letterSpacing: "0.08em",
				}}
			>
				Clear party
			</button>
		</div>
	) : null;

	if (sorted.length === 0) {
		return (
			<div style={{ overflowY: "auto", flex: 1 }}>
				<EmptyState
					title="No party members tracked"
					body="Albion only announces party members at the moment they join — if the agent started after you grouped, add your teammates by hand below. They'll persist across restarts."
				/>
				{addList}
			</div>
		);
	}

	return (
		<div style={{ overflowY: "auto", flex: 1 }}>
			{bar}
			{sorted.map((p) => (
				<PartyRow key={p.userGuid} p={p} onRemove={canEdit ? onRemove : undefined} />
			))}
			{addList}
		</div>
	);
}

// AddInRange lists tracked players who aren't in the party as one-tap
// add candidates. Guildmates (sorted to the top by the agent) get a
// "Your guild" section header + an "Add all guild" button so a big guild
// group can be assembled in one tap. Hidden entirely when empty.
function AddInRange({ players, localGuildKnown, onAdd }: { players: VisiblePlayer[]; localGuildKnown: boolean; onAdd?: (guid: string) => void }): React.ReactElement | null {
	if (players.length === 0) return null;
	const guildCount = players.filter((v) => v.sameGuild).length;
	const onAddAllGuild = (): void => {
		if (!onAdd) return;
		for (const v of players) if (v.sameGuild) onAdd(v.userGuid);
	};
	let lastSection = "";
	return (
		<div style={{ borderTop: "1px solid var(--sk-line-2)" }}>
			<div
				className="flex items-center justify-between"
				style={{ padding: "10px 16px 8px", background: "var(--sk-bg-inset)" }}
			>
				<span className="sk-upper" style={{ fontSize: 10, color: "var(--sk-fg-3)", letterSpacing: "0.1em", fontWeight: 700 }}>
					Add players in range · {players.length}
				</span>
				{onAdd && guildCount > 0 && (
					<button
						onClick={onAddAllGuild}
						className="sk-upper"
						style={{
							appearance: "none",
							border: "1px solid color-mix(in oklab, var(--sk-ok) 45%, var(--sk-line))",
							background: "color-mix(in oklab, var(--sk-ok) 12%, var(--sk-bg-2))",
							color: "var(--sk-ok)",
							padding: "3px 10px", borderRadius: 4, cursor: "pointer",
							fontSize: 9.5, fontWeight: 700, letterSpacing: "0.06em", whiteSpace: "nowrap",
						}}
					>
						+ Add all guild · {guildCount}
					</button>
				)}
			</div>
			{guildCount === 0 && !localGuildKnown && (
				<div style={{ padding: "0 16px 10px", fontSize: 11, lineHeight: 1.4, color: "var(--sk-fg-3)", background: "var(--sk-bg-inset)" }}>
					Your guild isn't detected yet — ride through a zone so the agent learns it, then guildmates in range sort to the top here.
				</div>
			)}
			{players.map((v) => {
				const section = v.sameGuild ? "guild" : "other";
				const showHeader = section !== lastSection;
				lastSection = section;
				const role = roleKeyOf(v.role);
				const accent = classAccent(v.classCode, role);
				return (
					<div key={v.userGuid}>
						{showHeader && (
							<div
								className="sk-upper"
								style={{
									padding: "8px 16px 5px",
									fontSize: 9, letterSpacing: "0.1em", fontWeight: 700,
									color: section === "guild" ? "var(--sk-ok)" : "var(--sk-fg-3)",
									background: "var(--sk-bg-2)",
								}}
							>
								{section === "guild" ? `Your guild · ${guildCount}` : "Other players in range"}
							</div>
						)}
						<div
							className="flex items-center"
							style={{ gap: 12, padding: "9px 16px", borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-1)" }}
						>
							<span
								style={{
									minWidth: 40, padding: "3px 8px", borderRadius: 4, textAlign: "center",
									border: `1px solid color-mix(in oklab, ${accent} 55%, var(--sk-line-2))`,
									background: `color-mix(in oklab, ${accent} 12%, var(--sk-bg-2))`,
									color: accent, fontFamily: "var(--sk-font-mono)", fontWeight: 700, fontSize: 12,
									fontVariantNumeric: "tabular-nums",
								}}
								title={v.itemPower ? `IP ${v.itemPower}` : "IP unknown"}
							>
								{v.itemPower && v.itemPower > 0 ? v.itemPower : "—"}
							</span>
							<div className="flex flex-col" style={{ minWidth: 0, flex: 1, gap: 1 }}>
								<span style={{ fontSize: 13.5, fontWeight: 600, color: "var(--sk-fg-0)" }}>{v.name}</span>
								<span className="flex items-center" style={{ gap: 8 }}>
									{v.roleLabel && (
										<span className="sk-upper" style={{ fontSize: 9, color: accent, fontWeight: 700 }}>{v.roleLabel}</span>
									)}
									{v.guild && (
										<span style={{ fontSize: 10, color: v.sameGuild ? "var(--sk-ok)" : "var(--sk-fg-3)" }}>{v.guild}</span>
									)}
								</span>
							</div>
							{onAdd && (
								<button
									onClick={() => onAdd(v.userGuid)}
									className="sk-upper"
									style={{
										appearance: "none",
										border: "1px solid color-mix(in oklab, var(--sk-ok) 45%, var(--sk-line))",
										background: "color-mix(in oklab, var(--sk-ok) 12%, var(--sk-bg-2))",
										color: "var(--sk-ok)",
										padding: "4px 12px", borderRadius: 4, cursor: "pointer",
										fontSize: 10, fontWeight: 700, letterSpacing: "0.08em",
									}}
								>
									+ Add
								</button>
							)}
						</div>
					</div>
				);
			})}
		</div>
	);
}

function PartyRow({ p, onRemove }: { p: PlayerSnapshot; onRemove?: (guid: string) => void }): React.ReactElement {
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
					{onRemove && !p.isLocal && (
						<button
							onClick={() => onRemove(p.userGuid)}
							title="Remove from party"
							className="sk-upper"
							style={{
								marginLeft: "auto",
								appearance: "none",
								border: "1px solid var(--sk-line)",
								background: "var(--sk-bg-2)",
								color: "var(--sk-fg-3)",
								padding: "2px 8px", borderRadius: 4, cursor: "pointer",
								fontSize: 9, fontWeight: 700, letterSpacing: "0.08em",
							}}
						>
							Remove
						</button>
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
