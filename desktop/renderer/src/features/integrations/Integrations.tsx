import { useState } from "react";
import { useMCPServers } from "./mcp/useMCPServers";
import { useSkills } from "./skills/useSkills";
import { MCPSettings } from "./mcp/MCPSettings";
import { SkillsSettings } from "./skills/SkillsSettings";
import "../models/providers/providers.css";
import "../models/providers/network/network.css";
import "./integrations.css";
export function Integrations({
    section,
    running,
    onSection,
}: {
    section: "mcps" | "skills";
    running: boolean;
    onSection: (s: "mcps" | "skills") => void;
}) {
    const mcp = useMCPServers();
    const skills = useSkills();
    const error = section === "mcps" ? mcp.error : skills.error;
    const [mcpID, setMcpID] = useState<string>();
    return (
        <section aria-label={section === "mcps" ? "MCP settings" : "Skill settings"}>
            {section === "mcps" ? (
                <MCPSettings
                    state={mcp}
                    running={running || skills.busy}
                    initialID={mcpID}
                    onSkills={() => onSection("skills")}
                />
            ) : (
                <SkillsSettings
                    state={{ ...skills, servers: mcp.servers }}
                    running={running || mcp.busy}
                    onMCP={(id) => {
                        setMcpID(id);
                        onSection("mcps");
                    }}
                />
            )}
            {running && (
                <p className="permission-description">
                    Stop the current reply to change integrations.
                </p>
            )}
            {error && (
                <p className="network-hint integration-error" role="alert">
                    {error}
                </p>
            )}
        </section>
    );
}
