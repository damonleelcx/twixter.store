"use client";

import { TagAutocomplete } from "@/components/TagAutocomplete";
import {
  fetchCurrentUser,
  getContentById,
  updateContent,
} from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

type ContentCategory = "light" | "dark";

export default function PostEditPage() {
  const router = useRouter();
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const idParam = params?.id as string;
  const contentId = idParam ? parseInt(idParam, 10) : NaN;
  const t = useTranslations("postDetail");
  const tAdmin = useTranslations("admin.upload");

  const [allowed, setAllowed] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [category, setCategory] = useState<ContentCategory>("light");
  const [priceCredits, setPriceCredits] = useState("0");
  const [tags, setTags] = useState<string[]>([]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!idParam || !Number.isInteger(contentId) || contentId <= 0) {
        if (!cancelled) setAllowed(false);
        return;
      }
      const user = await fetchCurrentUser();
      if (cancelled) return;
      if (!user) {
        setAllowed(false);
        setLoading(false);
        return;
      }
      const hasPermission = (user.permissions ?? []).includes("can_edit_content");
      if (!hasPermission) {
        setAllowed(false);
        setLoading(false);
        return;
      }
      setAllowed(true);
      try {
        const detail = await getContentById(contentId);
        if (cancelled) return;
        const c = detail.content;
        setName(c.name ?? "");
        setDescription(c.description ?? "");
        setCategory((c.category as ContentCategory) ?? "light");
        setPriceCredits(String(c.price ?? 0));
        setTags(c.tags ?? []);
      } catch {
        if (!cancelled) setError(t("editErrorNotFound"));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [idParam, contentId, t]);

  useEffect(() => {
    if (allowed === false) {
      router.replace(`/${locale}`);
    }
  }, [allowed, locale, router]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);
      if (!Number.isInteger(contentId) || contentId <= 0) {
        setError(t("editErrorNotFound"));
        return;
      }
      setSubmitting(true);
      try {
        await updateContent(contentId, {
          name: name.trim(),
          description: description.trim(),
          category,
          price: parseFloat(priceCredits) || 0,
          tags,
        });
        setSuccess(true);
        if (typeof window !== "undefined") {
          sessionStorage.setItem("twixter_feed_refetch", "1");
        }
        setTimeout(() => {
          router.push(`/${locale}/post/${contentId}?updated=1`);
        }, 800);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to update");
      } finally {
        setSubmitting(false);
      }
    },
    [contentId, name, description, category, priceCredits, tags, locale, router, t]
  );

  if (allowed === null || loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[var(--background)]">
        <p className="text-[var(--muted)]">{tAdmin("loading")}</p>
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
            href={`/${locale}/post/${contentId}`}
            className="rounded-lg border border-[var(--border)] px-3 py-2 text-sm hover:bg-[var(--hover)]"
          >
            ← {t("back")}
          </Link>
          <h1 className="text-2xl font-bold">{t("editTitle")}</h1>
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
            {t("editSuccess")}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--muted)]">
              {tAdmin("name")} *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="w-full rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-[var(--foreground)] outline-none focus:ring-2 focus:ring-[var(--accent)]"
              placeholder={tAdmin("namePlaceholder")}
            />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--muted)]">
              {tAdmin("description")}
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="w-full rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-[var(--foreground)] outline-none focus:ring-2 focus:ring-[var(--accent)]"
              placeholder={tAdmin("descriptionPlaceholder")}
            />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--muted)]">
              {tAdmin("category")}
            </label>
            <select
              value={category}
              onChange={(e) =>
                setCategory(e.target.value as ContentCategory)
              }
              className="w-full rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-[var(--foreground)] outline-none focus:ring-2 focus:ring-[var(--accent)]"
              aria-label={tAdmin("category")}
            >
              <option value="light">{tAdmin("categoryLight")}</option>
              <option value="dark">{tAdmin("categoryDark")}</option>
            </select>
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--muted)]">
              {tAdmin("priceCredits")}
            </label>
            <input
              type="number"
              min={0}
              step="any"
              value={priceCredits}
              onChange={(e) => setPriceCredits(e.target.value)}
              className="w-full rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-[var(--foreground)] outline-none focus:ring-2 focus:ring-[var(--accent)]"
              placeholder="0"
            />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--muted)]">
              {tAdmin("tags")}
            </label>
            <TagAutocomplete
              value={tags}
              onChange={setTags}
              placeholder={tAdmin("tagsPlaceholder")}
              aria-label={tAdmin("tags")}
            />
          </div>

          <div className="flex gap-3">
            <button
              type="submit"
              disabled={submitting}
              className="rounded-lg bg-[var(--accent)] px-6 py-3 font-medium text-[var(--accent-foreground)] disabled:opacity-50 hover:opacity-90"
            >
              {submitting ? t("editSaving") : t("editSave")}
            </button>
            <Link
              href={`/${locale}/post/${contentId}`}
              className="rounded-lg border border-[var(--border)] px-6 py-3 font-medium hover:bg-[var(--hover)]"
            >
              {tAdmin("cancel")}
            </Link>
          </div>
        </form>
      </div>
    </div>
  );
}
