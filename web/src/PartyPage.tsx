import { useEffect, useState } from "react";
import { useMeterSocket } from "./useMeterSocket.ts";
import { PartyBody } from "./PartyBody.tsx";
import { TabBar } from "./TabBar.tsx";
import { tabsFor, type TabId } from "./tabs.ts";
import { accentOklch, useSettings } from "./useSettings.ts";

const DEFAULT_VIEW_URL = "wss://albion-meter.goldpipe.workers.dev/view";

// PartyPage is the dedicated /party route. Mirrors LootPage's
// pairing flow — read ?token= from the URL if present, otherwise fall
// back to localStorage. Renders the shared PartyBody with no modal
// chrome.
export function PartyPage(): React.ReactElement {
	const { settings } = useSettings();
	const [token, setToken] = useState<string>("");
	const [url, setUrl] = useState<string>("");

	useEffect(() => {
		let resolvedToken = "";
		let resolvedUrl = "";
		try {
			const params = new URLSearchParams(window.location.search);
			const fromURL = params.get("token");
			if (fromURL && fromURL.trim()) {
				resolvedToken = fromURL.trim();
				try { localStorage.setItem("skirmish:token", resolvedToken); } catch { /* private mode */ }
				params.delete("token");
				const qs = params.toString();
				window.history.replaceState({}, "", window.location.pathname + (qs ? "?" + qs : ""));
			} else {
				resolvedToken = (localStorage.getItem("skirmish:token") ?? "").trim();
			}
			resolvedUrl = (localStorage.getItem("skirmish:url") ?? DEFAULT_VIEW_URL).trim();
		} catch {
			resolvedUrl = DEFAULT_VIEW_URL;
		}
		setToken(resolvedToken);
		setUrl(resolvedUrl);
	}, []);

	useEffect(() => {
		const { fg, tint } = accentOklch(settings.accent);
		document.documentElement.style.setProperty("--sk-local", fg);
		document.documentElement.style.setProperty("--sk-local-tint", tint);
	}, [settings.accent]);

	const configured = !!url && !!token;
	const { snapshot } = useMeterSocket({ url, token, enabled: configured });

	const tabs = tabsFor(snapshot, token);
	const active: TabId = "party";

	if (!configured) {
		return (
			<div className="min-h-dvh flex flex-col" style={{ background: "var(--sk-bg-0)", color: "var(--sk-fg-1)" }}>
				<TabBar tabs={tabs} active={active} />
				<div className="flex-1 flex items-center justify-center" style={{ padding: 24 }}>
					<div style={{ maxWidth: 360, textAlign: "center" }}>
						<div className="sk-upper" style={{ marginBottom: 8, color: "var(--sk-fg-3)" }}>Not paired</div>
						<p style={{ fontSize: 13, lineHeight: 1.5, color: "var(--sk-fg-2)" }}>
							Open <span className="sk-mono">/?pair=&lt;token&gt;</span> first to pair this browser
							with the agent, then come back to <span className="sk-mono">/party</span>.
						</p>
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="min-h-dvh flex flex-col" style={{ background: "var(--sk-bg-0)", color: "var(--sk-fg-0)" }}>
			<TabBar tabs={tabs} active={active} />
			<div className="flex-1 flex flex-col" style={{ minHeight: 0 }}>
				<PartyBody players={snapshot?.players ?? []} generatedAt={snapshot?.generatedAt} />
			</div>
		</div>
	);
}
