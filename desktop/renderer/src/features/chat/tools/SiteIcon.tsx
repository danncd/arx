import { useEffect, useState } from "react";
import { GlobeIcon } from "@phosphor-icons/react";

import { useResourcePolicy } from "../../../rich-text/ResourcePolicy";

export function SiteIcon({ url }: { url: string }) {
    const { conversation, mode } = useResourcePolicy();
    const [icon, setIcon] = useState("");
    useEffect(() => {
        let active = true;
        setIcon("");
        if (!conversation || mode === "ask") return;
        void window.arxDesktop
            ?.request("web.icon", { url, conversation })
            .then((value) => {
                if (
                    active &&
                    /^data:image\/(png|jpeg|gif|webp|x-icon|vnd.microsoft.icon);base64,/.test(
                        value,
                    )
                )
                    setIcon(value);
            })
            .catch(() => {});
        return () => {
            active = false;
        };
    }, [url, conversation, mode]);
    return icon ? (
        <img className="web-site-icon" src={icon} alt="" onError={() => setIcon("")} />
    ) : (
        <GlobeIcon size={16} className="web-site-icon" />
    );
}
