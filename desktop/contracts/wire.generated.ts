export type GenerationCategory = "image" | "video" | "speech";
export type ModelInfo = Pick<DiscoveredModel, "id" | "name">;
export type ChatEvent = {
    recovery?: { kind: string; attempt: number; limit: number };
    compacting?: boolean;
    revision: number;
    run: ChatRun;
    message?: TranscriptChunk;
    conversation?: Conversation;
    error?: string;
};
export type ChatRun = { id: string; conversation: string; state: ChatState };
export type ChatState = "idle" | "running" | "stopping";
export type ChunkUsage = {
    generation?: GenerationUsage;
    cacheUnknown?: boolean;
    input: number;
    cached: number;
    output: number;
    reasoning: number;
};
export type Connection = {
    configured: boolean;
    connected: boolean;
    models: Array<DiscoveredModel>;
    error: string;
};
export type ContextParts = {
    system: number;
    tools: number;
    messages: number;
    summary: number;
};
export type ContextReport = {
    usage: ChunkUsage;
    parts: ContextParts;
    known: boolean;
    used: number;
    limit: number;
    basis: string;
};
export type Conversation = {
    id: string;
    title: string;
    updated: string;
    status: string;
};
export type DiscoveredModel = {
    contextReason?: string;
    provider?: "deepseek" | "local" | "network";
    tools?: boolean;
    trainedContext?: number;
    capabilitySource?: string;
    vision: boolean;
    id: string;
    name: string;
    contextWindow: number;
    maxOutputTokens: number;
    thinking?: ThinkingCapabilities;
};
export type GenerationEntry = {
    model: GenerationModel;
    status: string;
    received: number;
    error?: string;
};
export type GenerationEvent = {
    library?: GenerationLibrary;
    job?: GenerationJob;
};
export type GenerationJob = {
    id: string;
    conversation: string;
    toolCall: string;
    operation: string;
    model: string;
    state: string;
    progress: number;
    detail: string;
    error?: string;
    output?: MediaArtifact;
    updated: string;
};
export type GenerationLibrary = {
    revision: number;
    models: Array<GenerationEntry>;
    defaults: Record<string, string>;
};
export type GenerationModel = {
    capabilities?: {
        steps: number;
        defaultFrames: number;
        maxFrames: number;
        maxPixels: number;
        maxReferences: number;
        fixedSpeed: boolean;
    };
    experimental?: boolean;
    runtime: string;
    id: string;
    name: string;
    company: string;
    category: GenerationCategory;
    repository: string;
    revision: string;
    operations: Array<string>;
    voices: Array<string>;
    minimumMemory: number;
    files: Array<{ name: string; url: string; size: number; sha256?: string }>;
    size: number;
};
export type GenerationState = {
    library: GenerationLibrary;
    jobs: Array<GenerationJob>;
    catalog: Array<GenerationModel>;
};
export type GenerationUsage = { tokens: number; seconds: number };
export type HistoryPage = {
    chunks: Array<TranscriptChunk>;
    before: string;
    more: boolean;
};
export type ImageAttachment = { id: string; name: string };
export type IntegrationActivity = {
    time: string;
    operation: string;
    conversation?: string;
    status: string;
};
export type IntegrationUsage = {
    count: number;
    lastUsed?: string;
    activity: Array<IntegrationActivity>;
};
export type LocalModel = {
    id: string;
    name: string;
    repository?: string;
    revision?: string;
    quantization?: string;
    files?: Array<{ name: string; url: string; size: number; sha256?: string }>;
    path: string;
    projector?: string;
    imported: boolean;
    size: number;
    status: string;
    received: number;
    error?: string;
    info: DiscoveredModel;
};
export type LocalState = {
    revision: number;
    hardware: { name: string; memory: number; supported: boolean };
    models: Array<LocalModel>;
    runtime: {
        state: string;
        model?: string;
        received: number;
        total: number;
        error?: string;
    };
    error?: string;
};
export type MCPConfiguration = {
    id: string;
    name: string;
    command: string;
    arguments: Array<string>;
    environment: Record<string, string>;
    enabled: boolean;
    policy: "ask" | "changes";
    readOnlyTools: Array<string>;
};
export type MCPServer = {
    id: string;
    name: string;
    command: string;
    arguments: Array<string>;
    environment: Record<string, string>;
    enabled: boolean;
    policy: "ask" | "changes";
    readOnlyTools: Array<string>;
    usage: IntegrationUsage;
    tools: Array<MCPTool>;
    status: string;
    error?: string;
};
export type MCPTool = {
    name: string;
    description: string;
    inputSchema: unknown;
    readOnlyHint: boolean;
    readOnly: boolean;
    calls: number;
};
export type MediaArtifact = {
    id: string;
    name: string;
    mime: "image/png" | "audio/wav" | "video/mp4";
    size: number;
    width?: number;
    height?: number;
    duration?: number;
};
export type NetworkServer = {
    id: string;
    name: string;
    url: string;
    connected: boolean;
    requiresToken: boolean;
    error?: string;
    models: Array<{ info: DiscoveredModel; key: string; loaded: boolean }>;
};
export type PermissionAction = {
    server?: string;
    arguments?: unknown;
    query?: string;
    url?: string;
    tool: string;
    operation: string;
    path?: string;
    command?: string;
    directory?: string;
    before?: string;
    after?: string;
};
export type PermissionMode = "ask" | "folders" | "full";
export type PermissionPolicy = { mode: PermissionMode; roots: Array<string> };
export type PermissionRequest = { id: string; action: PermissionAction };
export type Repository = {
    id: string;
    revision: string;
    gated: boolean;
    license?: string;
    variants: Array<Variant>;
};
export type RunSettings = {
    provider?: "deepseek" | "local" | "network";
    model: string;
    effort: string;
};
export type SavedSettings = {
    auto_continue: boolean;
    chat_permissions?: Record<string, PermissionPolicy>;
    permissions: PermissionPolicy;
    version: number;
    run: RunSettings;
    directory?: string;
    local_idle_minutes: number;
};
export type SearchModel = {
    pipeline_tag: string;
    tags: Array<string>;
    likes: number;
    id: string;
    downloads: number;
    gated?: boolean | string;
};
export type SearchPage = { models: Array<SearchModel>; next: string };
export type Skill = {
    id: string;
    name: string;
    description: string;
    path: string;
    enabled: boolean;
    activation: "auto" | "explicit";
    dependencies: Array<string>;
    usage: IntegrationUsage;
    error?: string;
};
export type SkillDetail = {
    id: string;
    name: string;
    description: string;
    path: string;
    enabled: boolean;
    activation: "auto" | "explicit";
    dependencies: Array<string>;
    usage: IntegrationUsage;
    error?: string;
    instructions: string;
    files: Array<string>;
};
export type SkillEdit = {
    id: string;
    path: string;
    instructions: string;
    enabled: boolean;
    activation: "auto" | "explicit";
    dependencies: Array<string>;
};
export type Snapshot = {
    conversations: Array<Conversation>;
    settings: SavedSettings;
    views: Record<string, string>;
    recovered: boolean;
};
export type ThinkingCapabilities = {
    efforts: Array<string>;
    defaultEffort: string;
    defaultEnabled: boolean;
    canDisable: boolean;
    source: string;
};
export type ToolRecord = {
    id: string;
    name: string;
    offset?: number;
    reasoning_offset?: number;
    status: string;
    summary: string;
    added?: number;
    removed?: number;
    arguments: string;
    result: string;
    failed: boolean;
    truncated?: boolean;
};
export type ToolResult = {
    artifacts?: Array<MediaArtifact>;
    images?: Array<ImageAttachment>;
    sources?: Array<{ url: string; title: string; snippet?: string }>;
    text: string;
    failed: boolean;
    truncated: boolean;
    exitCode?: number;
};
export type TranscriptChunk = {
    interruption?: string;
    session_title?: string;
    provider?: string;
    model?: string;
    images?: Array<ImageAttachment>;
    runtime?: { directory: string; date: string };
    provider_round?: boolean;
    context?: {
        model: string;
        effort: string;
        messages: number;
        digest: string;
        input: number;
    };
    conversation: string;
    id: string;
    role: string;
    offset: number;
    text: string;
    reasoning: string;
    reasoning_offset: number;
    tools: Array<ToolRecord> | null;
    usage: ChunkUsage | null;
    status: string;
    failed: boolean;
    reason: string;
    at: string;
};
export type Variant = {
    fit: { label: string; reason: string; tone: string };
    id: string;
    name: string;
    quantization: string;
    size: number;
    files: Array<{ name: string; url: string; size: number; sha256?: string }>;
    projector?: string;
};
