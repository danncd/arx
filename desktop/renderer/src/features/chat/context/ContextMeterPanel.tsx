import type { ContextReport } from "../../../../../contracts/wire.generated";
import { formatReading, formatTokens, readingLabel, segments } from "./contextReading";

export function ContextMeterPanel({ report }: { report: ContextReport }) {
    return (
        <>
            <div className="context-head">
                <span className="context-reading">{readingLabel(report)}</span>
                <span className="context-counts">{formatReading(report)}</span>
            </div>
            <div className="context-bar" aria-hidden="true">
                {segments(report).map((segment) => (
                    <span
                        key={segment.key}
                        className={`context-segment part-${segment.key}`}
                        style={{ width: `${segment.share * 100}%` }}
                    />
                ))}
            </div>
            {report.known && (
                <div className="context-legend">
                    {segments(report).map((segment) => (
                        <Row
                            key={segment.key}
                            swatch={segment.key}
                            label={segment.label}
                            tokens={segment.tokens}
                        />
                    ))}
                </div>
            )}
        </>
    );
}

function Row({
    swatch,
    label,
    detail,
    tokens,
}: {
    swatch: "system" | "tools" | "messages" | "summary";
    label: string;
    detail?: string;
    tokens: number;
}) {
    return (
        <div className="context-row">
            <span className={`context-swatch part-${swatch}`} aria-hidden="true" />
            <span className="context-what">{label}</span>
            {detail && <span className="context-detail">{detail}</span>}
            <span className="context-value">~{formatTokens(Math.max(0, tokens))}</span>
        </div>
    );
}
