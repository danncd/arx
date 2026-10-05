import { modelProvider } from "./selection";
import { CpuIcon, GlobeIcon } from "@phosphor-icons/react";

const deepseek = new URL("../providers/deepseek/deepseek.svg", import.meta.url).href;

export function ModelSourceIcon({ model }: { model: string }) {
    if (!model) return null;
    const provider = modelProvider(model);
    let icon = <img src={deepseek} alt="" />;
    if (provider === "network") icon = <GlobeIcon />;
    if (provider === "local") icon = <CpuIcon />;
    return (
        <span className="model-source-icon" aria-hidden="true">
            {icon}
        </span>
    );
}
