import { ArrowUpRightIcon } from "@phosphor-icons/react";
import { Select } from "../../../ui/select/Select";
import type { GenerationCategory } from "../../../../../contracts/wire.generated";
import { useGeneration } from "../state/useGeneration";
import "./generation.css";

export function GenerationSettings({
    onBrowse,
}: {
    onBrowse: (category: GenerationCategory) => void;
}) {
    const generation = useGeneration();
    return (
        <section aria-label="Generation settings">
            <h3>Generation</h3>
            <p className="generation-intro">
                Choose the models Arx uses to create media.
            </p>
            {generation.error && (
                <p role="alert" className="generation-error">
                    {generation.error}
                </p>
            )}
            {generation.loading ? (
                <p>Loading models…</p>
            ) : (
                (["image", "video", "speech"] as const).map((category) => (
                    <div className="generation-default" key={category}>
                        <div>
                            <strong>
                                {
                                    {
                                        image: "Images",
                                        video: "Videos",
                                        speech: "Speech",
                                    }[category]
                                }
                            </strong>
                            <span>
                                {
                                    {
                                        image: "Generate and edit images",
                                        video: "Text or images to video",
                                        speech: "Text to speech",
                                    }[category]
                                }
                            </span>
                        </div>
                        <div className="generation-default-controls">
                            <Select
                                label={`${category} model`}
                                value={generation.library.defaults[category] || ""}
                                choices={[
                                    { value: "", label: "Not configured" },
                                    ...generation.library.models
                                        .filter(
                                            (entry) =>
                                                entry.model.category === category &&
                                                entry.status === "installed",
                                        )
                                        .map((entry) => ({
                                            value: entry.model.id,
                                            label: entry.model.name,
                                        })),
                                ]}
                                onChange={(id) => void generation.configure(category, id)}
                            />
                            <button
                                type="button"
                                className="local-button generation-browse-icon"
                                aria-label={`Browse ${category} models`}
                                title={`Browse ${category} models`}
                                onClick={() => onBrowse(category)}
                            >
                                <ArrowUpRightIcon size={16} />
                            </button>
                        </div>
                    </div>
                ))
            )}
            <div className="generation-default">
                <div>
                    <strong>Local memory</strong>
                    <span>Release unused models between jobs</span>
                </div>
                <span>Automatic</span>
            </div>
            <div className="generation-default">
                <div>
                    <strong>Output folder</strong>
                    <span>Saved with each conversation</span>
                </div>
                <span>Arx / Media</span>
            </div>
        </section>
    );
}
