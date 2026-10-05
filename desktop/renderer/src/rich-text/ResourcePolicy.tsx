import { createContext, useContext } from "react";
import type { PermissionMode } from "../../../contracts/wire.generated";

export const ResourcePolicy = createContext<{
    conversation: string;
    mode: PermissionMode;
}>({ conversation: "", mode: "ask" });
export const useResourcePolicy = () => useContext(ResourcePolicy);
