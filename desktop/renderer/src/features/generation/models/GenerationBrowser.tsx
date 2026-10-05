import { BrowserSearch } from "../../models/catalog/BrowserSearch";
import { BrowserFilters } from "../../models/providers/local/browser/BrowserFilters";
import { type ReactNode, useState } from "react";
import type { GenerationCategory } from "../../../../../contracts/wire.generated";
import { GenerationModelRow } from "./GenerationModelRow";
import type { Generation } from "../state/useGeneration";

export function GenerationBrowser({
    generation,
    category,
    navigation,
    hardware,
    running,
}: {
    generation: Generation;
    category: GenerationCategory;
    navigation: ReactNode;
    hardware: { name: string; memory: number; supported: boolean };
    running: boolean;
}) {
    const [recommended, setRecommended] = useState(false);
    const [query, setQuery] = useState("");
    const [sort, setSort] = useState("name");
    const [type, setType] = useState("all");
    const models = generation.catalog
        .filter(
            (model) =>
                model.category === category &&
                (type === "all" || model.operations.includes(type)) &&
                (!recommended || hardware.memory >= model.minimumMemory) &&
                `${model.name} ${model.company}`
                    .toLowerCase()
                    .includes(query.toLowerCase()),
        )
        .sort((a, b) =>
            sort === "size" ? a.size - b.size : a.name.localeCompare(b.name),
        );
    return (
        <section aria-label="Browse generation models">
            {navigation}
            <BrowserSearch hardware={hardware} query={query} onQuery={setQuery} />
            <BrowserFilters
                recommended={recommended}
                onRecommended={setRecommended}
                sort={sort}
                onSort={setSort}
                type={type}
                onType={setType}
                typeChoices={[
                    { value: "all", label: "All types" },
                    ...(category === "image"
                        ? [{ value: "image-edit", label: "Image editing" }]
                        : category === "video"
                          ? [{ value: "image-to-video", label: "Image to video" }]
                          : []),
                ]}
                sortChoices={[
                    { value: "name", label: "Name" },
                    { value: "size", label: "Smallest download" },
                ]}
            />
            {generation.error && (
                <p role="alert" className="local-error">
                    {generation.error}
                </p>
            )}
            {generation.loading && (
                <div className="local-empty" role="status">
                    Loading models…
                </div>
            )}
            {models.map((model) => (
                <GenerationModelRow
                    key={model.id}
                    model={model}
                    generation={generation}
                    memory={hardware.memory}
                    supported={hardware.supported}
                    running={running}
                />
            ))}
            {!models.length && !generation.loading && (
                <div className="local-empty">No matching models.</div>
            )}
            <div className="model-pagination">
                <span>{models.length} models shown</span>
            </div>
        </section>
    );
}
