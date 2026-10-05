import { useState } from "react";
import type { Preferences } from "../../platform/preferences/usePreferences";
import type { PermissionPolicy } from "../../../../contracts/wire.generated";

export function useChatPermissions(conversation: string, preferences: Preferences) {
    const [draft, setDraft] = useState<PermissionPolicy | null>(null);
    const fallback: PermissionPolicy = {
        mode: "folders",
        roots: [preferences.settings.directory],
    };
    const policy = conversation
        ? preferences.settings.chat_permissions?.[conversation] || fallback
        : draft || preferences.settings.permissions;
    const change = async (next: PermissionPolicy) => {
        if (conversation) await preferences.configurePermissions(conversation, next);
        else setDraft(next);
    };
    return {
        policy,
        change,
        reset: () => setDraft(null),
        remember: preferences.rememberPermissions,
    };
}
