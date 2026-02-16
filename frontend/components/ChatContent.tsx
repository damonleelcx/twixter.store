"use client";

import {
  fetchCurrentUser,
  getChatSession,
  postChatOnboarding,
  postChatStream,
  type ChatPersonality,
  type ChatSessionResponse,
} from "@/lib/api";
import { getViewingCookie } from "@/lib/viewing";
import { useTranslations } from "next-intl";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { Icon } from "./Icon";

const defaultPersonality: ChatPersonality = {
  tone: "confident, playful",
  speaking_style: "short teasing sentences",
  boundaries: "never breaks character",
  quirks: "sarcastic humor, slow reveals",
  emotional_range: "intimate but controlled",
};

const MAX_CHARS_PER_FIELD = 50;
function limitToMaxChars(value: string, max = MAX_CHARS_PER_FIELD): string {
  return value.slice(0, max);
}

function formatMessageTime(iso?: string, locale = "en"): string {
  if (!iso) return "";
  try {
    const d = new Date(iso);
    const now = new Date();
    const isToday =
      d.getDate() === now.getDate() &&
      d.getMonth() === now.getMonth() &&
      d.getFullYear() === now.getFullYear();
    if (isToday) {
      return d.toLocaleTimeString(locale, {
        hour: "2-digit",
        minute: "2-digit",
      });
    }
    return d.toLocaleString(locale, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return "";
  }
}

export function ChatContent() {
  const t = useTranslations("chat");
  const params = useParams();
  const router = useRouter();
  const searchParams = useSearchParams();
  const locale = (params?.locale as string) || "en";
  const [state, setState] = useState<
    "loading" | "denied" | "onboarding" | "ready" | "error"
  >("loading");
  const [session, setSession] = useState<ChatSessionResponse | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [personality, setPersonality] = useState<ChatPersonality>(defaultPersonality);
  const [messages, setMessages] = useState<
    { role: string; content: string; created_at?: string }[]
  >([]);
  const [input, setInput] = useState("");
  const [streaming, setStreaming] = useState(false);
  const [streamingContent, setStreamingContent] = useState("");
  const [walletBalance, setWalletBalance] = useState<number | null>(null);
  /** When true, user is not signed in but has viewing_dark; show onboarding and redirect to sign up on Start chat */
  const [onboardingAsGuest, setOnboardingAsGuest] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const hasViewingParamOrCookie =
    (searchParams?.get("viewing")?.trim()?.length ?? 0) > 0 ||
    getViewingCookie() != null;

  const loadSession = useCallback(
    async (sessionId?: string) => {
      setState("loading");
      setErrorMsg(null);
      setOnboardingAsGuest(false);
      try {
        const data = await getChatSession(sessionId ?? session?.session_id);
        setSession(data);
        if (data.need_onboarding) {
          setState("onboarding");
        } else {
          setState("ready");
          setMessages(
          data.messages?.map((m) => ({
            role: m.role,
            content: m.content,
            created_at: m.created_at,
          })) ?? []
        );
        }
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        const isAuthError =
          msg.includes("don't have access") ||
          msg.includes("403") ||
          msg.includes("401") ||
          msg.toLowerCase().includes("unauthorized") ||
          msg.toLowerCase().includes("authorization");
        if (isAuthError && hasViewingParamOrCookie) {
          setState("onboarding");
          setOnboardingAsGuest(true);
        } else if (msg.includes("don't have access") || msg.includes("403")) {
          setState("denied");
        } else {
          setState("error");
          setErrorMsg(msg);
        }
      }
    },
    [session?.session_id, hasViewingParamOrCookie]
  );

  useEffect(() => {
    loadSession();
  }, []);

  useEffect(() => {
    if (state === "ready" || state === "onboarding") {
      fetchCurrentUser().then((u) => {
        if (u?.wallet_balance != null) setWalletBalance(u.wallet_balance);
      });
    }
  }, [state]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, streamingContent]);

  const handleOnboarding = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setErrorMsg(null);
    if (onboardingAsGuest) {
      const viewingToken =
        getViewingCookie() ?? searchParams?.get("viewing")?.trim() ?? null;
      const signupUrl = `/${locale}/auth/signup${viewingToken ? `?viewing=${encodeURIComponent(viewingToken)}` : ""}`;
      router.push(signupUrl);
      return;
    }
    try {
      await postChatOnboarding(name.trim(), personality);
      await loadSession(session?.session_id);
    } catch (e) {
      setErrorMsg(e instanceof Error ? e.message : String(e));
    }
  };

  const sendMessage = async () => {
    const text = input.trim();
    if (!text || streaming || !session?.session_id) return;
    setInput("");
    setMessages((prev) => [
      ...prev,
      { role: "user", content: text, created_at: new Date().toISOString() },
    ]);
    setStreaming(true);
    setStreamingContent("");
    try {
      let full = "";
      await postChatStream(session.session_id, text, (chunk) => {
        full += chunk;
        setStreamingContent(full);
      });
      setMessages((prev) => [
        ...prev,
        {
          role: "assistant",
          content: full,
          created_at: new Date().toISOString(),
        },
      ]);
      const u = await fetchCurrentUser();
      if (u?.wallet_balance != null) setWalletBalance(u.wallet_balance);
      window.dispatchEvent(new CustomEvent("wallet-updated"));
    } catch (e) {
      setErrorMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setStreaming(false);
      setStreamingContent("");
    }
  };

  if (state === "loading") {
    return (
      <div className="flex flex-1 items-center justify-center p-8">
        <span className="text-[var(--muted)]">{t("loading")}</span>
      </div>
    );
  }
  if (state === "denied") {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-4 p-8 text-center">
        <p className="text-[var(--muted)]">{t("accessDenied")}</p>
      </div>
    );
  }
  if (state === "error") {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-4 p-8 text-center">
        <p className="text-red-500">{errorMsg}</p>
        <button
          type="button"
          onClick={() => loadSession()}
          className="rounded-full bg-[var(--accent)] px-4 py-2 text-sm font-medium text-[var(--accent-foreground)]"
        >
          {t("retry")}
        </button>
      </div>
    );
  }
  if (state === "onboarding") {
    return (
      <div className="flex flex-1 flex-col p-4 md:p-6">
        <h2 className="mb-4 text-lg font-semibold">{t("onboardingTitle")}</h2>
        <p className="mb-4 text-sm text-[var(--muted)]">{t("onboardingHint")}</p>
        <form onSubmit={handleOnboarding} className="flex flex-col gap-4">
          <div>
            <label className="mb-1 block text-sm font-medium">{t("agentName")}</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full rounded-lg border border-[var(--border)] bg-[var(--background)] px-4 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--accent)]"
              placeholder={t("agentNamePlaceholder")}
              required
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium">{t("personality")} <span className="font-normal text-[var(--muted)]">({t("personalityModifyHint")})</span></label>
            <div className="grid gap-2 text-sm">
              <input
                type="text"
                value={limitToMaxChars(personality.tone)}
                maxLength={MAX_CHARS_PER_FIELD}
                onChange={(e) =>
                  setPersonality((p) => ({ ...p, tone: limitToMaxChars(e.target.value) }))
                }
                onPaste={(e) => {
                  e.preventDefault();
                  const text = e.clipboardData.getData("text/plain");
                  setPersonality((p) => ({ ...p, tone: limitToMaxChars(text) }));
                }}
                className="w-full rounded border border-[var(--border)] bg-[var(--background)] px-3 py-2"
                placeholder="tone (max 50 chars)"
              />
              <input
                type="text"
                value={limitToMaxChars(personality.speaking_style)}
                maxLength={MAX_CHARS_PER_FIELD}
                onChange={(e) =>
                  setPersonality((p) => ({
                    ...p,
                    speaking_style: limitToMaxChars(e.target.value),
                  }))
                }
                onPaste={(e) => {
                  e.preventDefault();
                  const text = e.clipboardData.getData("text/plain");
                  setPersonality((p) => ({ ...p, speaking_style: limitToMaxChars(text) }));
                }}
                className="w-full rounded border border-[var(--border)] bg-[var(--background)] px-3 py-2"
                placeholder="speaking_style (max 50 chars)"
              />
              <input
                type="text"
                value={limitToMaxChars(personality.boundaries)}
                maxLength={MAX_CHARS_PER_FIELD}
                onChange={(e) =>
                  setPersonality((p) => ({
                    ...p,
                    boundaries: limitToMaxChars(e.target.value),
                  }))
                }
                onPaste={(e) => {
                  e.preventDefault();
                  const text = e.clipboardData.getData("text/plain");
                  setPersonality((p) => ({ ...p, boundaries: limitToMaxChars(text) }));
                }}
                className="w-full rounded border border-[var(--border)] bg-[var(--background)] px-3 py-2"
                placeholder="boundaries (max 50 chars)"
              />
              <input
                type="text"
                value={limitToMaxChars(personality.quirks)}
                maxLength={MAX_CHARS_PER_FIELD}
                onChange={(e) => {
                  setPersonality((p) => ({
                    ...p,
                    quirks: limitToMaxChars(e.target.value),
                  }));
                }}
                onPaste={(e) => {
                  e.preventDefault();
                  const text = e.clipboardData.getData("text/plain");
                  setPersonality((p) => ({
                    ...p,
                    quirks: limitToMaxChars(text),
                  }));
                }}
                className="w-full rounded border border-[var(--border)] bg-[var(--background)] px-3 py-2"
                placeholder="quirks, comma-separated (max 50 chars)"
              />
              <input
                type="text"
                value={limitToMaxChars(personality.emotional_range)}
                maxLength={MAX_CHARS_PER_FIELD}
                onChange={(e) =>
                  setPersonality((p) => ({
                    ...p,
                    emotional_range: limitToMaxChars(e.target.value),
                  }))
                }
                onPaste={(e) => {
                  e.preventDefault();
                  const text = e.clipboardData.getData("text/plain");
                  setPersonality((p) => ({
                    ...p,
                    emotional_range: limitToMaxChars(text),
                  }));
                }}
                className="w-full rounded border border-[var(--border)] bg-[var(--background)] px-3 py-2"
                placeholder="emotional_range (max 50 chars)"
              />
            </div>
          </div>
          {errorMsg && <p className="text-sm text-red-500">{errorMsg}</p>}
          <button
            type="submit"
            className="rounded-full bg-[var(--accent)] px-6 py-3 font-bold text-[var(--accent-foreground)] hover:opacity-90"
          >
            {t("saveAndStart")}
          </button>
        </form>
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col">
      <div className="border-b border-[var(--border)] px-4 py-2 text-sm text-[var(--muted)]">
        {t("creditPerMessage")} · {t("balance")}: {walletBalance ?? "—"}
      </div>
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.map((m, i) => (
          <div
            key={i}
            className={`flex ${m.role === "user" ? "justify-end" : "justify-start"}`}
          >
            <div
              className={`max-w-[85%] rounded-2xl px-4 py-2 ${
                m.role === "user"
                  ? "bg-[var(--accent)] text-[var(--accent-foreground)]"
                  : "bg-[var(--hover)]"
              }`}
            >
              <p className="whitespace-pre-wrap text-sm">{m.content}</p>
              {m.created_at && (
                <p
                  className={`mt-1 text-xs opacity-80 ${
                    m.role === "user"
                      ? "text-[var(--accent-foreground)]"
                      : "text-[var(--muted)]"
                  }`}
                >
                  {formatMessageTime(m.created_at, locale)}
                </p>
              )}
            </div>
          </div>
        ))}
        {streaming && (
          <div className="flex justify-start">
            <div className="max-w-[85%] rounded-2xl bg-[var(--hover)] px-4 py-3">
              {streamingContent ? (
                <p className="whitespace-pre-wrap text-sm">{streamingContent}</p>
              ) : (
                <div className="flex gap-1" aria-label="Bot is typing">
                  <span className="h-2 w-2 rounded-full bg-[var(--muted)] animate-bounce [animation-delay:0ms]" />
                  <span className="h-2 w-2 rounded-full bg-[var(--muted)] animate-bounce [animation-delay:150ms]" />
                  <span className="h-2 w-2 rounded-full bg-[var(--muted)] animate-bounce [animation-delay:300ms]" />
                </div>
              )}
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>
      {errorMsg && (
        <div className="border-t border-[var(--border)] bg-red-500/10 px-4 py-2 text-sm text-red-600 dark:text-red-400">
          {errorMsg}
        </div>
      )}
      <div className="border-t border-[var(--border)] p-4">
        <div className="flex gap-2">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && sendMessage()}
            placeholder={t("messagePlaceholder")}
            className="min-w-0 flex-1 rounded-full border border-[var(--border)] bg-[var(--background)] px-4 py-3 focus:outline-none focus:ring-2 focus:ring-[var(--accent)]"
            disabled={streaming}
          />
          <button
            type="button"
            onClick={sendMessage}
            disabled={streaming || !input.trim()}
            className="shrink-0 rounded-full bg-[var(--accent)] p-3 text-[var(--accent-foreground)] hover:opacity-90 disabled:opacity-50"
          >
            <Icon d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" className="h-6 w-6" />
          </button>
        </div>
      </div>
    </div>
  );
}
