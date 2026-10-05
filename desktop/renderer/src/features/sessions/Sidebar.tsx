import { compareTime } from "./merge";
import type { KeyboardEventHandler, PointerEventHandler } from "react";
import { PlusIcon } from "@phosphor-icons/react";
import type { ConversationSummary } from "./types";

type Props = {
    sidebar: {
        open: boolean;
        width: number;
        maximumWidth: number;
        onKeyDown: KeyboardEventHandler<HTMLDivElement>;
        onPointerDown: PointerEventHandler<HTMLDivElement>;
    };
    conversations: ConversationSummary[];
    selected: string;
    onNewChat: () => void;
    onSelect: (id: string) => void;
};

export function Sidebar({
    sidebar,
    conversations,
    selected,
    onNewChat,
    onSelect,
}: Props) {
    const rows = [...conversations].sort(
        (a, b) => compareTime(b.updatedAt, a.updatedAt) || b.id.localeCompare(a.id),
    );
    return (
        <>
            <aside
                className="sidebar"
                aria-label="Sessions"
                aria-hidden={!sidebar.open}
                inert={!sidebar.open}
            >
                <div className="sidebar-inner">
                    <div className="sidebar-titlebar" aria-hidden="true" />
                    <div className="sidebar-content">
                        <button className="new-chat-button" onClick={onNewChat}>
                            <PlusIcon size={18} />
                            <span>New chat</span>
                            <kbd>⌘ N</kbd>
                        </button>
                        <div className="sidebar-section-label">Sessions</div>
                        <div className="chat-list">
                            {rows.map((chat) => (
                                <div
                                    className={`chat-row ${chat.id === selected ? "selected" : ""}`}
                                    key={chat.id}
                                >
                                    <button
                                        className="chat-select"
                                        onClick={() => onSelect(chat.id)}
                                        aria-current={
                                            chat.id === selected ? "page" : undefined
                                        }
                                    >
                                        <span
                                            className={`session-status ${chat.status}`}
                                            aria-hidden="true"
                                        />
                                        <span>{chat.title}</span>
                                    </button>
                                </div>
                            ))}
                        </div>
                    </div>
                </div>
            </aside>
            <div
                className="sidebar-resizer"
                role="separator"
                aria-hidden={!sidebar.open}
                inert={!sidebar.open}
                aria-label="Sidebar width"
                aria-orientation="vertical"
                aria-valuemin={0}
                aria-valuemax={sidebar.maximumWidth}
                aria-valuenow={sidebar.open ? Math.round(sidebar.width) : 0}
                tabIndex={sidebar.open ? 0 : -1}
                onKeyDown={sidebar.onKeyDown}
                onPointerDown={sidebar.onPointerDown}
            />
        </>
    );
}
