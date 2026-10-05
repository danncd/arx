import { MediaResult } from "../media/MediaResult";
import { Fragment } from "react";
import type { Message } from "../types";
import { Markdown } from "../../../rich-text/Markdown";
import { Thinking } from "./Thinking";
import { useSmoothText } from "./useSmoothText";
import { responseGroups } from "./responseGroups";
import { ToolActivity } from "../tools/ToolActivity";
import { ToolReveal } from "../tools/ToolReveal";

function Segment({
    text,
    reasoning,
    active,
    persistenceKey = "tail",
    onExpand,
}: {
    text: string;
    reasoning: string;
    active: boolean;
    persistenceKey?: string;
    onExpand: () => void;
}) {
    const revealed = useSmoothText(text, active);
    return (
        <>
            {reasoning.trim() && (
                <Thinking
                    text={reasoning}
                    persistenceKey={persistenceKey}
                    streaming={active && !text.trim()}
                    onExpand={onExpand}
                />
            )}
            {text.trim() && (
                <div className={`prose${active ? " streaming" : ""}`}>
                    <Markdown text={revealed} />
                </div>
            )}
        </>
    );
}

export function ResponseContent({
    message,
    streaming,
    onExpand,
}: {
    message: Message;
    streaming: boolean;
    onExpand: () => void;
}) {
    const response = responseGroups(message);
    return (
        <>
            {response.groups.map((group) => (
                <Fragment key={group.tools[0].id}>
                    <Segment
                        text={group.text}
                        reasoning={group.thought}
                        active={false}
                        persistenceKey={group.tools[0].id}
                        onExpand={onExpand}
                    />
                    <ToolReveal live={streaming}>
                        <ToolActivity
                            tools={group.tools}
                            live={streaming}
                            onExpand={onExpand}
                        />
                    </ToolReveal>
                    <MediaResult tools={group.tools} />
                </Fragment>
            ))}
            <Segment
                text={response.text}
                reasoning={response.reasoning}
                active={streaming}
                onExpand={onExpand}
            />
        </>
    );
}
