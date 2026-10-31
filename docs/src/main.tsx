import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "@fontsource-variable/geist";
import "@fontsource-variable/geist-mono";
import "./styles.css";
import { MotionGlobalConfig } from "motion/react";
import App from "./App";

// ?still: no animations at all (static screenshots).
if (new URLSearchParams(location.search).has("still")) MotionGlobalConfig.skipAnimations = true;

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
