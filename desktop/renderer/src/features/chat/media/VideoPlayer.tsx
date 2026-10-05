import { useRef, useState } from "react";
import { PlayIcon } from "@phosphor-icons/react";

export function VideoPlayer({ source }: { source: string }) {
    const video = useRef<HTMLVideoElement>(null);
    const [playing, setPlaying] = useState(false);
    const [error, setError] = useState("");
    return (
        <div className="generated-video">
            <video
                ref={video}
                src={source}
                controls
                preload="metadata"
                onPlay={() => setPlaying(true)}
                onPause={() => setPlaying(false)}
                onEnded={() => setPlaying(false)}
                onError={() => setError("Video is unavailable")}
            />
            {!playing && !error && (
                <button
                    type="button"
                    className="generated-video-play"
                    aria-label="Play video"
                    onClick={() =>
                        void video.current
                            ?.play()
                            .catch(() => setError("Could not play video"))
                    }
                >
                    <PlayIcon size={24} weight="fill" />
                </button>
            )}
            {error && <p role="alert">{error}</p>}
        </div>
    );
}
