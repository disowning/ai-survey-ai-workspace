import { extractPage } from "./extractPage";

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === "SURVEY_AI_GET_SELECTION") {
    sendResponse({ ok: true, text: window.getSelection()?.toString().trim() ?? "" });
    return true;
  }

  if (message?.type !== "SURVEY_AI_EXTRACT_PAGE") return false;

  try {
    sendResponse({ ok: true, page: extractPage() });
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown extraction error";
    sendResponse({ ok: false, error: message });
  }

  return true;
});
