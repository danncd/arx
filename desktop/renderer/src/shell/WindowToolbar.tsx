import {
    GearSixIcon,
    NotePencilIcon,
    SidebarIcon,
    SidebarSimpleIcon,
} from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { useWindowDrag } from "./useWindowDrag";
import { SessionUsage } from "../features/chat/usage/SessionUsage";
import type { ContextReport } from "../../../contracts/wire.generated";
import "./toolbar.css";
import "../styles/icon-pill.css";

const pillSize = 37;
const openControlWidth = 36;
const pillGap = 8;
const edgeInset = 14;
const windowedToggleLeft = 89;

export function WindowToolbar({
    sidebarOpen,
    sidebarWidth,
    resizing,
    title,
    conversation,
    usage,
    onToggleSidebar,
    onNewChat,
    onSettings,
    settingsReady,
}: {
    sidebarOpen: boolean;
    sidebarWidth: number;
    resizing: boolean;
    title: string;
    conversation: string;
    usage: ContextReport["usage"] | undefined;
    onToggleSidebar: () => void;
    onNewChat: () => void;
    onSettings: () => void;
    settingsReady: boolean;
}) {
    const drag = useWindowDrag();
    const [fullscreen, setFullscreen] = useState(false);
    useEffect(() => window.arxDesktop?.onFullscreenChanged?.(setFullscreen), []);
    const toggleLeft = fullscreen ? edgeInset : windowedToggleLeft;
    const controlStyle = { width: sidebarOpen ? openControlWidth : pillSize };
    return (
        <header className="window-toolbar" {...drag}>
            <div
                className="header-controls"
                style={{
                    left:
                        toggleLeft +
                        (sidebarOpen ? (pillSize - openControlWidth) / 2 : 0),
                    gap: sidebarOpen ? 0 : pillGap,
                }}
            >
                <button
                    type="button"
                    className={`sidebar-control ${sidebarOpen ? "" : "is-pill"}`}
                    style={controlStyle}
                    aria-label={sidebarOpen ? "Hide sidebar" : "Show sidebar"}
                    title={sidebarOpen ? "Hide sidebar" : "Show sidebar"}
                    onClick={onToggleSidebar}
                >
                    {sidebarOpen ? <SidebarIcon /> : <SidebarSimpleIcon />}
                </button>
                <button
                    type="button"
                    className={`sidebar-control ${sidebarOpen ? "" : "is-pill"}`}
                    style={controlStyle}
                    aria-label="Settings"
                    title="Settings (⌘ ,)"
                    onClick={onSettings}
                    disabled={!settingsReady}
                >
                    <GearSixIcon />
                </button>
            </div>
            <div
                className={`header-chat-action ${resizing ? "dragging" : ""}`}
                style={{
                    left: sidebarOpen
                        ? sidebarWidth + edgeInset
                        : toggleLeft + 2 * (pillSize + pillGap),
                }}
            >
                <button
                    type="button"
                    className="icon-pill header-action-pill"
                    aria-label="New chat"
                    title="New chat (⌘ N)"
                    onClick={onNewChat}
                >
                    <NotePencilIcon />
                </button>
                <span className="chat-title">{title}</span>
                <SessionUsage key={conversation} usage={usage} empty={!conversation} />
            </div>
        </header>
    );
}
