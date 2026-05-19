import { useEffect, useState } from "react";
import { useMeterSocket } from "./useMeterSocket.ts";
import { LootBody } from "./LootBody.tsx";
import { accentOklch, useSettings } from "./useSettings.ts";

const DEFAULT_VIEW_URL = "wss://albion-meter.goldpipe.workers.dev/view";

// LootPage is the fullscreen /loot route. Same body as the modal, but
// without backdrop / Close — designed to live on a second monitor while
// the main meter runs in another tab.
//
// Pairing: prefer ?token=<token> in the URL (the modal's "Open as page"
// button passes it), fall back to localStorage so a refresh keeps
// working. When the URL supplied the token we mirror it into
// localStorage and strip the query param so bookmarks stay clean.
export function LootPage(): React.ReactElement {
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

	if (!configured) {
		return (
			<div className="min-h-dvh flex items-center justify-center" style={{ background: "var(--sk-bg-0)", color: "var(--sk-fg-1)", padding: 24 }}>
				<div style={{ maxWidth: 360, textAlign: "center" }}>
					<div className="sk-upper" style={{ marginBottom: 8, color: "var(--sk-fg-3)" }}>Not paired</div>
					<p style={{ fontSize: 13, lineHeight: 1.5, color: "var(--sk-fg-2)" }}>
						Open <span className="sk-mono">/?pair=&lt;token&gt;</span> first to pair this browser
						with the agent, then come back to <span className="sk-mono">/loot</span>.
					</p>
				</div>
			</div>
		);
	}

	return (
		<div
			className="min-h-dvh flex flex-col"
			style={{ background: "var(--sk-bg-0)", color: "var(--sk-fg-0)" }}
		>
			<header
				className="flex items-center"
				style={{
					gap: 12,
					padding: "12px 18px",
					borderBottom: "1px solid var(--sk-line)",
					background: "var(--sk-bg-1)",
				}}
			>
				<img
					src="/assets/icon-GDA-32.png"
					srcSet="/assets/icon-GDA-32.png 1x, /assets/icon-GDA-64.png 2x"
					alt="GDA"
					width={20}
					height={20}
					style={{ display: "block", borderRadius: 4 }}
				/>
				<span style={{ fontSize: 13, fontWeight: 600, color: "var(--sk-fg-0)" }}>
					Loot — who farmed what
				</span>
				<span className="sk-upper" style={{ fontSize: 9.5, color: "var(--sk-fg-3)", letterSpacing: "0.08em", marginLeft: 4 }}>
					session view
				</span>
			</header>
			<div className="flex-1 flex flex-col" style={{ minHeight: 0 }}>
				<LootBody
					loot={snapshot?.loot ?? []}
					looterTotals={snapshot?.looterTotals ?? []}
					players={snapshot?.players ?? []}
					session={snapshot?.session ?? null}
					generatedAt={snapshot?.generatedAt}
				/>
			</div>
		</div>
	);
}
