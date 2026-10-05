import { formatSize } from "../../format";

export function LocalModelProgress({
    label,
    received,
    total,
}: {
    label: string;
    received?: number;
    total?: number;
}) {
    const percent =
        total && received !== undefined
            ? Math.min(100, Math.round((received / total) * 100))
            : undefined;
    return (
        <div className="local-progress">
            <div className="local-progress-label">
                <span>{label}</span>
                {percent !== undefined && (
                    <span>
                        {formatSize(received || 0)} / {formatSize(total || 0)} · {percent}
                        %
                    </span>
                )}
            </div>
            <div
                className="local-progress-track"
                role="progressbar"
                aria-label={label}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={percent}
            >
                <div
                    className={`local-progress-fill${percent === undefined ? " indeterminate" : ""}`}
                    style={percent === undefined ? undefined : { width: `${percent}%` }}
                />
            </div>
        </div>
    );
}
