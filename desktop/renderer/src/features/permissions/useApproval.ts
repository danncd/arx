import { useEffect, useRef, useState } from "react";
import type { BackendStatus } from "../../../../contracts/backend";
import type { PermissionRequest } from "../../../../contracts/wire.generated";
import { useNotifications } from "../../ui/notifications/Notifications";

export function useApproval(status: BackendStatus) {
    const [request, setRequest] = useState<PermissionRequest | null>(null);
    const [responding, setResponding] = useState(false);
    const version = useRef(0);
    const { notify } = useNotifications();
    useEffect(() => {
        const desktop = window.arxDesktop;
        if (!desktop || status.state !== "ready") {
            setRequest(null);
            return;
        }
        let active = true;
        const unsubscribe = desktop.onPermission((value) => {
            version.current++;
            setRequest(value);
            setResponding(false);
        });
        const current = version.current;
        void desktop
            .request("permissions.pending", undefined)
            .then((value) => {
                if (active && current === version.current) setRequest(value.pending);
            })
            .catch(() => {});
        return () => {
            active = false;
            unsubscribe();
        };
    }, [status.state]);
    const respond = async (allow: boolean) => {
        if (!request || responding) return;
        setResponding(true);
        try {
            await window.arxDesktop?.request("permissions.respond", {
                id: request.id,
                allow,
            });
        } catch (error) {
            notify(
                error instanceof Error ? error.message : "Couldn’t respond to approval",
                "permission-response",
            );
        } finally {
            setResponding(false);
        }
    };
    return { request, responding, respond };
}
