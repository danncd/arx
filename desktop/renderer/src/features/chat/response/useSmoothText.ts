import { useEffect, useRef, useState } from "react";

const revealInterval = 32;
const revealMinimum = 3;

export function useSmoothText(target: string, active: boolean) {
    const [shown, setShown] = useState(target);
    const revealed = useRef(target);
    const wasActive = useRef(active);
    const finishing = useRef(false);

    useEffect(() => {
        const motion = window.matchMedia("(prefers-reduced-motion: reduce)");
        if (wasActive.current && !active) finishing.current = true;
        wasActive.current = active;
        if (active) finishing.current = false;
        if (
            motion.matches ||
            (!active && !finishing.current) ||
            !target.startsWith(revealed.current)
        ) {
            revealed.current = target;
            setShown(target);
            finishing.current = false;
            return;
        }

        let frame = 0;
        let lastStep = 0;
        const reveal = (now: number) => {
            if (now - lastStep < revealInterval) {
                frame = requestAnimationFrame(reveal);
                return;
            }
            lastStep = now;
            const remaining = target.length - revealed.current.length;
            if (remaining <= 0) {
                finishing.current = false;
                return;
            }
            let end = motion.matches
                ? target.length
                : Math.min(
                      target.length,
                      revealed.current.length +
                          Math.max(revealMinimum, Math.round(remaining * 0.2)),
                  );

            const last = target.charCodeAt(end - 1);
            if (last >= 0xd800 && last <= 0xdbff && end < target.length) end++;
            revealed.current = target.slice(0, end);
            setShown(revealed.current);
            if (end < target.length) frame = requestAnimationFrame(reveal);
            else finishing.current = false;
        };
        if (revealed.current.length < target.length)
            frame = requestAnimationFrame(reveal);
        return () => cancelAnimationFrame(frame);
    }, [active, target]);

    return shown;
}
