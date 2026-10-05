import type { useSessions } from "../features/sessions/useSessions";
import type { useChat } from "../features/chat/useChat";
import type { useAttachments } from "../features/chat/attachments/useAttachments";
import type { useChatPermissions } from "../features/permissions/useChatPermissions";
import type { useApproval } from "../features/permissions/useApproval";
import type { usePreferences } from "../platform/preferences/usePreferences";
import type { useLocalModels } from "../features/models/providers/local/useLocalModels";
import type { useContextReport } from "../features/chat/context/useContextReport";
import type { useChatActions } from "./useChatActions";
import type { DiscoveredModel } from "../../../contracts/wire.generated";
import { ResourcePolicy } from "../rich-text/ResourcePolicy";
import { Chat } from "../features/chat/Chat";
import { Composer } from "../features/chat/composer/Composer";
import { Approval } from "../features/permissions/Approval";
import { selectRun } from "../features/models/picker/selection";

type Props = {
    sessions: ReturnType<typeof useSessions>;
    chat: ReturnType<typeof useChat>;
    attachments: ReturnType<typeof useAttachments>;
    permissions: ReturnType<typeof useChatPermissions>;
    approval: ReturnType<typeof useApproval>;
    preferences: ReturnType<typeof usePreferences>;
    local: ReturnType<typeof useLocalModels>;
    actions: ReturnType<typeof useChatActions>;
    contextReport: ReturnType<typeof useContextReport>;
    availableModels: DiscoveredModel[];
    selectedModel?: DiscoveredModel;
    availability: "ready" | "requiresLoad" | "unavailable";
    backendReady: boolean;
    catalogueBusy: boolean;
    openConnections: () => void;
};
export function ChatWorkspace({
    sessions,
    chat,
    attachments,
    permissions,
    approval,
    preferences,
    local,
    actions,
    contextReport,
    availableModels,
    selectedModel,
    availability,
    backendReady,
    catalogueBusy,
    openConnections,
}: Props) {
    const { model, effort } = preferences.settings.run;
    const { loadingLocal } = actions;
    return (
        <>
            <ResourcePolicy.Provider
                value={{
                    conversation: sessions.selected,
                    mode: permissions.policy.mode,
                }}
            >
                <Chat
                    key={sessions.selected}
                    session={sessions.current}
                    stopping={chat.cancelling}
                    compacting={chat.event.compacting}
                    recovery={chat.event.recovery}
                    activeMessage={
                        chat.running && chat.event.run.conversation === sessions.selected
                            ? chat.event.message?.id
                            : undefined
                    }
                    more={sessions.more}
                    loading={sessions.loading}
                    onLoadOlder={sessions.loadOlder}
                />
            </ResourcePolicy.Provider>
            {approval.request && (
                <Approval
                    key={approval.request.id}
                    request={approval.request}
                    responding={approval.responding}
                    onRespond={approval.respond}
                />
            )}
            <Composer
                attachments={attachments}
                vision={Boolean(selectedModel?.vision)}
                permissions={permissions.policy}
                directory={preferences.settings.directory}
                onPermissions={permissions.change}
                draft={sessions.draft}
                onChange={sessions.updateDraft}
                onSend={actions.send}
                onStop={actions.stop}
                onSettings={openConnections}
                onModel={async (value) => {
                    await preferences.configure({
                        run: selectRun(availableModels, {
                            model: value,
                            effort,
                        }),
                    });
                    return true;
                }}
                effort={effort}
                capabilities={selectedModel?.thinking}
                onEffort={async (value) =>
                    preferences.configure({ run: { effort: value } })
                }
                models={availableModels}
                unloadedModels={local.models
                    .filter(
                        (item) =>
                            local.runtime.state !== "ready" ||
                            local.runtime.model !== item.id,
                    )
                    .map((item) => item.id)}
                catalogueError={
                    !availableModels.length && !catalogueBusy ? "Choose a model" : ""
                }
                model={model}
                running={chat.running}
                starting={chat.starting}
                loadingModel={loadingLocal}
                cancelling={chat.cancelling}
                disabled={Boolean(window.arxDesktop) && !backendReady}
                ready={Boolean(window.arxDesktop) && availability !== "unavailable"}
                contextConversation={sessions.selected}
                contextReport={contextReport}
            />
        </>
    );
}
