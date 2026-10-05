import type { DiscoveredModel } from "../../../../contracts/wire.generated";
import type { NetworkServer } from "../../../../contracts/wire.generated";
import type { LocalModel } from "../../../../contracts/wire.generated";
export function availableModels(
    cloud: DiscoveredModel[],
    network: NetworkServer[],
    local: LocalModel[],
): DiscoveredModel[] {
    return [
        ...cloud,
        ...network
            .filter((server) => server.connected)
            .flatMap((server) =>
                server.models
                    .filter((model) => model.loaded)
                    .map((model) => ({
                        ...model.info,
                        name: `${model.info.name} · ${server.name}`,
                    })),
            ),
        ...local
            .filter((item) => ["installed", "loaded", "loading"].includes(item.status))
            .map((item) => item.info),
    ];
}
