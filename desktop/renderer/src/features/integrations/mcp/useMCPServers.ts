import type {
    MCPConfiguration,
    MCPServer,
} from "../../../../../contracts/wire.generated";
import { api, useRequests } from "../shared/useRequests";
const load = () => api().request("mcp.state", undefined);
const empty: MCPServer[] = [];
export function useMCPServers() {
    const { data: servers, act, ...state } = useRequests(load, empty);
    return {
        ...state,
        servers,
        saveServer: (c: MCPConfiguration) =>
            act(() =>
                api().request("mcp.save", {
                    id: c.id,
                    name: c.name,
                    command: c.command,
                    arguments: c.arguments,
                    environment: c.environment,
                    enabled: c.enabled,
                    policy: c.policy,
                    readOnlyTools: c.readOnlyTools,
                }),
            ),
        removeServer: (id: string) => act(() => api().request("mcp.remove", { id })),
        testServer: (id: string, configuration?: MCPConfiguration) =>
            act(() => api().request("mcp.test", { id, configuration })),
    };
}
export type MCPState = ReturnType<typeof useMCPServers>;
