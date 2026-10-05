import type { Skill, SkillEdit } from "../../../../../contracts/wire.generated";
import { api, useRequests } from "../shared/useRequests";
const load = () => api().request("skills.state", undefined);
const empty: Skill[] = [];
export function useSkills() {
    const { data: skills, act, ...state } = useRequests(load, empty);
    return {
        ...state,
        skills,
        detail: (id: string) => act(() => api().request("skills.detail", { id }), false),
        saveSkill: (e: SkillEdit) =>
            act(() =>
                api().request("skills.save", {
                    id: e.id,
                    path: e.path,
                    instructions: e.instructions,
                    enabled: e.enabled,
                    activation: e.activation,
                    dependencies: e.dependencies,
                }),
            ),
        removeSkill: (id: string) => act(() => api().request("skills.remove", { id })),
        read: (id: string, path: string) =>
            act(() => api().request("skills.read", { id, path }), false),
        validate: (instructions: string) =>
            act(() => api().request("skills.validate", { instructions }), false),
    };
}
export type SkillsState = ReturnType<typeof useSkills>;
