import { PowerIcon } from "@phosphor-icons/react";

export function ModelLoadIndicator() {
    return (
        <PowerIcon
            size={13}
            className="model-unloaded"
            aria-label="Not loaded — loads when you send a message"
            role="img"
        />
    );
}
