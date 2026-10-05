import type { ImageAttachment } from "../../contracts/wire.generated";
import type { BackendInfo, BackendStatus } from "../../contracts/backend";

import type { PermissionRequest } from "../../contracts/wire.generated";
import type { ChatEvent } from "../../contracts/wire.generated";
import type { Requests } from "../../contracts/requests.generated";

export {};

declare global {
    interface Window {
        arxDesktop?: {
            request<K extends keyof Requests>(
                method: K,
                params: Requests[K]["params"],
            ): Promise<Requests[K]["result"]>;
            onChat(callback: (event: ChatEvent) => void): () => void;
            onPermission(
                callback: (request: PermissionRequest | null) => void,
            ): () => void;
            onGeneration(
                callback: (
                    event: import("../../contracts/wire.generated").GenerationEvent,
                ) => void,
            ): () => void;
            saveMedia(id: string): Promise<boolean>;
            onLocal(
                callback: (
                    state: import("../../contracts/wire.generated").LocalState,
                ) => void,
            ): () => void;
            chooseModel(): Promise<string | null>;
            chooseImages(): Promise<ImageAttachment[]>;
            readImage(id: string): Promise<string>;
            chooseDirectory(): Promise<string | null>;
            backendInfo(): Promise<BackendInfo>;
            restartBackend(): Promise<BackendStatus>;
            onBackendStatus(callback: (status: BackendStatus) => void): () => void;
            copy(text: string): Promise<void>;
            openExternal(url: string): Promise<void>;
            beginWindowDrag(point: { x: number; y: number }): void;
            moveWindowDrag(point: { x: number; y: number }): void;
            endWindowDrag(): void;
            doubleClickTitleBar(): Promise<void>;
            onFullscreenChanged(callback: (fullscreen: boolean) => void): () => void;
            signalReady(): void;
        };
    }
}
