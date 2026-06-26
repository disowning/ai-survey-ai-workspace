import { DEFAULT_API_BASE_URL, STORAGE_KEYS } from "../shared/constants";
import type { ExtensionSettings, LocalProfile, SiteDetection } from "../shared/types";

const defaultProfile: LocalProfile = {
  profileId: null,
  profileKey: "",
  profileName: ""
};

const defaultSettings: ExtensionSettings = {
  apiBaseUrl: DEFAULT_API_BASE_URL
};

export async function getLocalProfile(): Promise<LocalProfile> {
  if (!hasChromeStorage()) return defaultProfile;
  const result = await chrome.storage.local.get(STORAGE_KEYS.profile);
  return { ...defaultProfile, ...(result[STORAGE_KEYS.profile] as Partial<LocalProfile> | undefined) };
}

export async function setLocalProfile(profile: LocalProfile): Promise<void> {
  if (!hasChromeStorage()) return;
  await chrome.storage.local.set({ [STORAGE_KEYS.profile]: profile });
}

export async function getSettings(): Promise<ExtensionSettings> {
  if (!hasChromeStorage()) return defaultSettings;
  const result = await chrome.storage.local.get(STORAGE_KEYS.settings);
  return { ...defaultSettings, ...(result[STORAGE_KEYS.settings] as Partial<ExtensionSettings> | undefined) };
}

export async function setSettings(settings: ExtensionSettings): Promise<void> {
  if (!hasChromeStorage()) return;
  await chrome.storage.local.set({ [STORAGE_KEYS.settings]: settings });
}

export async function getLastSite(): Promise<SiteDetection | null> {
  if (!hasChromeStorage()) return null;
  const result = await chrome.storage.local.get(STORAGE_KEYS.lastSite);
  return (result[STORAGE_KEYS.lastSite] as SiteDetection | undefined) ?? null;
}

export async function setLastSite(site: SiteDetection): Promise<void> {
  if (!hasChromeStorage()) return;
  await chrome.storage.local.set({ [STORAGE_KEYS.lastSite]: site });
}

function hasChromeStorage(): boolean {
  return typeof chrome !== "undefined" && Boolean(chrome.storage?.local);
}
