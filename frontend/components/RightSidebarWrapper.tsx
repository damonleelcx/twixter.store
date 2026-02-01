import {
  AUTH_TOKEN_COOKIE,
  fetchCurrentUserServer,
  fetchTrendingTagsServer,
  type TrendingTagItem,
} from "@/lib/api";
import { cookies } from "next/headers";
import { RightSidebar } from "./RightSidebar";

export async function RightSidebarWrapper() {
  const cookieStore = await cookies();
  const token = cookieStore.get(AUTH_TOKEN_COOKIE)?.value ?? undefined;
  const user = await fetchCurrentUserServer(token);
  const category = user?.permissions?.includes("can_view_nsfw") ? "all" : "light";
  const initialTrendingTags: TrendingTagItem[] = await fetchTrendingTagsServer(
    category,
    10,
    token ?? undefined
  );
  return <RightSidebar initialTrendingTags={initialTrendingTags} />;
}
