import { CaretLeftIcon } from "@phosphor-icons/react";
import type { IntegrationUsage } from "../../../../../contracts/wire.generated";
export function Usage({
    usage,
    kind,
}: {
    usage: IntegrationUsage;
    kind: "calls" | "times used";
}) {
    return (
        <div className="integration-usage">
            <span>
                {usage.count} {kind}
            </span>
            <span>Last used {when(usage.lastUsed)}</span>
        </div>
    );
}
export function when(value?: string) {
    if (!value) return "Never";
    const date = new Date(value);
    return Number.isNaN(date.getTime())
        ? "Unknown"
        : date.toLocaleString(undefined, {
              month: "short",
              day: "numeric",
              hour: "numeric",
              minute: "2-digit",
          });
}
export function Back({ name, onBack }: { name: string; onBack: () => void }) {
    return (
        <button type="button" className="text-button integration-back" onClick={onBack}>
            <CaretLeftIcon size={15} />
            {name}
        </button>
    );
}
export function DetailTabs<T extends string>({
    values,
    value,
    onChange,
}: {
    values: readonly { value: T; label: string }[];
    value: T;
    onChange: (v: T) => void;
}) {
    return (
        <div className="integration-tabs" aria-label="Details">
            {values.map((v) => (
                <button
                    type="button"
                    key={v.value}
                    aria-pressed={value === v.value}
                    onClick={() => onChange(v.value)}
                >
                    {v.label}
                </button>
            ))}
        </div>
    );
}
export function Activity({
    usage,
    kind,
}: {
    usage: IntegrationUsage;
    kind: "mcp" | "skill";
}) {
    return (
        <>
            <p className="network-hint">
                {kind === "mcp"
                    ? "Calls include executions and denied requests. Discovery is excluded."
                    : "Times used counts a skill’s first load per assistant reply."}
            </p>
            {usage.activity?.length ? (
                usage.activity.map((a, i) => (
                    <div className="integration-data-row" key={`${a.time}-${i}`}>
                        <div>
                            <strong>{a.operation}</strong>
                            <small>
                                {when(a.time)}
                                {a.conversation ? ` · Chat ${a.conversation}` : ""}
                            </small>
                        </div>
                        <span
                            className={
                                a.status === "Succeeded"
                                    ? "integration-tail"
                                    : "integration-tail integration-error"
                            }
                        >
                            {a.status}
                        </span>
                    </div>
                ))
            ) : (
                <p className="network-status">No activity yet.</p>
            )}
        </>
    );
}
export function Remove({
    name,
    description,
    busy,
    onRemove,
}: {
    name: string;
    description: string;
    busy: boolean;
    onRemove: () => void;
}) {
    return (
        <div className="integration-removal">
            <h4 className="settings-section-title">Remove {name}?</h4>
            <p className="permission-description">{description}</p>
            <button
                type="button"
                className="directory-button"
                disabled={busy}
                onClick={onRemove}
            >
                Remove
            </button>
        </div>
    );
}
