import { useEffect, useState } from "react";
import type { BackendStatus } from "../../../../contracts/backend";

export function useBackend() {
    const [status, setStatus] = useState<BackendStatus>({
        state: "starting",
        error: null,
    });
    useEffect(() => {
        const desktop = window.arxDesktop;
        if (!desktop) {
            setStatus({ state: "stopped", error: null });
            return;
        }
        return desktop.onBackendStatus(setStatus);
    }, []);
    const restart = async () => {
        try {
            await window.arxDesktop?.restartBackend();
        } catch {
            setStatus({ state: "error", error: "Could not restart backend" });
        }
    };
    return { status, restart };
}
