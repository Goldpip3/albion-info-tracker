import { useState } from "react";
import type { ActivityEvent, Snapshot } from "./types.ts";

interface ActivityLogProps {
	snapshot: Snapshot | null;
}

type Filter = "all" | "deaths" | "bighits" | "heals";

export function ActivityLog({ snapshot }: ActivityLogProps): React.ReactElement {
	const [filter, setFilter] = useState<Filter>("all");
	const events = snapshot?.events ?? [];

	const visible = events.filter((e) => {
		switch (filter) {
			case "all":     return true;
			case "deaths":  return e.kind === "death";
			case "bighits": return e.kind === "hit" && Boolean(e.bigHit);
			case "heals":   return e.kind === "heal";
		}
	});

	return (
		<div className="bg-skirmish-bg2 border border-skirmish-line rounded-md flex flex-col text-xs">
			<header className="flex items-center justify-between px-3 py-2 border-b border-skirmish-line">
				<div className="text-skirmish-text font-medium">Activity log · kill feed</div>
				<div className="flex gap-1">
					<FilterTab current={filter} value="all"     onClick={setFilter}>All</FilterTab>
					<FilterTab current={filter} value="deaths"  onClick={setFilter}>Deaths</FilterTab>
					<FilterTab current={filter} value="bighits" onClick={setFilter}>Big hits</FilterTab>
					<FilterTab current={filter} value="heals"   onClick={setFilter}>Heals</FilterTab>
				</div>
			</header>
			<div className="flex-1 overflow-auto max-h-64">
				{visible.length === 0 ? (
					<div className="text-skirmish-muted text-center py-6 text-[11px]">
						{events.length === 0 ? "No events yet." : "No matches in this view."}
					</div>
				) : (
					<ul>
						{visible.map((ev, i) => (
							<ActivityRow key={`${ev.ts}-${i}`} ev={ev} />
						))}
					</ul>
				)}
			</div>
		</div>
	);
}

function FilterTab({
	current,
	value,
	onClick,
	children,
}: {
	current: Filter;
	value: Filter;
	onClick: (f: Filter) => void;
	children: React.ReactNode;
}): React.ReactElement {
	const active = current === value;
	return (
		<button
			onClick={() => onClick(value)}
			className={`px-2 py-0.5 text-[10px] uppercase tracking-[0.12em] rounded transition ${
				active
					? "bg-skirmish-text text-skirmish-bg"
					: "text-skirmish-dim hover:text-skirmish-text"
			}`}
		>
			{children}
		</button>
	);
}

function ActivityRow({ ev }: { ev: ActivityEvent }): React.ReactElement {
	const time = formatTime(ev.ts);
	const tag = kindTag(ev);
	return (
		<li className="grid grid-cols-[3rem_3.5rem_1fr_auto] gap-2 px-3 py-1 items-center border-b border-skirmish-line/40 last:border-b-0">
			<span className="text-skirmish-muted tnum">{time}</span>
			<span className={`text-[10px] uppercase tracking-[0.12em] ${tag.tone}`}>{tag.label}</span>
			<span className="truncate text-skirmish-text">
				<EventBody ev={ev} />
			</span>
			{ev.amount !== undefined && ev.amount > 0 && (
				<span className="tnum text-skirmish-text">{formatNum(ev.amount)}</span>
			)}
		</li>
	);
}

function EventBody({ ev }: { ev: ActivityEvent }): React.ReactElement {
	switch (ev.kind) {
		case "death":
			return (
				<>
					<Name n={ev.target} /> defeated{ev.actor ? <> by <Name n={ev.actor} /></> : null}
				</>
			);
		case "heal":
			return (
				<>
					<Name n={ev.actor} local={ev.actorIsLocal} />
					{ev.spellName ? <> {prettySpell(ev.spellName)} →</> : <> healed</>}
					{ev.target ? <> <Name n={ev.target} /></> : null}
				</>
			);
		case "hit":
			return (
				<>
					<Name n={ev.actor} local={ev.actorIsLocal} />
					{ev.spellName ? <> {prettySpell(ev.spellName)} →</> : <> hit</>}
					{ev.target ? <> <Name n={ev.target} /></> : <> someone</>}
					{ev.bigHit && <span className="ml-1 text-[10px] text-amber-300">crit?</span>}
				</>
			);
	}
}

function Name({ n, local }: { n?: string; local?: boolean }): React.ReactElement {
	if (!n) return <span className="text-skirmish-muted">—</span>;
	return <span className={local ? "text-skirmish-amber" : "text-skirmish-text"}>{n}</span>;
}

function kindTag(ev: ActivityEvent): { label: string; tone: string } {
	if (ev.kind === "death") return { label: "Death", tone: "text-rose-300" };
	if (ev.kind === "heal") return { label: "Heal", tone: "text-emerald-300" };
	if (ev.bigHit) return { label: "Big hit", tone: "text-amber-300" };
	return { label: "Hit", tone: "text-skirmish-dim" };
}

function formatTime(unixMs: number): string {
	const d = new Date(unixMs);
	const hh = d.getHours().toString().padStart(2, "0");
	const mm = d.getMinutes().toString().padStart(2, "0");
	const ss = d.getSeconds().toString().padStart(2, "0");
	return `${hh}:${mm}:${ss}`;
}

function prettySpell(name: string): string {
	const parts = name.split("_").filter(Boolean);
	if (parts.length === 0) return name;
	const first = parts[0].charAt(0).toUpperCase() + parts[0].slice(1).toLowerCase();
	const rest = parts.slice(1).map((p) => p.toLowerCase()).join(" ");
	return rest ? `${first} ${rest}` : first;
}

function formatNum(n: number): string {
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
	if (n >= 10_000) return `${(n / 1000).toFixed(1)}K`;
	if (n >= 1000) return `${(n / 1000).toFixed(2)}K`;
	return Math.round(n).toString();
}
