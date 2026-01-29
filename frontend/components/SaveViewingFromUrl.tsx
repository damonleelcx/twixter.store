"use client";

import {
  getViewingFromSearchParams,
  setViewingCookie,
} from "@/lib/viewing";
import { useSearchParams } from "next/navigation";
import { useEffect } from "react";

/**
 * When user lands on a shared link with ?viewing_dark (e.g. post detail),
 * persist to cookie so registration/signup keeps dark account type.
 */
export function SaveViewingFromUrl() {
  const searchParams = useSearchParams();

  useEffect(() => {
    const fromUrl = getViewingFromSearchParams(searchParams);
    if (fromUrl) {
      setViewingCookie(fromUrl);
    }
  }, [searchParams]);

  return null;
}
