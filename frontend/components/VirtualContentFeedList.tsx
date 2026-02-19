"use client";

import { useWindowVirtualizer } from "@tanstack/react-virtual";
import { useLayoutEffect, useRef, useState, type ReactNode } from "react";

const ESTIMATE_ITEM_HEIGHT = 420;
const OVERSCAN = 5;

type VirtualContentFeedListProps<T> = {
  items: T[];
  getItemKey: (item: T, index: number) => string | number;
  renderItem: (item: T, index: number) => ReactNode;
  /** Footer (load more trigger + loading / all caught up). Rendered after the virtual list. */
  footer: ReactNode;
};

export function VirtualContentFeedList<T>({
  items,
  getItemKey,
  renderItem,
  footer,
}: VirtualContentFeedListProps<T>) {
  const listContainerRef = useRef<HTMLDivElement>(null);
  const [scrollMargin, setScrollMargin] = useState(0);

  useLayoutEffect(() => {
    const el = listContainerRef.current;
    if (!el) return;
    const update = () => {
      setScrollMargin(el.getBoundingClientRect().top + window.scrollY);
    };
    update();
    const ro = new ResizeObserver(update);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const virtualizer = useWindowVirtualizer({
    count: items.length,
    getScrollElement: () => window,
    estimateSize: () => ESTIMATE_ITEM_HEIGHT,
    overscan: OVERSCAN,
    scrollMargin,
    getItemKey: (index) => String(getItemKey(items[index]!, index)),
  });

  const virtualItems = virtualizer.getVirtualItems();
  const totalSize = virtualizer.getTotalSize();

  if (items.length === 0) {
    return <>{footer}</>;
  }

  return (
    <div ref={listContainerRef}>
      <div
        style={{
          height: totalSize,
          width: "100%",
          position: "relative",
        }}
      >
        {virtualItems.map((virtualRow) => {
          const item = items[virtualRow.index];
          if (item == null) return null;
          return (
            <div
              key={virtualRow.key}
              data-index={virtualRow.index}
              ref={virtualizer.measureElement}
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "100%",
                transform: `translateY(${virtualRow.start - scrollMargin}px)`,
              }}
            >
              {renderItem(item, virtualRow.index)}
            </div>
          );
        })}
      </div>
      {footer}
    </div>
  );
}
