import { useMemo, useState } from "react";
import type { RosterEntry } from "./types.ts";
import { classAccent, roleKeyOf } from "./format.ts";

interface PlayerPickerProps {
	roster: RosterEntry[];
	onClose: () => void;
	sendCommand: (action: string, arg?: string) => boolean;
}

// PlayerPicker is the Meter's "who do I track" menu. It lists every named
// player the agent can see (guildmates sorted to the top by the agent),
// and lets the user click players in and out of the tracked party. Each
// toggle routes through addPartyMember / removePartyMember; the next
// snapshot reflects the new state. The local player is always tracked and
// shown pinned at the top.
export function PlayerPicker({ roster, onClose, sendCommand }: PlayerPickerProps): React.ReactElement {
	const [query, setQuery] = useState("");

	const filtered = useMemo(() => {
		const q = query.trim().toLowerCase();
		if (!q) return roster;
		return roster.filter(
			(r) => r.name.toLowerCase().includes(q) || (r.guild ?? "").toLowerCase().includes(q),
		);
	}, [roster, query]);

	const guildToAdd = useMemo(
		() => roster.filter((r) => r.sameGuild && !r.isInParty && !r.isLocal),
		[roster],
	);
	const trackedCount = roster.filter((r) => r.isInParty || r.isLocal).length;

	const toggle = (r: RosterEntry): void => {
		if (r.isLocal) return; // local is always tracked
		sendCommand(r.isInParty ? "removePartyMember" : "addPartyMember", r.userGuid);
	};
	const addAllGuild = (): void => {
		for (const r of guildToAdd) sendCommand("addPartyMember", r.userGuid);
	};

	// Section dividers: the agent sorts local → same-guild → others, so we
	// just label the first row of each block as we walk the list.
	let lastSection = "";
	const sectionOf = (r: RosterEntry): string =>
		r.isLocal ? "you" : r.sameGuild ? "guild" : "other";

	return (
		<div
			className="fixed inset-0 z-50 flex items-center justify-center p-4"
			style={{ background: "rgba(0,0,0,0.6)", backdropFilter: "blur(4px)" }}
			onClick={onClose}
		>
			<div
				className="w-full max-w-lg flex flex-col"
				style={{
					background: "var(--sk-bg-0)",
					border: "1px solid var(--sk-line)",
					borderRadius: 8,
					maxHeight: "85vh",
					overflow: "hidden",
				}}
				onClick={(e) => e.stopPropagation()}
			>
				<header
					className="flex items-center justify-between px-4 py-3"
					style={{ borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-1)" }}
				>
					<div className="flex items-center" style={{ gap: 10 }}>
						<span style={{ fontSize: 14, fontWeight: 600 }}>Track Players</span>
						<span
							className="sk-upper pl-2.5"
							style={{ borderLeft: "1px solid var(--sk-line)", color: "var(--sk-fg-3)", fontSize: 10 }}
						>
							{trackedCount} tracked · {roster.length} in range
						</span>
					</div>
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
						✕
					</button>
				</header>

				<div
					className="flex items-center"
					style={{ gap: 8, padding: "10px 14px", borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-inset)" }}
				>
					<input
						value={query}
						onChange={(e) => setQuery(e.target.value)}
						placeholder="Search name or guild…"
						style={{
							flex: 1,
							appearance: "none",
							background: "var(--sk-bg-0)",
							border: "1px solid var(--sk-line)",
							borderRadius: 5,
							padding: "6px 10px",
							fontSize: 13,
							color: "var(--sk-fg-0)",
						}}
					/>
					{guildToAdd.length > 0 && (
						<button
							onClick={addAllGuild}
							className="sk-upper"
							style={{
								whiteSpace: "nowrap",
								appearance: "none",
								border: "1px solid color-mix(in oklab, var(--sk-ok) 45%, var(--sk-line))",
								background: "color-mix(in oklab, var(--sk-ok) 12%, var(--sk-bg-2))",
								color: "var(--sk-ok)",
								padding: "6px 10px",
								borderRadius: 5,
								cursor: "pointer",
								fontSize: 10,
								fontWeight: 700,
								letterSpacing: "0.06em",
							}}
						>
							+ Add all guild · {guildToAdd.length}
						</button>
					)}
				</div>

				<div className="overflow-auto" style={{ flex: 1 }}>
					{filtered.length === 0 ? (
						<div style={{ padding: "28px 16px", textAlign: "center", color: "var(--sk-fg-3)", fontSize: 13 }}>
							{roster.length === 0
								? "No players seen yet — they'll appear here as the agent sees them in your zone."
								: "No matches."}
						</div>
					) : (
						filtered.map((r) => {
							const section = sectionOf(r);
							const showHeader = section !== lastSection;
							lastSection = section;
							return (
								<div key={r.userGuid}>
									{showHeader && (
										<div
											className="sk-upper"
											style={{
												padding: "9px 16px 6px",
												fontSize: 9.5,
												letterSpacing: "0.1em",
												fontWeight: 700,
												color: "var(--sk-fg-3)",
												background: "var(--sk-bg-inset)",
											}}
										>
											{section === "you" ? "You" : section === "guild" ? "Your guild" : "Other players in range"}
										</div>
									)}
									<PickerRow r={r} onToggle={toggle} />
								</div>
							);
						})
					)}
				</div>
			</div>
		</div>
	);
}

function PickerRow({ r, onToggle }: { r: RosterEntry; onToggle: (r: RosterEntry) => void }): React.ReactElement {
	const role = roleKeyOf(r.role);
	const accent = classAccent(r.classCode, role);
	const tracked = !!r.isInParty || !!r.isLocal;
	return (
		<div
			className="flex items-center"
			style={{ gap: 12, padding: "9px 16px", borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-1)" }}
		>
			<span
				style={{
					minWidth: 40,
					padding: "3px 8px",
					borderRadius: 4,
					textAlign: "center",
					border: `1px solid color-mix(in oklab, ${accent} 55%, var(--sk-line-2))`,
					background: `color-mix(in oklab, ${accent} 12%, var(--sk-bg-2))`,
					color: accent,
					fontFamily: "var(--sk-font-mono)",
					fontWeight: 700,
					fontSize: 12,
					fontVariantNumeric: "tabular-nums",
				}}
				title={r.itemPower ? `IP ${r.itemPower}` : "IP unknown"}
			>
				{r.itemPower && r.itemPower > 0 ? r.itemPower : "—"}
			</span>
			<div className="flex flex-col" style={{ minWidth: 0, flex: 1, gap: 1 }}>
				<span style={{ fontSize: 13.5, fontWeight: 600, color: "var(--sk-fg-0)" }}>{r.name}</span>
				<span className="flex items-center" style={{ gap: 8 }}>
					{r.roleLabel && (
						<span className="sk-upper" style={{ fontSize: 9, color: accent, fontWeight: 700 }}>{r.roleLabel}</span>
					)}
					{r.guild && !r.sameGuild && (
						<span style={{ fontSize: 10, color: "var(--sk-fg-3)" }}>{r.guild}</span>
					)}
				</span>
			</div>
			{r.isLocal ? (
				<span
					className="sk-upper"
					style={{ fontSize: 9.5, fontWeight: 700, letterSpacing: "0.06em", color: "var(--sk-fg-3)", padding: "4px 10px" }}
				>
					You
				</span>
			) : (
				<button
					onClick={() => onToggle(r)}
					className="sk-upper"
					style={{
						appearance: "none",
						minWidth: 78,
						border: tracked
							? "1px solid color-mix(in oklab, var(--sk-local) 55%, var(--sk-line))"
							: "1px solid var(--sk-line)",
						background: tracked
							? "color-mix(in oklab, var(--sk-local) 18%, var(--sk-bg-2))"
							: "var(--sk-bg-2)",
						color: tracked ? "var(--sk-local)" : "var(--sk-fg-2)",
						padding: "4px 12px",
						borderRadius: 4,
						cursor: "pointer",
						fontSize: 10,
						fontWeight: 700,
						letterSpacing: "0.06em",
					}}
				>
					{tracked ? "✓ Tracking" : "+ Track"}
				</button>
			)}
		</div>
	);
}
