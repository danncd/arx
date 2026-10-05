import { CpuIcon, MagnifyingGlassIcon } from "@phosphor-icons/react";

export function BrowserSearch({
    hardware,
    query,
    onQuery,
}: {
    hardware: { name: string; memory: number };
    query: string;
    onQuery: (query: string) => void;
}) {
    return (
        <>
            <div className="local-machine">
                <CpuIcon size={14} />
                {hardware.name} · {Math.round(hardware.memory / 2 ** 30)} GB unified
                memory
            </div>
            <label className="local-search">
                <MagnifyingGlassIcon size={15} />
                <input
                    type="search"
                    value={query}
                    placeholder="Search models"
                    aria-label="Search models"
                    onChange={(event) => onQuery(event.target.value)}
                />
            </label>
        </>
    );
}
