"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { searchTags, type TagOption } from "@/lib/api";

const DEBOUNCE_MS = 200;
const MIN_QUERY_LENGTH = 0;

type TagAutocompleteProps = {
  value: string[];
  onChange: (tags: string[]) => void;
  placeholder?: string;
  disabled?: boolean;
  "aria-label"?: string;
};

export function TagAutocomplete({
  value,
  onChange,
  placeholder = "Search or add tags...",
  disabled = false,
  "aria-label": ariaLabel,
}: TagAutocompleteProps) {
  const [input, setInput] = useState("");
  const [suggestions, setSuggestions] = useState<TagOption[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [highlightIndex, setHighlightIndex] = useState(-1);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const fetchSuggestions = useCallback(async (query: string) => {
    setLoading(true);
    try {
      const list = await searchTags(query, 15);
      setSuggestions(list);
      setHighlightIndex(-1);
    } catch {
      setSuggestions([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      if (input.length >= MIN_QUERY_LENGTH) {
        fetchSuggestions(input);
        setOpen(true);
      } else {
        setSuggestions([]);
        setOpen(input.length > 0);
      }
      debounceRef.current = null;
    }, DEBOUNCE_MS);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [input, fetchSuggestions]);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const addTag = (tag: string) => {
    const trimmed = tag.trim();
    if (!trimmed) return;
    if (value.includes(trimmed)) return;
    onChange([...value, trimmed]);
    setInput("");
    setSuggestions([]);
    setOpen(false);
  };

  const removeTag = (index: number) => {
    onChange(value.filter((_, i) => i !== index));
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!open) {
      if (e.key === "Enter" && input.trim()) {
        addTag(input);
        e.preventDefault();
      }
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setHighlightIndex((i) => (i < suggestions.length - 1 ? i + 1 : i));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setHighlightIndex((i) => (i > 0 ? i - 1 : -1));
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (highlightIndex >= 0 && suggestions[highlightIndex]) {
        addTag(suggestions[highlightIndex].name);
      } else if (input.trim()) {
        addTag(input);
      }
    } else if (e.key === "Escape") {
      setOpen(false);
      setHighlightIndex(-1);
    }
  };

  return (
    <div ref={containerRef} className="relative w-full">
      <div className="flex min-h-10 flex-wrap items-center gap-1.5 rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 focus-within:ring-2 focus-within:ring-[var(--accent)] focus-within:ring-offset-2 focus-within:ring-offset-[var(--background)]">
        {value.map((tag, i) => (
          <span
            key={`${tag}-${i}`}
            className="inline-flex items-center gap-1 rounded-md bg-[var(--hover)] px-2 py-0.5 text-sm"
          >
            {tag}
            {!disabled && (
              <button
                type="button"
                onClick={() => removeTag(i)}
                className="rounded p-0.5 hover:bg-[var(--muted)]"
                aria-label={`Remove ${tag}`}
              >
                <span className="text-[var(--muted)]">×</span>
              </button>
            )}
          </span>
        ))}
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onFocus={() => input.length >= MIN_QUERY_LENGTH && setOpen(true)}
          onKeyDown={handleKeyDown}
          placeholder={value.length === 0 ? placeholder : ""}
          disabled={disabled}
          aria-label={ariaLabel}
          className="min-w-[120px] flex-1 border-0 bg-transparent p-0 text-[var(--foreground)] outline-none placeholder:text-[var(--muted)] disabled:opacity-50"
        />
      </div>
      {open && (suggestions.length > 0 || loading || input.trim()) && (
        <ul
          className="absolute z-10 mt-1 max-h-48 w-full overflow-auto rounded-lg border border-[var(--border)] bg-[var(--background)] py-1 shadow-lg"
          role="listbox"
        >
          {loading && (
            <li className="px-3 py-2 text-sm text-[var(--muted)]">
              Loading...
            </li>
          )}
          {!loading &&
            suggestions.map((s, i) => (
              <li
                key={s.id}
                role="option"
                aria-selected={highlightIndex === i}
                className={`cursor-pointer px-3 py-2 text-sm ${
                  highlightIndex === i ? "bg-[var(--hover)]" : ""
                }`}
                onMouseEnter={() => setHighlightIndex(i)}
                onMouseDown={(e) => {
                  e.preventDefault();
                  addTag(s.name);
                }}
              >
                {s.name}
              </li>
            ))}
          {!loading && input.trim() && (
            <li
              role="option"
              className="cursor-pointer px-3 py-2 text-sm text-[var(--muted)]"
              onMouseDown={(e) => {
                e.preventDefault();
                addTag(input.trim());
              }}
            >
              Add &quot;{input.trim()}&quot;
            </li>
          )}
        </ul>
      )}
    </div>
  );
}
