import { NetworkModels } from "../models/providers/network/NetworkModels";
import type { NetworkModelsState } from "../models/providers/network/useNetworkModels";
import type { GenerationCategory } from "../../../../contracts/wire.generated";
import type { DeepSeekConnection } from "../models/providers/deepseek/useConnection";
import type { LocalModelsState } from "../models/providers/local/useLocalModels";
import { DeepSeekConnectionSection } from "../models/providers/deepseek/DeepSeekConnection";
import { LocalModels } from "../models/providers/local/LocalModels";
import { ModelCatalog } from "../models/catalog/ModelCatalog";
import type { Preferences } from "../../platform/preferences/usePreferences";

export function Connections({
    network,
    connection,
    local,
    running,
    browsing,
    onBrowse,
    preferences,
}: {
    network: NetworkModelsState;
    connection: DeepSeekConnection;
    local: LocalModelsState;
    running: boolean;
    browsing: GenerationCategory | "chat" | null;
    onBrowse: (category: GenerationCategory | "chat" | null) => void;
    preferences: Preferences;
}) {
    if (browsing)
        return (
            <ModelCatalog
                local={local}
                running={running}
                initialCategory={browsing}
                onBack={() => onBrowse(null)}
            />
        );
    return (
        <section aria-label="Connections">
            <h3>Connections</h3>
            <DeepSeekConnectionSection connection={connection} />
            <LocalModels
                local={local}
                idleMinutes={preferences.settings.local_idle_minutes || 5}
                onIdleMinutesChange={(minutes) =>
                    void preferences.configureLocalIdle(minutes)
                }
                running={running}
                onBrowse={() => onBrowse("chat")}
            />
            <NetworkModels network={network} />
        </section>
    );
}
