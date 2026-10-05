import { useState } from "react";
import { DownloadSimpleIcon } from "@phosphor-icons/react";
import type { MediaArtifact } from "../../../../../contracts/wire.generated";
import type { ToolRecord } from "../../../../../contracts/wire.generated";
import { AudioPlayer } from "./AudioPlayer";
import { VideoPlayer } from "./VideoPlayer";
import "./media.css";

function artifacts(tools: ToolRecord[]): MediaArtifact[] {
    const results = new Map<string, MediaArtifact>();
    for (const tool of tools) {
        try {
            const output = JSON.parse(tool.result);
            if (!Array.isArray(output.artifacts)) continue;
            for (const artifact of output.artifacts) {
                if (
                    /^[a-f0-9]{32}$/.test(artifact.id) &&
                    ["video/mp4", "audio/wav"].includes(artifact.mime)
                )
                    results.set(artifact.id, artifact);
            }
        } catch {}
    }
    return [...results.values()];
}

export function MediaResult({ tools }: { tools: ToolRecord[] }) {
    const [error, setError] = useState("");
    const outputs = artifacts(tools);
    if (!outputs.length) return null;
    return (
        <>
            <div className="generated-media-list">
                {outputs.map((artifact) => {
                    const source = `arx-media://artifact/${artifact.id}`;
                    return (
                        <div className="generated-media" key={artifact.id}>
                            {artifact.mime === "audio/wav" ? (
                                <AudioPlayer artifact={artifact} source={source} />
                            ) : (
                                <>
                                    <VideoPlayer source={source} />
                                    <div className="generated-caption">
                                        <span>{artifact.name}</span>
                                        <button
                                            type="button"
                                            className="text-button"
                                            onClick={() =>
                                                void window.arxDesktop
                                                    ?.saveMedia(artifact.id)
                                                    .catch(() =>
                                                        setError("Could not save media"),
                                                    )
                                            }
                                        >
                                            <DownloadSimpleIcon size={14} />
                                            Save
                                        </button>
                                    </div>
                                </>
                            )}
                        </div>
                    );
                })}
            </div>
            {error && <p role="alert">{error}</p>}
        </>
    );
}
