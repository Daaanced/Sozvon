//sozvon-client\src\layouts\AppShell.tsx

import { Outlet } from "react-router-dom";
import Sidebar from "../components/Sidebar";
import UserInfo from "../components/UserInfo";
import { ChatProvider } from "../context/ChatContext";
import { connectWS } from "../services/ws";
import { useEffect, useState, useRef, useCallback } from "react";

const USERINFO_BREAKPOINT = 1100;
const SIDEBAR_MIN = 180;
const SIDEBAR_MAX = 400;
const SIDEBAR_DEFAULT = 260;

export default function AppShell() {
  const [width, setWidth] = useState(window.innerWidth);
  const [sidebarWidth, setSidebarWidth] = useState(SIDEBAR_DEFAULT);
  const dragging = useRef(false);
  const startX = useRef(0);
  const startWidth = useRef(0);

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (token) connectWS(token);
  }, []);

  useEffect(() => {
    const handleResize = () => setWidth(window.innerWidth);
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  const onMouseDown = useCallback(
    (e: React.MouseEvent) => {
      dragging.current = true;
      startX.current = e.clientX;
      startWidth.current = sidebarWidth;
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
    },
    [sidebarWidth],
  );

  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => {
      if (!dragging.current) return;
      const delta = e.clientX - startX.current;
      const next = Math.min(
        SIDEBAR_MAX,
        Math.max(SIDEBAR_MIN, startWidth.current + delta),
      );
      setSidebarWidth(next);
    };
    const onMouseUp = () => {
      if (!dragging.current) return;
      dragging.current = false;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
    window.addEventListener("mousemove", onMouseMove);
    window.addEventListener("mouseup", onMouseUp);
    return () => {
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseup", onMouseUp);
    };
  }, []);

  const showUserInfo = width >= USERINFO_BREAKPOINT;

  return (
    <ChatProvider>
      <div style={{ display: "flex", height: "100vh", overflow: "hidden" }}>
        {/* SIDEBAR */}
        <div
          style={{
            width: sidebarWidth,
            flexShrink: 0,
            borderRight: "1px solid #ddd",
            overflow: "hidden",
          }}
        >
          <Sidebar />
        </div>

        {/* RESIZER */}
        <div
          onMouseDown={onMouseDown}
          style={{
            width: 4,
            flexShrink: 0,
            cursor: "col-resize",
            background: "transparent",
            transition: "background 0.15s",
          }}
          onMouseEnter={(e) => (e.currentTarget.style.background = "#ddd")}
          onMouseLeave={(e) =>
            (e.currentTarget.style.background = "transparent")
          }
        />

        {/* CENTER */}
        <div
          style={{
            flex: 1,
            display: "flex",
            flexDirection: "column",
            overflow: "hidden",
            padding: 12,
          }}
        >
          <Outlet />
        </div>

        {/* USER INFO */}
        {showUserInfo && (
          <div
            style={{
              width: 300,
              flexShrink: 0,
              borderLeft: "1px solid #ddd",
              background: "#fafafa",
              overflowY: "auto",
            }}
          >
            <UserInfo />
          </div>
        )}
      </div>
    </ChatProvider>
  );
}
