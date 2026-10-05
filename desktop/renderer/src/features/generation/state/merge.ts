import type {
    GenerationEvent,
    GenerationJob,
    GenerationState,
} from "../../../../../contracts/wire.generated";
export function mergeJobs(previous: GenerationJob[], incoming: GenerationJob[]) {
    const jobs = new Map(previous.map((job) => [job.id, job]));
    for (const job of incoming) {
        const held = jobs.get(job.id);
        if (!held || Date.parse(job.updated) >= Date.parse(held.updated))
            jobs.set(job.id, job);
    }
    return [...jobs.values()];
}
export function mergeGeneration(
    held: GenerationState,
    next: GenerationState,
): GenerationState {
    return {
        ...next,
        library:
            held.library.revision > next.library.revision ? held.library : next.library,
        jobs: mergeJobs(held.jobs, next.jobs),
    };
}
export function applyGeneration(
    held: GenerationState,
    event: GenerationEvent,
): GenerationState {
    return {
        ...held,
        library:
            event.library && event.library.revision >= held.library.revision
                ? event.library
                : held.library,
        jobs: event.job ? mergeJobs(held.jobs, [event.job]) : held.jobs,
    };
}
