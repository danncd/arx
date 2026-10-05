import { useGeneration } from "../../generation/state/useGeneration";
import type { ToolRecord } from "../../../../../contracts/wire.generated";

export function useGenerationProgress(tools: ToolRecord[], live: boolean) {
    const generation = useGeneration();
    const jobs = Object.fromEntries(generation.jobs.map((job) => [job.toolCall, job]));
    return tools.map((tool) => {
        const job = jobs[tool.id];
        if (!live || !job || tool.status === "done") return tool;
        const percent = job.progress > 0 ? ` ${Math.round(job.progress * 100)}%` : "";
        return {
            ...tool,
            summary: percent.trim(),
            result: JSON.stringify({ text: job.error || `${job.detail}${percent}` }),
        };
    });
}
