"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter, useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { getApiBase, getAccessToken, fetchCurrentUser } from "@/lib/api";
import { TagAutocomplete } from "@/components/TagAutocomplete";

const MAX_VIDEOS = 10;
const VIDEO_ACCEPT = "video/mp4,video/quicktime,video/x-msvideo,video/webm";

type VideoEntry = {
  file: File | null;
  name: string;
  description: string;
  priceCredits: string;
  tags: string[];
};

const defaultEntry: VideoEntry = {
  file: null,
  name: "",
  description: "",
  priceCredits: "0",
  tags: [],
};

export default function AdminUploadPage() {
  const router = useRouter();
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("admin.upload");
  const [allowed, setAllowed] = useState<boolean | null>(null);
  const [entries, setEntries] = useState<VideoEntry[]>([{ ...defaultEntry }]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const fileInputRefs = useRef<(HTMLInputElement | null)[]>([]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const user = await fetchCurrentUser();
      if (cancelled) return;
      if (!user) {
        setAllowed(false);
        return;
      }
      const isAdmin = user.account_type === "admin";
      const hasPermission = (user.permissions ?? []).includes("can_upload_content");
      setAllowed(isAdmin && hasPermission);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (allowed === false) {
      router.replace(`/${locale}`);
    }
  }, [allowed, locale, router]);

  const updateEntry = useCallback((index: number, patch: Partial<VideoEntry>) => {
    setEntries((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], ...patch };
      return next;
    });
  }, []);

  const addSlot = useCallback(() => {
    if (entries.length >= MAX_VIDEOS) return;
    setEntries((prev) => [...prev, { ...defaultEntry }]);
  }, [entries.length]);

  const removeSlot = useCallback((index: number) => {
    setEntries((prev) => {
      if (prev.length <= 1) return prev;
      return prev.filter((_, i) => i !== index);
    });
  }, []);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);
      const filled = entries.filter((e) => e.file && e.name.trim());
      if (filled.length === 0) {
        setError(t("errorNoVideos"));
        return;
      }
      const token = getAccessToken();
      if (!token) {
        setError(t("errorNotLoggedIn"));
        return;
      }
      setSubmitting(true);
      try {
        const formData = new FormData();
        const videosMeta = filled.map((e) => ({
          name: e.name.trim(),
          description: e.description.trim(),
          price: parseFloat(e.priceCredits) || 0,
          tags: e.tags,
          category: "light",
        }));
        formData.set("videos", JSON.stringify(videosMeta));
        filled.forEach((e) => {
          if (e.file) formData.append("files", e.file);
        });
        const base = getApiBase();
        const res = await fetch(`${base}/content/videos/upload`, {
          method: "POST",
          headers: { Authorization: `Bearer ${token}` },
          body: formData,
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) {
          setError(data?.error || data?.details || res.statusText);
          return;
        }
        setSuccess(true);
        setEntries([{ ...defaultEntry }]);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Request failed");
      } finally {
        setSubmitting(false);
      }
    },
    [entries, t]
  );

  if (allowed === null) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[var(--background)]">
        <p className="text-[var(--muted)]">{t("loading")}</p>
      </div>
    );
  }

  if (allowed === false) {
    return null;
  }

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto max-w-2xl px-4 py-8">
        <div className="mb-6 flex items-center gap-4">
          <Link
            href={`/${locale}`}
            className="rounded-lg border border-[var(--border)] px-3 py-2 text-sm hover:bg-[var(--hover)]"
          >
            ← {t("back")}
          </Link>
          <h1 className="text-2xl font-bold">{t("title")}</h1>
        </div>

        {error && (
          <div
            className="mb-4 rounded-lg border border-red-500/50 bg-red-500/10 px-4 py-3 text-sm text-red-600 dark:text-red-400"
            role="alert"
          >
            {error}
          </div>
        )}
        {success && (
          <div
            className="mb-4 rounded-lg border border-green-500/50 bg-green-500/10 px-4 py-3 text-sm text-green-600 dark:text-green-400"
            role="status"
          >
            {t("success")}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-8">
          {entries.map((entry, index) => (
            <fieldset
              key={index}
              className="rounded-xl border border-[var(--border)] bg-[var(--background)] p-4"
            >
              <div className="mb-3 flex items-center justify-between">
                <legend className="text-lg font-semibold">
                  {t("video")} {index + 1}
                </legend>
                {entries.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeSlot(index)}
                    className="rounded px-2 py-1 text-sm text-red-600 hover:bg-red-500/10 dark:text-red-400"
                  >
                    {t("remove")}
                  </button>
                )}
              </div>

              <div className="space-y-4">
                <div>
                  <label className="mb-1 block text-sm font-medium text-[var(--muted)]">
                    {t("file")}
                  </label>
                  <input
                    ref={(el) => {
                      fileInputRefs.current[index] = el;
                    }}
                    type="file"
                    accept={VIDEO_ACCEPT}
                    onChange={(e) => {
                      const file = e.target.files?.[0];
                      updateEntry(index, { file: file ?? null });
                    }}
                    className="block w-full text-sm text-[var(--foreground)] file:mr-4 file:rounded file:border-0 file:bg-[var(--accent)] file:px-4 file:py-2 file:text-[var(--accent-foreground)]"
                  />
                  {entry.file && (
                    <p className="mt-1 text-xs text-[var(--muted)]">
                      {entry.file.name}
                    </p>
                  )}
                </div>

                <div>
                  <label className="mb-1 block text-sm font-medium text-[var(--muted)]">
                    {t("name")} *
                  </label>
                  <input
                    type="text"
                    value={entry.name}
                    onChange={(e) => updateEntry(index, { name: e.target.value })}
                    required
                    className="w-full rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-[var(--foreground)] outline-none focus:ring-2 focus:ring-[var(--accent)]"
                    placeholder={t("namePlaceholder")}
                  />
                </div>

                <div>
                  <label className="mb-1 block text-sm font-medium text-[var(--muted)]">
                    {t("description")}
                  </label>
                  <textarea
                    value={entry.description}
                    onChange={(e) =>
                      updateEntry(index, { description: e.target.value })
                    }
                    rows={2}
                    className="w-full rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-[var(--foreground)] outline-none focus:ring-2 focus:ring-[var(--accent)]"
                    placeholder={t("descriptionPlaceholder")}
                  />
                </div>

                <div>
                  <label className="mb-1 block text-sm font-medium text-[var(--muted)]">
                    {t("priceCredits")}
                  </label>
                  <input
                    type="number"
                    min={0}
                    step={1}
                    value={entry.priceCredits}
                    onChange={(e) =>
                      updateEntry(index, { priceCredits: e.target.value })
                    }
                    className="w-full rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-[var(--foreground)] outline-none focus:ring-2 focus:ring-[var(--accent)]"
                    placeholder="0"
                  />
                </div>

                <div>
                  <label className="mb-1 block text-sm font-medium text-[var(--muted)]">
                    {t("tags")}
                  </label>
                  <TagAutocomplete
                    value={entry.tags}
                    onChange={(tags) => updateEntry(index, { tags })}
                    placeholder={t("tagsPlaceholder")}
                    aria-label={t("tags")}
                  />
                </div>
              </div>
            </fieldset>
          ))}

          {entries.length < MAX_VIDEOS && (
            <button
              type="button"
              onClick={addSlot}
              className="w-full rounded-lg border-2 border-dashed border-[var(--border)] py-4 text-sm font-medium text-[var(--muted)] hover:border-[var(--accent)] hover:text-[var(--foreground)]"
            >
              + {t("addVideo")}
            </button>
          )}

          <div className="flex gap-3">
            <button
              type="submit"
              disabled={submitting}
              className="rounded-lg bg-[var(--accent)] px-6 py-3 font-medium text-[var(--accent-foreground)] disabled:opacity-50 hover:opacity-90"
            >
              {submitting ? t("submitting") : t("submit")}
            </button>
            <Link
              href={`/${locale}`}
              className="rounded-lg border border-[var(--border)] px-6 py-3 font-medium hover:bg-[var(--hover)]"
            >
              {t("cancel")}
            </Link>
          </div>
        </form>
      </div>
    </div>
  );
}
