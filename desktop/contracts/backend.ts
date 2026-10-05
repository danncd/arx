export type BackendStatus = {
    state: "starting" | "ready" | "stopping" | "stopped" | "error";
    error: string | null;
};

export type BackendInfo = {
    version: string;
    execution: "idle" | "running" | "stopping";
};
