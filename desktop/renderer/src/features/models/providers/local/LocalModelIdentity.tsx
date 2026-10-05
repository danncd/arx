import { CpuIcon } from "@phosphor-icons/react";
import { useState } from "react";
import { modelBrand } from "./capabilities";
import "./identity.css";

export function LocalModelIdentity({
    name,
    source,
    subtitle,
    badges,
    tools,
    verified = false,
    hideTools = false,
}: {
    name: string;
    source: string;
    subtitle: string;
    badges: string[];
    tools?: boolean;
    verified?: boolean;
    hideTools?: boolean;
}) {
    const brand = modelBrand(source);
    const [failed, setFailed] = useState(false);
    return (
        <div className="model-identity">
            <div className="model-identity-heading">
                <div className="model-logo" aria-hidden="true">
                    {brand && !failed ? (
                        <img
                            src={`./model-logos/${brand}.png`}
                            alt=""
                            onError={() => setFailed(true)}
                        />
                    ) : (
                        <CpuIcon size={22} />
                    )}
                </div>
                <div className="local-model-info">
                    <div className="local-model-title">{name}</div>
                    <div className="local-model-meta">{subtitle}</div>
                </div>
            </div>
            <div className="model-capabilities">
                {badges.map((badge) => (
                    <span
                        key={badge}
                        className="model-capability"
                        title={
                            verified
                                ? "Reported by the loaded runtime"
                                : "Model metadata; checked when loaded"
                        }
                    >
                        {badge}
                    </span>
                ))}
                {!hideTools && (
                    <span
                        className={`model-capability${tools === true ? " verified" : ""}`}
                        title="Tool support is reported by the runtime when loaded"
                    >
                        {tools === undefined
                            ? "Tools unverified"
                            : tools
                              ? "Tools"
                              : "No tools"}
                    </span>
                )}
            </div>
        </div>
    );
}
