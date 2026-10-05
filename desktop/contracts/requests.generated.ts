import type * as Wire from "./wire.generated";
export type Requests = {
    "chat.context": {
        params: { conversation: string; model: string };
        result: Wire.ContextReport;
    };
    "chat.send": {
        params: {
            images?: Array<Wire.ImageAttachment>;
            permissions?: Wire.PermissionPolicy;
            id: string;
            conversation: string;
            text: string;
        };
        result: Wire.ChatRun;
    };
    "chat.state": { params: undefined; result: Wire.ChatEvent };
    "chat.stop": { params: { id: string }; result: { accepted: boolean } };
    configure: {
        params: {
            run: Wire.RunSettings;
            auto_continue?: boolean;
            directory: string;
        };
        result: Wire.SavedSettings;
    };
    "deepseek.connect": { params: { key: string }; result: Wire.Connection };
    "deepseek.disconnect": { params: undefined; result: Wire.Connection };
    "deepseek.refresh": { params: undefined; result: Wire.Connection };
    "deepseek.status": { params: undefined; result: Wire.Connection };
    "generation.action": {
        params: { action: string; id: string };
        result: Wire.GenerationLibrary | boolean;
    };
    "generation.configure": {
        params: { category: string; id: string };
        result: Wire.GenerationLibrary;
    };
    "generation.state": { params: undefined; result: Wire.GenerationState };
    history: {
        params: { conversation: string; before?: string; limit?: number };
        result: Wire.HistoryPage;
    };
    "local.action": {
        params: { action: string; id: string; deleteFiles?: boolean };
        result: Wire.LocalState;
    };
    "local.configure": {
        params: { idle_minutes: number };
        result: Wire.SavedSettings;
    };
    "local.download": {
        params: { repository: string; variant: string };
        result: { id: string };
    };
    "local.import": {
        params: { path: string; projector?: string };
        result: { id: string };
    };
    "local.repository": { params: { id: string }; result: Wire.Repository };
    "local.search": {
        params: { query: string; sort?: string; cursor?: string };
        result: Wire.SearchPage;
    };
    "local.state": { params: undefined; result: Wire.LocalState };
    "mcp.remove": { params: { id: string }; result: Array<Wire.MCPServer> };
    "mcp.save": {
        params: Wire.MCPConfiguration;
        result: Array<Wire.MCPServer>;
    };
    "mcp.state": { params: undefined; result: Array<Wire.MCPServer> };
    "mcp.test": {
        params: { id: string; configuration?: Wire.MCPConfiguration };
        result: Array<Wire.MCPServer>;
    };
    "network.cancel": { params: undefined; result: boolean };
    "network.connect": {
        params: { url: string; name: string; token: string };
        result: Wire.NetworkServer;
    };
    "network.remove": { params: { id: string }; result: boolean };
    "network.scan": { params: undefined; result: Array<Wire.NetworkServer> };
    "network.state": { params: undefined; result: Array<Wire.NetworkServer> };
    "permissions.configure": {
        params: {
            mode: Wire.PermissionMode;
            roots: Array<string>;
            conversation?: string;
        };
        result: Wire.SavedSettings;
    };
    "permissions.pending": {
        params: undefined;
        result: { pending: Wire.PermissionRequest | null };
    };
    "permissions.respond": {
        params: { id: string; allow: boolean };
        result: Record<string, never>;
    };
    save_view: {
        params: { key: string; value: string };
        result: Record<string, never>;
    };
    "skills.detail": { params: { id: string }; result: Wire.SkillDetail };
    "skills.read": { params: { id: string; path: string }; result: string };
    "skills.remove": { params: { id: string }; result: Array<Wire.Skill> };
    "skills.save": { params: Wire.SkillEdit; result: Array<Wire.Skill> };
    "skills.state": { params: undefined; result: Array<Wire.Skill> };
    "skills.validate": {
        params: { instructions: string };
        result: { name: string; description: string };
    };
    snapshot: { params: undefined; result: Wire.Snapshot };
    status: {
        params: undefined;
        result: { version: string; execution: string };
    };
    "tools.cancel": { params: { id: string }; result: { accepted: boolean } };
    "tools.definitions": {
        params: undefined;
        result: Array<{
            name: string;
            description: string;
            parameters: unknown;
        }>;
    };
    "tools.run": {
        params: {
            id: string;
            name: string;
            arguments: unknown;
            conversation?: string;
        };
        result: Wire.ToolResult;
    };
    "web.icon": {
        params: { url: string; conversation: string };
        result: string;
    };
    "web.image": {
        params: { url: string; conversation: string };
        result: string;
    };
};
