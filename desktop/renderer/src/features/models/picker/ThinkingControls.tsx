import "./thinking-controls.css";
import { useEffect, useState, type CSSProperties } from "react";
import type { ThinkingCapabilities } from "../../../../../contracts/wire.generated";

export const effortLabel = (effort: string) =>
    ({
        minimal: "Minimal",
        low: "Low",
        medium: "Medium",
        high: "High",
        xhigh: "Extra high",
        max: "Max",
        ultra: "Ultra",
    })[effort] || effort;

export function ThinkingControls({
    capabilities: caps,
    effort,
    disabled,
    onChange,
}: {
    capabilities?: ThinkingCapabilities;
    effort: string;
    disabled: boolean;
    onChange: (effort: string) => Promise<void>;
}) {
    const [preview, setPreview] = useState<string | null>(null);
    const visibleEffort = preview ?? effort;
    const levels = caps?.efforts || [];
    const enabled = caps
        ? !caps.canDisable ||
          (visibleEffort ? visibleEffort !== "none" : caps.defaultEnabled)
        : false;
    const current = levels.includes(effort)
        ? effort
        : caps?.defaultEffort || levels[0] || "";
    const [saving, setSaving] = useState(false);
    const levelKey = levels.join(",");
    useEffect(() => {
        setPreview(null);
    }, [effort, levelKey, caps?.defaultEffort, caps?.defaultEnabled, caps?.canDisable]);
    const commit = async (value: string) => {
        if (disabled || saving) return;
        setPreview(value);
        setSaving(true);
        try {
            await onChange(value);
        } catch {
            setPreview(null);
        } finally {
            setSaving(false);
        }
    };
    if (!caps) return null;
    return (
        <div className="thinking-controls">
            {levels.length > 1 && (
                <>
                    <div className="effort-caption">
                        <span>Thinking effort</span>
                        <output>
                            {levels.length
                                ? effortLabel(
                                      preview === "none" ? current : preview || current,
                                  )
                                : "Unavailable"}
                        </output>
                    </div>
                    <div
                        className="effort-slider"
                        data-disabled={
                            (disabled && !saving) || !enabled || levels.length < 2
                        }
                        style={
                            {
                                "--effort-position":
                                    Math.max(
                                        0,
                                        levels.indexOf(
                                            preview === "none"
                                                ? current
                                                : preview || current,
                                        ),
                                    ) / Math.max(1, levels.length - 1),
                            } as CSSProperties
                        }
                    >
                        <div className="effort-track" aria-hidden="true">
                            <div className="effort-fill" />
                        </div>
                        <input
                            className="effort-range"
                            type="range"
                            aria-label="Thinking effort"
                            aria-valuetext={effortLabel(
                                preview === "none" ? current : preview || current,
                            )}
                            min={0}
                            max={Math.max(1, levels.length - 1)}
                            step={1}
                            value={Math.max(
                                0,
                                levels.indexOf(
                                    preview === "none" ? current : preview || current,
                                ),
                            )}
                            disabled={disabled || saving || !enabled || levels.length < 2}
                            onChange={(event) =>
                                setPreview(levels[Number(event.target.value)] || null)
                            }
                            onPointerUp={(event) => {
                                const value = levels[Number(event.currentTarget.value)];
                                if (value && value !== current) void commit(value);
                            }}
                            onPointerCancel={() => setPreview(null)}
                            onKeyUp={(event) => {
                                if (
                                    [
                                        "ArrowLeft",
                                        "ArrowRight",
                                        "ArrowUp",
                                        "ArrowDown",
                                        "Home",
                                        "End",
                                        "PageUp",
                                        "PageDown",
                                    ].includes(event.key)
                                ) {
                                    const value =
                                        levels[Number(event.currentTarget.value)];
                                    if (value) void commit(value);
                                }
                            }}
                            onBlur={() => {
                                if (preview && preview !== current) void commit(preview);
                            }}
                        />
                        <div className="effort-stops" aria-hidden="true">
                            {levels.map((level, index) => (
                                <span
                                    key={level}
                                    style={{
                                        left: `${(index / Math.max(1, levels.length - 1)) * 100}%`,
                                    }}
                                />
                            ))}
                        </div>
                        <span className="effort-thumb" aria-hidden="true" />
                    </div>
                </>
            )}
            <div className="thinking-toggle-row">
                <span>Thinking</span>
                <button
                    type="button"
                    role="switch"
                    aria-label="Thinking"
                    aria-checked={enabled}
                    className="thinking-toggle"
                    data-saving={saving}
                    disabled={
                        disabled ||
                        saving ||
                        !caps?.canDisable ||
                        (!enabled && !levels.length)
                    }
                    title={
                        !caps
                            ? "Thinking capabilities unavailable"
                            : !caps.canDisable
                              ? "Thinking is required by this model"
                              : undefined
                    }
                    onClick={() => void commit(enabled ? "none" : current)}
                >
                    <span />
                </button>
            </div>
        </div>
    );
}
