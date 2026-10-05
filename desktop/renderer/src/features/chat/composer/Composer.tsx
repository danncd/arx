import { Attachments } from "../attachments/Attachments";
import type { useAttachments } from "../attachments/useAttachments";
import { PermissionPicker } from "../../permissions/PermissionPicker";
import type { PermissionPolicy } from "../../../../../contracts/wire.generated";
import { ArrowUpIcon, StopIcon, PlusIcon, CircleNotchIcon } from "@phosphor-icons/react";
import { useLayoutEffect, useRef } from "react";

import type {
    ModelInfo,
    ThinkingCapabilities,
} from "../../../../../contracts/wire.generated";
import type { ContextReport } from "../../../../../contracts/wire.generated";
import { ContextMeter } from "../context/ContextMeter";
import { ModelPicker } from "../../models/picker/ModelPicker";

type Props = {
    attachments: ReturnType<typeof useAttachments>;
    vision: boolean;
    permissions: PermissionPolicy;
    directory: string;
    onPermissions: (policy: PermissionPolicy) => Promise<void>;
    draft: string;
    onChange: (value: string) => void;
    onSend: () => void;
    onStop: () => void;
    onSettings: () => void;
    onModel: (model: string) => Promise<boolean | undefined>;
    effort: string;
    capabilities?: ThinkingCapabilities;
    onEffort: (effort: string) => Promise<void>;
    models: ModelInfo[];
    unloadedModels?: string[];
    catalogueError: string;
    model: string;
    running: boolean;

    starting: boolean;
    loadingModel?: boolean;
    cancelling: boolean;
    disabled: boolean;
    ready: boolean;

    contextConversation: string;
    contextReport: ContextReport | null;
};

export function Composer(props: Props) {
    const canSend =
        !props.attachments.busy &&
        (Boolean(props.draft.trim()) || props.attachments.images.length > 0) &&
        (!props.attachments.images.length || props.vision);
    const showStop = props.running || props.starting || Boolean(props.loadingModel);
    const loading = !props.running && (Boolean(props.loadingModel) || props.starting);
    const actionLabel = loading
        ? props.loadingModel
            ? "Loading model — click to cancel"
            : "Sending message"
        : showStop
          ? props.cancelling
              ? "Stopping reply"
              : "Stop reply"
          : "Send message";
    const input = useRef<HTMLTextAreaElement>(null);
    const resizeInput = () => {
        const element = input.current;
        if (!element) return;
        element.style.height = "auto";
        element.style.height = `${element.scrollHeight}px`;
    };
    useLayoutEffect(resizeInput, [props.draft]);
    useLayoutEffect(() => {
        const element = input.current;
        if (!element) return;
        let width = element.clientWidth;
        let frame = 0;
        const observer = new ResizeObserver(() => {
            if (element.clientWidth === width) return;
            width = element.clientWidth;
            cancelAnimationFrame(frame);
            frame = requestAnimationFrame(resizeInput);
        });
        observer.observe(element);
        return () => {
            observer.disconnect();
            cancelAnimationFrame(frame);
        };
    }, []);

    return (
        <div className="composer-area">
            <form
                className="composer"
                onSubmit={(event) => {
                    event.preventDefault();
                }}
            >
                <Attachments
                    images={props.attachments.images}
                    onRemove={props.starting ? undefined : props.attachments.remove}
                />
                <textarea
                    ref={input}
                    autoFocus
                    aria-label="Message"
                    placeholder="Message Arx…"
                    value={props.draft}
                    onChange={(event) => props.onChange(event.target.value)}
                    onKeyDown={(event) => {
                        if (
                            event.key === "Enter" &&
                            !event.shiftKey &&
                            !event.nativeEvent.isComposing &&
                            event.keyCode !== 229
                        ) {
                            event.preventDefault();

                            if (
                                props.ready &&
                                !props.disabled &&
                                !props.running &&
                                !props.starting &&
                                !props.loadingModel &&
                                canSend
                            )
                                props.onSend();
                        }
                    }}
                    rows={2}
                    spellCheck={false}
                />
                <div className="composer-controls">
                    <button
                        type="button"
                        className="attach-button"
                        aria-label="Attach images"
                        title={
                            props.vision
                                ? "Attach images"
                                : "Select a vision-capable model to attach images"
                        }
                        disabled={
                            props.disabled ||
                            !props.vision ||
                            props.starting ||
                            props.attachments.busy ||
                            props.attachments.images.length >= 4
                        }
                        onClick={() => void props.attachments.add()}
                    >
                        <PlusIcon size={19} />
                    </button>
                    <PermissionPicker
                        policy={props.permissions}
                        directory={props.directory}
                        disabled={props.disabled}
                        onChange={props.onPermissions}
                    />
                    <ModelPicker
                        models={props.models}
                        unloadedModels={props.unloadedModels}
                        capabilities={props.capabilities}
                        catalogueError={props.catalogueError}
                        model={props.model}
                        disabled={
                            props.disabled ||
                            props.starting ||
                            Boolean(props.loadingModel)
                        }
                        onModel={props.onModel}
                        effort={props.effort}
                        onEffort={props.onEffort}
                        onSettings={props.onSettings}
                    />
                    <ContextMeter
                        report={props.contextReport}
                        conversation={props.contextConversation}
                    />
                    <button
                        type="button"
                        className={`send-button${showStop ? " stop-button" : ""}`}
                        aria-label={actionLabel}
                        aria-busy={loading}
                        title={actionLabel}
                        disabled={
                            showStop
                                ? props.disabled || props.cancelling || props.starting
                                : props.disabled ||
                                  !props.ready ||
                                  !canSend ||
                                  props.running ||
                                  props.starting
                        }
                        onClick={showStop ? props.onStop : props.onSend}
                    >
                        {loading ? (
                            <CircleNotchIcon size={19} className="send-spinner" />
                        ) : showStop ? (
                            <StopIcon size={15} weight="fill" />
                        ) : (
                            <ArrowUpIcon size={19} weight="bold" />
                        )}
                    </button>
                </div>
            </form>
        </div>
    );
}
