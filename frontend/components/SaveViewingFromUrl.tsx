"use client";

import {
  getViewingFromSearchParams,
  setViewingCookie,
} from "@/lib/viewing";
import { useSearchParams } from "next/navigation";
import { useEffect } from "react";

/**
 * When user lands on a shared link with ?viewing=<encrypted token> (e.g. post detail),
 * persist token to cookie so registration/signup can send viewing_token (backend decrypts).
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
