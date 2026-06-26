import type { ExtractedPage } from "../shared/types";

type ExtractPageResponse = { ok: true; page: ExtractedPage } | { ok: false; error: string };
type SelectionResponse = { ok: true; text: string } | { ok: false; error: string };

export async function extractFromActiveTab(): Promise<ExtractedPage> {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.id) {
    throw new Error("未找到当前标签页");
  }

  const response = (await chrome.tabs.sendMessage(tab.id, {
    type: "SURVEY_AI_EXTRACT_PAGE"
  })) as ExtractPageResponse | undefined;

  if (!response) {
    throw new Error("当前页面没有返回提取结果");
  }
  if (!response.ok) {
    throw new Error(response.error);
  }

  return response.page;
}

export async function getSelectionFromActiveTab(): Promise<string> {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.id) {
    throw new Error("未找到当前标签页");
  }

  const response = (await chrome.tabs.sendMessage(tab.id, {
    type: "SURVEY_AI_GET_SELECTION"
  })) as SelectionResponse | undefined;

  if (!response) {
    throw new Error("当前页面没有返回选中文本");
  }
  if (!response.ok) {
    throw new Error(response.error);
  }

  return response.text;
}
