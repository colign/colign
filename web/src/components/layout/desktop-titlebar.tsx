"use client";

import { useEffect, useState } from "react";

export function DesktopTitlebar() {
  const [isDesktop, setIsDesktop] = useState(false);

  useEffect(() => {
    setIsDesktop(document.documentElement.classList.contains("desktop-app"));
  }, []);

  if (!isDesktop) return null;

  return <div className="desktop-drag-region" />;
}
