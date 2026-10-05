import {
    ChatCircleDotsIcon,
    ShieldCheckIcon,
    LightningIcon,
} from "@phosphor-icons/react";
import type {
    PermissionMode,
    PermissionPolicy,
} from "../../../../contracts/wire.generated";

export const permissionModes = [
    {
        value: "ask",
        label: "Ask",
        icon: ChatCircleDotsIcon,
        description: "Ask before every tool use.",
    },
    {
        value: "folders",
        label: "Auto",
        icon: ShieldCheckIcon,
        description: "Allow changes in selected folders. Bash has no network access.",
    },
    {
        value: "full",
        label: "Bypass",
        tone: "danger",
        icon: LightningIcon,
        description: "Allow changes and commands anywhere without asking.",
    },
] as const;

export function changePermissionMode(
    policy: PermissionPolicy,
    mode: PermissionMode,
    directory: string,
): PermissionPolicy {
    return {
        mode,
        roots: mode === "folders" && !policy.roots.length ? [directory] : policy.roots,
    };
}
