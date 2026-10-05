import { useRef, useState } from "react";
import { DownloadSimpleIcon, PauseIcon, PlayIcon } from "@phosphor-icons/react";
import "./audio.css";
import type { MediaArtifact } from "../../../../../contracts/wire.generated";

function time(seconds: number) {
    return `${Math.floor(seconds / 60)}:${String(Math.floor(seconds % 60)).padStart(2, "0")}`;
}

export function AudioPlayer({
    artifact,
    source,
}: {
    artifact: MediaArtifact;
    source: string;
}) {
    const audio = useRef<HTMLAudioElement>(null);
    const [playing, setPlaying] = useState(false);
    const [position, setPosition] = useState(0);
    const [duration, setDuration] = useState(artifact.duration || 0);
    const [error, setError] = useState("");
    return (
        <div className="speech-result">
            <div className="speech-player">
                <button
                    type="button"
                    className="speech-play"
                    aria-label={playing ? "Pause narration" : "Play narration"}
                    onClick={async () => {
                        if (!audio.current) return;
                        if (playing) audio.current.pause();
                        else {
                            try {
                                await audio.current.play();
                                setError("");
                            } catch {
                                setError("Unable to play this audio.");
                            }
                        }
                    }}
                >
                    {playing ? (
                        <PauseIcon size={17} weight="fill" />
                    ) : (
                        <PlayIcon size={17} weight="fill" />
                    )}
                </button>
                <input
                    type="range"
                    aria-label="Narration position"
                    min={0}
                    max={duration || 1}
                    step={0.1}
                    value={position}
                    onChange={(event) => {
                        const value = Number(event.target.value);
                        if (audio.current) audio.current.currentTime = value;
                        setPosition(value);
                    }}
                />
                <span className="speech-time">
                    {time(position)} / {time(duration)}
                </span>
                <button
                    type="button"
                    className="speech-download"
                    aria-label="Save narration"
                    onClick={() =>
                        void window.arxDesktop
                            ?.saveMedia(artifact.id)
                            .catch(() => setError("Could not save audio"))
                    }
                >
                    <DownloadSimpleIcon size={16} />
                </button>
            </div>
            <span className="speech-caption">{error || artifact.name}</span>
            <audio
                ref={audio}
                src={source}
                preload="metadata"
                onLoadedMetadata={(event) => setDuration(event.currentTarget.duration)}
                onTimeUpdate={(event) => setPosition(event.currentTarget.currentTime)}
                onPlay={() => setPlaying(true)}
                onPause={() => setPlaying(false)}
                onEnded={() => setPlaying(false)}
                onError={() => setError("Audio is unavailable")}
            />
        </div>
    );
}
