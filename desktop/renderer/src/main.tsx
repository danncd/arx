import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Startup } from "./app/Startup";
import { NotificationsProvider } from "./ui/notifications/Notifications";

createRoot(document.getElementById("root")!).render(
    <StrictMode>
        <NotificationsProvider>
            <Startup />
        </NotificationsProvider>
    </StrictMode>,
);
