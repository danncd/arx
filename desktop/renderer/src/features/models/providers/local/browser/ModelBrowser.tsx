import { BrowserFilters } from "./BrowserFilters";
import { catalogCapabilities } from "../capabilities";
import "./browser.css";
import { BrowserSearch } from "../../../catalog/BrowserSearch";
import { type ReactNode, useCallback, useState } from "react";
import type { LocalModelsState } from "../useLocalModels";
import { useModelSearch } from "./useModelSearch";
import { RepositoryRow } from "./RepositoryRow";

export function ModelBrowser({
    local,
    running,
    navigation,
}: {
    local: LocalModelsState;
    running: boolean;
    navigation: ReactNode;
}) {
    const [query, setQuery] = useState("");
    const [recommended, setRecommended] = useState(false);
    const [sort, setSort] = useState("downloads");
    const [type, setType] = useState("all");
    const [matches, setMatches] = useState<Record<string, boolean | undefined>>({});
    const reportMatch = useCallback(
        (id: string, match: boolean | undefined) =>
            setMatches((previous) =>
                previous[id] === match ? previous : { ...previous, [id]: match },
            ),
        [],
    );
    const search = useModelSearch(query, sort);
    const models = search.models.filter((model) => {
        const capabilities = catalogCapabilities(model);
        return type === "all" || capabilities[type as keyof typeof capabilities];
    });
    const visible = models.filter((model) => !recommended || matches[model.id]).length;
    const checking =
        recommended && models.some((model) => matches[model.id] === undefined);
    return (
        <section aria-label="Browse models">
            {navigation}
            <BrowserSearch hardware={local.hardware} query={query} onQuery={setQuery} />
            <BrowserFilters
                recommended={recommended}
                onRecommended={setRecommended}
                sort={sort}
                onSort={setSort}
                type={type}
                onType={setType}
            />
            {local.error && (
                <p className="local-error" role="alert">
                    {local.error}
                </p>
            )}
            {search.busy && (
                <div className="local-empty" role="status">
                    Searching…
                </div>
            )}
            {search.error && (
                <p className="local-error" role="alert">
                    {search.error}
                </p>
            )}
            {!search.busy && !search.error && !visible && !checking && (
                <div className="local-empty">No matching models in these results.</div>
            )}
            {!search.busy && checking && !visible && (
                <div className="local-empty" role="status">
                    Checking compatibility…
                </div>
            )}
            {!search.busy &&
                models.map((model) => (
                    <RepositoryRow
                        key={model.id}
                        model={model}
                        local={local}
                        recommended={recommended}
                        running={running}
                        onMatch={reportMatch}
                    />
                ))}
            {!search.busy && (
                <div className="model-pagination">
                    <span>{visible} models shown</span>
                    {search.next && (
                        <button
                            className="local-button"
                            type="button"
                            disabled={search.moreBusy}
                            onClick={() => void search.loadMore()}
                        >
                            {search.moreBusy ? "Loading…" : "Load more"}
                        </button>
                    )}
                </div>
            )}
        </section>
    );
}
