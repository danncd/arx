import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "../app/App";
import { NotificationsProvider } from "../ui/notifications/Notifications";
import { installScenario } from "./scenarios";
const initial = installScenario();
createRoot(document.getElementById("root")!).render(
    <StrictMode>
        <NotificationsProvider>
            <App initial={initial} />
        </NotificationsProvider>
    </StrictMode>,
);
