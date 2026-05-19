import { accentOklch } from "./useSettings.ts";
import type { AccentColor, BarStyle, Density, Settings } from "./useSettings.ts";

interface SettingsPanelProps {
	settings: Settings;
	update: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
	reset: () => void;
	onClose: () => void;
}

const ACCENT_OPTIONS: AccentColor[] = ["amber", "blue", "purple", "green", "red", "yellow", "pink"];
const DENSITY_OPTIONS: Density[] = [24, 28, 34];
const BAR_STYLE_OPTIONS: BarStyle[] = ["outline", "tint", "solid"];

export function SettingsPanel({ settings, update, reset, onClose }: SettingsPanelProps): React.ReactElement {
	return (
		<div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4" onClick={onClose}>
			<div
				className="w-full max-w-lg bg-skirmish-bg2 border border-skirmish-line rounded-md text-sm"
				onClick={(e) => e.stopPropagation()}
			>
				<header className="flex items-center justify-between px-4 py-3 border-b border-skirmish-line">
					<div>
						<div className="text-skirmish-text font-medium">Settings</div>
						<div className="text-[10px] uppercase tracking-[0.2em] text-skirmish-muted">
							meter.skirmish.gg
						</div>
					</div>
					<button onClick={onClose} className="text-skirmish-muted hover:text-skirmish-text text-xl leading-none">
						×
					</button>
				</header>

				<Section title="Display">
					<Row label="Columns" hint="Which secondary metrics show on each row">
						<div className="flex flex-wrap gap-1.5">
							<Chip on={settings.columns.itemPower} onClick={() => update("columns", { ...settings.columns, itemPower: !settings.columns.itemPower })}>Item Power</Chip>
							<Chip on={settings.columns.dps} onClick={() => update("columns", { ...settings.columns, dps: !settings.columns.dps })}>DPS</Chip>
							<Chip on={settings.columns.hps} onClick={() => update("columns", { ...settings.columns, hps: !settings.columns.hps })}>HPS</Chip>
							<Chip on={settings.columns.damageTaken} onClick={() => update("columns", { ...settings.columns, damageTaken: !settings.columns.damageTaken })}>Damage Taken</Chip>
							<Chip on={settings.columns.healing} onClick={() => update("columns", { ...settings.columns, healing: !settings.columns.healing })}>Healing</Chip>
						</div>
					</Row>

					<Row label="Theme accent" hint="Color used for your local-player row">
						<div className="flex gap-1.5">
							{ACCENT_OPTIONS.map((c) => (
								<button
									key={c}
									onClick={() => update("accent", c)}
									className={`w-5 h-5 rounded-sm border ${settings.accent === c ? "border-white" : "border-skirmish-line"}`}
									style={{ background: accentOklch(c).fg }}
									title={c}
								/>
							))}
						</div>
					</Row>

					<Row label="Density" hint="Row height">
						<Segmented
							options={DENSITY_OPTIONS.map((d) => ({ value: d, label: d.toString() }))}
							current={settings.density}
							onChange={(v) => update("density", v)}
						/>
					</Row>

					<Row label="Bar style">
						<Segmented
							options={BAR_STYLE_OPTIONS.map((b) => ({ value: b, label: cap(b) }))}
							current={settings.barStyle}
							onChange={(v) => update("barStyle", v)}
						/>
					</Row>
				</Section>

				<Section title="Sort & grouping">
					<Row label="Group by role" hint="Cluster Tanks, Healers, DPS visually">
						<Toggle on={settings.groupByRole} onClick={() => update("groupByRole", !settings.groupByRole)} />
					</Row>
					<Row label="Pin local user" hint="Always show your row, even if outside top N">
						<Toggle on={settings.pinLocal} onClick={() => update("pinLocal", !settings.pinLocal)} />
					</Row>
				</Section>

				<footer className="flex items-center justify-between px-4 py-3 border-t border-skirmish-line text-[10px] uppercase tracking-[0.15em] text-skirmish-muted">
					<span>Changes auto-save</span>
					<div className="flex gap-2">
						<button
							onClick={reset}
							className="px-3 py-1 rounded border border-skirmish-line text-skirmish-dim hover:text-skirmish-text"
						>
							Reset defaults
						</button>
						<button
							onClick={onClose}
							className="px-3 py-1 rounded bg-skirmish-amber text-skirmish-bg font-medium"
						>
							Done
						</button>
					</div>
				</footer>
			</div>
		</div>
	);
}

function Section({ title, children }: { title: string; children: React.ReactNode }): React.ReactElement {
	return (
		<section className="px-4 py-3 border-b border-skirmish-line/60 last:border-b-0">
			<div className="text-[10px] uppercase tracking-[0.2em] text-skirmish-muted mb-2">{title}</div>
			<div className="space-y-3">{children}</div>
		</section>
	);
}

function Row({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }): React.ReactElement {
	return (
		<div className="grid grid-cols-[1fr_auto] gap-3 items-center">
			<div>
				<div className="text-skirmish-text text-sm">{label}</div>
				{hint && <div className="text-skirmish-muted text-xs">{hint}</div>}
			</div>
			<div>{children}</div>
		</div>
	);
}

function Chip({ on, onClick, children }: { on: boolean; onClick: () => void; children: React.ReactNode }): React.ReactElement {
	return (
		<button
			onClick={onClick}
			className={`px-2 py-0.5 rounded-sm text-[11px] border transition ${
				on
					? "bg-skirmish-amber/15 text-skirmish-amber border-skirmish-amber/40"
					: "bg-skirmish-bg text-skirmish-muted border-skirmish-line hover:text-skirmish-text"
			}`}
		>
			{children}
		</button>
	);
}

function Toggle({ on, onClick }: { on: boolean; onClick: () => void }): React.ReactElement {
	return (
		<button
			onClick={onClick}
			className={`w-9 h-5 rounded-full transition relative ${on ? "bg-skirmish-amber" : "bg-skirmish-line"}`}
		>
			<span
				className={`absolute top-0.5 w-4 h-4 bg-skirmish-bg rounded-full transition ${on ? "left-4" : "left-0.5"}`}
			/>
		</button>
	);
}

interface SegmentedProps<T extends string | number> {
	options: Array<{ value: T; label: string }>;
	current: T;
	onChange: (v: T) => void;
}

function Segmented<T extends string | number>({ options, current, onChange }: SegmentedProps<T>): React.ReactElement {
	return (
		<div className="inline-flex border border-skirmish-line rounded overflow-hidden">
			{options.map((opt) => (
				<button
					key={opt.value.toString()}
					onClick={() => onChange(opt.value)}
					className={`px-2.5 py-1 text-xs ${
						current === opt.value
							? "bg-skirmish-text text-skirmish-bg"
							: "text-skirmish-dim hover:text-skirmish-text"
					}`}
				>
					{opt.label}
				</button>
			))}
		</div>
	);
}

function cap(s: string): string {
	return s.charAt(0).toUpperCase() + s.slice(1);
}
