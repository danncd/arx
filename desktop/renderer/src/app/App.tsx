import { GenerationProvider } from "../features/generation/state/useGeneration";
import { availableModels as selectModels } from "../features/models/catalog";
import { ChatWorkspace } from "./ChatWorkspace";
import { useNetworkModels } from "../features/models/providers/network/useNetworkModels";
import { useShellPreferences } from "./useShellPreferences";
import { useChatActions } from "./useChatActions";
import { useEffect, useState, type CSSProperties } from "react";
import type { InitialState } from "./loadState";
import { usePreferences } from "../platform/preferences/usePreferences";
import { useChatPermissions } from "../features/permissions/useChatPermissions";
import { useApproval } from "../features/permissions/useApproval";
import { useBackend } from "../platform/backend/useBackend";
import { ConnectionError } from "../shell/ConnectionError";
import { WindowToolbar } from "../shell/WindowToolbar";
import { useSidebar } from "../shell/useSidebar";
import { Sidebar } from "../features/sessions/Sidebar";
import { useSessions } from "../features/sessions/useSessions";
import { useContextReport } from "../features/chat/context/useContextReport";
import { useAttachments } from "../features/chat/attachments/useAttachments";
import { useChat } from "../features/chat/useChat";
import { ResponseViewProvider } from "../features/chat/response/ViewState";
import { useLocalModels } from "../features/models/providers/local/useLocalModels";
import { useConnection } from "../features/models/providers/deepseek/useConnection";
import { modelAvailability } from "../features/models/picker/selection";
import { Settings } from "../features/settings/Settings";
import {
    useNotifications,
    NotificationsViewport,
} from "../ui/notifications/Notifications";
import "../styles/theme.css";
import "../styles/base.css";
import "../styles/controls.css";
import "../shell/shell.css";
import "../shell/sidebar.css";
import "../features/chat/chat.css";

export function App({ initial }: { initial: InitialState }) {
    return (
        <GenerationProvider>
            <Workspace initial={initial} />
        </GenerationProvider>
    );
}

function Workspace({ initial }: { initial: InitialState }) {
    const [layout, setLayout] = useState(initial.layout);
    const [settings, setSettings] = useState(false);
    const [settingsSection, setSettingsSection] = useState<"general" | "connections">(
        "general",
    );
    const preferences = usePreferences(initial.snapshot);
    const { notify } = useNotifications();
    useEffect(() => {
        if (initial.snapshot.recovered)
            notify(
                "Some saved messages could not be recovered",
                "session-recovery",
                "warning",
            );
    }, [initial.snapshot.recovered, notify]);
    const { model } = preferences.settings.run;
    const sidebar = useSidebar({ layout, setLayout });
    const sessions = useSessions(initial, preferences);
    const backend = useBackend();
    const approval = useApproval(backend.status);
    const permissions = useChatPermissions(sessions.selected, preferences);
    const attachments = useAttachments(sessions.selected, preferences);
    const chat = useChat(backend.status, sessions, attachments, permissions);
    const newSession = () => {
        permissions.reset();
        sessions.newSession();
    };
    const connection = useConnection(backend.status, preferences);
    const local = useLocalModels(backend.status);
    const network = useNetworkModels(backend.status);
    const availableModels = selectModels(
        connection.models,
        network.servers,
        local.models,
    );
    const selectedModel = availableModels.find((item) => item.id === model);
    const availability = modelAvailability(
        selectedModel,
        connection.connected,
        local.runtime,
    );
    const actions = useChatActions(model, availability, chat, local);
    const { loadingLocal } = actions;
    const contextReport = useContextReport({
        conversation: sessions.selected,
        model,
        contextWindow: selectedModel?.contextWindow,
        directory: preferences.settings.directory,
        ready: backend.status.state === "ready",
        running: chat.running,
        compacting: Boolean(chat.event.compacting),
        revision: chat.event.revision,
    });
    const openConnections = () => {
        setSettingsSection("connections");
        setSettings(true);
    };
    const disconnected =
        Boolean(window.arxDesktop) &&
        (backend.status.state === "error" || backend.status.state === "stopped");
    useShellPreferences(layout, sidebar, preferences, settings, setSettings, newSession);
    return (
        <ResponseViewProvider>
            <div
                className={`app-shell ${sidebar.open ? "" : "sidebar-closed"} ${sidebar.resizing ? "resizing" : ""}`}
                style={{ "--sidebar-width": `${sidebar.width}px` } as CSSProperties}
            >
                <WindowToolbar
                    sidebarOpen={sidebar.open}
                    sidebarWidth={sidebar.width}
                    resizing={sidebar.resizing}
                    title={sessions.current?.title ?? ""}
                    conversation={sessions.selected}
                    usage={contextReport?.usage}
                    onToggleSidebar={sidebar.toggle}
                    onNewChat={newSession}
                    onSettings={() => {
                        setSettingsSection("general");
                        setSettings(true);
                    }}
                    settingsReady
                />
                <Sidebar
                    sidebar={sidebar}
                    conversations={sessions.sessions}
                    selected={sessions.selected}
                    onNewChat={newSession}
                    onSelect={sessions.select}
                />
                <main className="main-pane">
                    <div className="chat-content">
                        {disconnected ? (
                            <ConnectionError onRetry={() => void backend.restart()} />
                        ) : (
                            <ChatWorkspace
                                sessions={sessions}
                                chat={chat}
                                attachments={attachments}
                                permissions={permissions}
                                approval={approval}
                                preferences={preferences}
                                local={local}
                                actions={actions}
                                contextReport={contextReport}
                                availableModels={availableModels}
                                selectedModel={selectedModel}
                                availability={availability}
                                backendReady={backend.status.state === "ready"}
                                catalogueBusy={connection.busy}
                                openConnections={openConnections}
                            />
                        )}
                    </div>
                </main>
                {settings ? (
                    <Settings
                        preferences={preferences}
                        permissions={preferences.settings.permissions}
                        onPermissions={(policy) =>
                            preferences.configurePermissions("", policy)
                        }
                        network={network}
                        connection={connection}
                        local={local}
                        running={chat.running || loadingLocal}
                        initialSection={settingsSection}
                        onClose={() => setSettings(false)}
                    />
                ) : (
                    <NotificationsViewport />
                )}
            </div>
        </ResponseViewProvider>
    );
}
