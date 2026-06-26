import type { ExtractedPage } from "../../shared/types";

const MAX_PAGE_TEXT_LENGTH = 20000;
const MAX_OPTION_TEXT_LENGTH = 8000;

const interactiveSelector = [
  "button",
  "[role='button']",
  "[role='radio']",
  "[role='checkbox']",
  "label",
  "select option",
  "input[type='button']",
  "input[type='submit']"
].join(",");

export function extractDefaultPage(): ExtractedPage {
  const pageText = normalizeWhitespace(document.body?.innerText ?? "").slice(0, MAX_PAGE_TEXT_LENGTH);
  const optionsText = collectOptionText();

  return {
    url: window.location.href,
    domain: window.location.hostname.replace(/^www\./, ""),
    title: document.title || firstNonEmptyLine(pageText) || "Untitled page",
    pageText,
    questionText: findQuestionText(pageText),
    optionsText,
    language: document.documentElement.lang || undefined,
    extractedAt: new Date().toISOString()
  };
}

function collectOptionText(): string | undefined {
  const values = new Set<string>();
  document.querySelectorAll<HTMLElement>(interactiveSelector).forEach((node) => {
    if (isSensitiveInput(node)) return;

    const text = normalizeWhitespace(readNodeText(node));
    if (text.length >= 1 && text.length <= 300) {
      values.add(text);
    }
  });

  const output = Array.from(values).slice(0, 80).join("; ");
  return output ? output.slice(0, MAX_OPTION_TEXT_LENGTH) : undefined;
}

function readNodeText(node: HTMLElement): string {
  if (node instanceof HTMLInputElement) {
    return node.value || node.getAttribute("aria-label") || "";
  }
  return node.innerText || node.textContent || node.getAttribute("aria-label") || "";
}

function isSensitiveInput(node: HTMLElement): boolean {
  if (!(node instanceof HTMLInputElement)) return false;

  const type = node.type.toLowerCase();
  const name = `${node.name} ${node.id} ${node.autocomplete}`.toLowerCase();
  return (
    type === "password" ||
    name.includes("password") ||
    name.includes("token") ||
    name.includes("cookie") ||
    name.includes("secret")
  );
}

function findQuestionText(pageText: string): string | undefined {
  const lines = pageText
    .split("\n")
    .map((line) => normalizeWhitespace(line))
    .filter(Boolean);

  const questionLike = lines.find((line) => /[?？]$/.test(line) && line.length <= 500);
  return questionLike || lines.find((line) => line.length >= 8 && line.length <= 500);
}

function firstNonEmptyLine(text: string): string | undefined {
  return text
    .split("\n")
    .map((line) => normalizeWhitespace(line))
    .find(Boolean);
}

function normalizeWhitespace(value: string): string {
  return value
    .replace(/\u00a0/g, " ")
    .replace(/[ \t]+/g, " ")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}
