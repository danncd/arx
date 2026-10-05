import { useState } from "react";
import { ArrowLeftIcon } from "@phosphor-icons/react";
import "./catalog.css";
import type { GenerationCategory } from "../../../../../contracts/wire.generated";
import { GenerationBrowser } from "../../generation/models/GenerationBrowser";
import { useGeneration } from "../../generation/state/useGeneration";
import { ModelBrowser } from "../providers/local/browser/ModelBrowser";
import type { LocalModelsState } from "../providers/local/useLocalModels";
import "../../generation/settings/generation.css";

export function ModelCatalog({
    local,
    running,
    onBack,
    initialCategory = "chat",
}: {
    local: LocalModelsState;
    running: boolean;
    onBack: () => void;
    initialCategory?: GenerationCategory | "chat";
}) {
    const [category, setCategory] = useState(initialCategory);
    const generation = useGeneration();
    const navigation = (
        <>
            <div className="model-catalog-heading">
                <button
                    type="button"
                    className="icon-button"
                    aria-label="Back to settings"
                    onClick={onBack}
                >
                    <ArrowLeftIcon size={16} />
                </button>
                <h3>Browse models</h3>
                <span>Hugging Face</span>
            </div>
            <div className="generation-tabs" aria-label="Model categories">
                {(["chat", "image", "video", "speech"] as const).map((value) => (
                    <button
                        type="button"
                        key={value}
                        aria-pressed={category === value}
                        onClick={() => setCategory(value)}
                    >
                        {
                            {
                                chat: "Chat",
                                image: "Images",
                                video: "Videos",
                                speech: "Speech",
                            }[value]
                        }
                    </button>
                ))}
            </div>
        </>
    );
    return category === "chat" ? (
        <ModelBrowser local={local} running={running} navigation={navigation} />
    ) : (
        <GenerationBrowser
            key={category}
            generation={generation}
            category={category}
            navigation={navigation}
            hardware={local.hardware}
            running={running}
        />
    );
}
