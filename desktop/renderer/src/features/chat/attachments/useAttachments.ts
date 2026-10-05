import { useRef, useState } from "react";
import type { Preferences } from "../../../platform/preferences/usePreferences";
import { useNotifications } from "../../../ui/notifications/Notifications";
import type { ImageAttachment } from "../../../../../contracts/wire.generated";

function read(value: string | undefined): ImageAttachment[] {
    try {
        const images: unknown = JSON.parse(value || "[]");
        return Array.isArray(images)
            ? images
                  .filter(
                      (image) =>
                          typeof image?.id === "string" &&
                          /^[a-f0-9]{64}$/.test(image.id) &&
                          typeof image?.name === "string",
                  )
                  .slice(0, 4)
            : [];
    } catch {
        return [];
    }
}

export function useAttachments(conversation: string, preferences: Preferences) {
    const key = `images:${conversation || "new"}`;
    const images = read(preferences.views[key]);
    const current = useRef(preferences.views);
    current.current = preferences.views;
    const picking = useRef(false);
    const [busy, setBusy] = useState(false);
    const { notify } = useNotifications();
    const add = async () => {
        if (!window.arxDesktop || picking.current) return;
        picking.current = true;
        setBusy(true);
        try {
            const selected = await window.arxDesktop.chooseImages();
            if (!selected.length) return;
            const held = read(current.current[key]);
            const merged = [
                ...new Map(
                    [...held, ...selected].map((image) => [image.id, image]),
                ).values(),
            ];
            if (merged.length > 4) throw new Error("Attach up to 4 images per message");
            await preferences.saveView(key, JSON.stringify(merged));
        } catch (error) {
            notify(
                error instanceof Error ? error.message : "Couldn’t attach images",
                "attach-images",
            );
        } finally {
            picking.current = false;
            setBusy(false);
        }
    };
    const remove = (id: string) => {
        void preferences.saveView(
            key,
            JSON.stringify(images.filter((image) => image.id !== id)),
        );
    };
    const sent = async (from: string, sentImages: ImageAttachment[]) => {
        const savedKey = `images:${from || "new"}`;
        const sentIDs = new Set(sentImages.map((image) => image.id));
        await preferences.saveView(
            savedKey,
            JSON.stringify(
                read(current.current[savedKey]).filter((image) => !sentIDs.has(image.id)),
            ),
        );
    };
    return { images, busy, add, remove, sent };
}
