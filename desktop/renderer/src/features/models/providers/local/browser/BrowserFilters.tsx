import { Select } from "../../../../../ui/select/Select";

export function BrowserFilters({
    recommended,
    onRecommended,
    sort,
    onSort,
    type,
    onType,
    typeChoices,
    sortChoices,
}: {
    recommended: boolean;
    onRecommended: (value: boolean) => void;
    sort: string;
    onSort: (value: string) => void;
    typeChoices?: { value: string; label: string }[];
    sortChoices?: { value: string; label: string }[];
    type: string;
    onType: (value: string) => void;
}) {
    return (
        <div className="local-browser-filters">
            <label>
                <input
                    type="checkbox"
                    checked={recommended}
                    onChange={(event) => onRecommended(event.target.checked)}
                />
                Recommended for this Mac
            </label>
            <div className="model-filter-selects">
                <Select
                    label="Model type"
                    value={type}
                    onChange={onType}
                    choices={
                        typeChoices || [
                            { value: "all", label: "All types" },
                            { value: "text", label: "Text" },
                            { value: "vision", label: "Vision" },
                            { value: "thinking", label: "Thinking" },
                        ]
                    }
                />
                <Select
                    label="Sort models"
                    value={sort}
                    onChange={onSort}
                    choices={
                        sortChoices || [
                            { value: "downloads", label: "Most downloaded" },
                            { value: "trendingScore", label: "Trending" },
                            { value: "likes", label: "Most liked" },
                            { value: "lastModified", label: "Recently updated" },
                        ]
                    }
                />
            </div>
        </div>
    );
}
