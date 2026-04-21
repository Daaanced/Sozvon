//sozvon-client\src\router.tsx
import { createBrowserRouter, Navigate } from "react-router-dom";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Main from "./pages/Main";
import ChatPage from "./pages/ChatPage";
import AppShell from "./layouts/AppShell";
import { ProtectedRoute } from "./functions/protect";
import VoiceRoomsPage from "./components/voiceRooms/VoiceRoomsPage";

export const router = createBrowserRouter([
  { path: "/", element: <Navigate to="/login" replace /> },

  { path: "/login", element: <Login /> },
  { path: "/register", element: <Register /> },
  {
    element: <ProtectedRoute />,
    children: [
      {
        path: "/app",
        element: <AppShell />,
        children: [
          {
            index: true,
            element: <Main />,
          },
          {
            path: "chats/:chatId",
            element: <ChatPage />,
          },
          { path: "rooms", element: <VoiceRoomsPage /> },
        ],
      },
    ],
  },
]);
