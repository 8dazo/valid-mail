const FAVORITES_KEY = "weme:favorites:v1";
const RECENT_KEY = "weme:recent:v1";
export function readList(key: string): string[] {
  try {
    const data: unknown = JSON.parse(localStorage.getItem(key) ?? "[]");
    return Array.isArray(data)
      ? data
          .filter((item): item is string => typeof item === "string")
          .slice(0, 50)
      : [];
  } catch {
    return [];
  }
}
export function saveList(key: string, value: string[]) {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    /* Storage is optional in private browsing. */
  }
}
export const storageKeys = { favorites: FAVORITES_KEY, recent: RECENT_KEY };
