import type { ExtractedPage } from "../../shared/types";

const MAX_PAGE_TEXT_LENGTH = 20000;
const MAX_OPTION_TEXT_LENGTH = 8000;

const questionContainerSelector = [
  "[role='form']",
  "[role='group']",
  "form",
  "fieldset",
  "[class*='question' i]",
  "[id*='question' i]",
  "[class*='survey' i]",
  "[id*='survey' i]",
  "[class*='poll' i]",
  "[class*='matrix' i]"
].join(",");

const optionSelector = [
  "button",
  "[role='button']",
  "[role='radio']",
  "[role='checkbox']",
  "label",
  "select option",
  "input[type='button']",
  "input[type='submit']"
].join(",");

type ExtractedBlock = NonNullable<ExtractedPage["extractedBlocks"]>[number];

type QuestionBlock = {
  questionText?: string;
  optionsText?: string;
  progressText?: string;
  questionType?: string;
  confidence: number;
  blocks: ExtractedBlock[];
};

export function extractDefaultPage(): ExtractedPage {
  const rootText = collectReadableText(document);
  const pageText = normalizeWhitespace(rootText).slice(0, MAX_PAGE_TEXT_LENGTH);
  const block = extractQuestionBlock(pageText);

  return {
    url: window.location.href,
    domain: window.location.hostname.replace(/^www\./, ""),
    title: document.title || firstNonEmptyLine(pageText) || "Untitled page",
    pageText,
    questionText: block.questionText || findQuestionText(pageText),
    optionsText: block.optionsText || collectOptionText(document),
    progressText: block.progressText || findProgressText(pageText),
    questionType: block.questionType,
    extractionConfidence: block.confidence,
    extractedBlocks: block.blocks,
    language: document.documentElement.lang || undefined,
    extractedAt: new Date().toISOString()
  };
}

function extractQuestionBlock(pageText: string): QuestionBlock {
  const candidates = Array.from(document.querySelectorAll<HTMLElement>(questionContainerSelector))
    .filter(isVisible)
    .filter((node) => !containsSensitiveNode(node))
    .map((node) => {
      const text = normalizeWhitespace(readNodeText(node));
      return { node, text, score: scoreQuestionContainer(node, text) };
    })
    .filter((item) => item.text.length >= 8)
    .sort((a, b) => b.score - a.score);

  const best = candidates[0];
  const source = best?.score > 2 ? best.node : document.body;
  const sourceText = normalizeWhitespace(readNodeText(source));
  const options = collectOptionValues(source);
  const matrix = collectMatrixText(source);
  const progressText = findProgressText(sourceText) || findProgressText(pageText);
  const questionText = findQuestionText(sourceText) || findQuestionText(pageText);
  const questionType = inferQuestionType(source, options, matrix);
  const blocks: ExtractedBlock[] = [];

  if (progressText) blocks.push({ role: "progress", text: progressText });
  if (questionText) blocks.push({ role: "question", text: questionText });
  for (const option of options.slice(0, 40)) {
    blocks.push({ role: "option", text: option });
  }
  if (matrix) blocks.push({ role: "instruction", text: matrix });

  const optionsText = [...options, matrix].filter(Boolean).join("; ").slice(0, MAX_OPTION_TEXT_LENGTH) || undefined;
  const confidence = computeConfidence({ questionText, optionsText, progressText, bestScore: best?.score ?? 0 });

  return {
    questionText,
    optionsText,
    progressText,
    questionType,
    confidence,
    blocks
  };
}

function collectReadableText(root: Document): string {
  const parts = [root.body?.innerText ?? ""];
  for (const frame of Array.from(root.querySelectorAll("iframe"))) {
    try {
      const doc = frame.contentDocument;
      if (doc?.body?.innerText) parts.push(doc.body.innerText);
    } catch {
      parts.push("[iframe content unavailable]");
    }
  }
  return parts.join("\n\n");
}

function collectOptionText(root: ParentNode): string | undefined {
  const output = collectOptionValues(root).slice(0, 80).join("; ");
  return output ? output.slice(0, MAX_OPTION_TEXT_LENGTH) : undefined;
}

function collectOptionValues(root: ParentNode): string[] {
  const values = new Set<string>();
  root.querySelectorAll<HTMLElement>(optionSelector).forEach((node) => {
    if (!isVisible(node) || isSensitiveInput(node)) return;

    const text = normalizeWhitespace(readNodeText(node));
    if (text.length >= 1 && text.length <= 300 && !isNavigationText(text)) {
      values.add(text);
    }
  });
  return Array.from(values);
}

function collectMatrixText(root: ParentNode): string | undefined {
  const tables = Array.from(root.querySelectorAll<HTMLTableElement>("table")).filter(isVisible);
  const rows: string[] = [];
  for (const table of tables.slice(0, 2)) {
    const cells = Array.from(table.querySelectorAll("th,td"))
      .map((cell) => normalizeWhitespace(cell.textContent ?? ""))
      .filter((text) => text.length >= 1 && text.length <= 120);
    if (cells.length >= 4) rows.push(`Matrix: ${Array.from(new Set(cells)).slice(0, 40).join(" | ")}`);
  }
  return rows.join("\n") || undefined;
}

function scoreQuestionContainer(node: HTMLElement, text: string): number {
  const attr = `${node.className} ${node.id} ${node.getAttribute("aria-label") ?? ""}`.toLowerCase();
  let score = 0;
  if (/question|survey|poll|matrix|answer|choice/.test(attr)) score += 3;
  if (/[?？]/.test(text)) score += 2;
  if (node.querySelector(optionSelector)) score += 2;
  if (node.querySelector("table")) score += 1;
  if (text.length > 20 && text.length < 4000) score += 1;
  return score;
}

function inferQuestionType(root: ParentNode, options: string[], matrix?: string): string | undefined {
  if (matrix) return "matrix";
  if (root.querySelector("[role='checkbox'], input[type='checkbox']")) return "multiple_choice";
  if (root.querySelector("[role='radio'], input[type='radio']")) return "single_choice";
  if (root.querySelector("textarea, input[type='text'], input:not([type])")) return "text";
  if (options.length >= 2) return "choice";
  return undefined;
}

function computeConfidence(input: { questionText?: string; optionsText?: string; progressText?: string; bestScore: number }): number {
  let score = Math.min(input.bestScore / 8, 0.5);
  if (input.questionText) score += 0.25;
  if (input.optionsText) score += 0.18;
  if (input.progressText) score += 0.07;
  return Math.max(0.05, Math.min(1, Number(score.toFixed(2))));
}

function readNodeText(node: Element): string {
  if (node instanceof HTMLInputElement) {
    return node.value || node.getAttribute("aria-label") || "";
  }
  return (node as HTMLElement).innerText || node.textContent || node.getAttribute("aria-label") || "";
}

function isVisible(node: Element): boolean {
  const element = node as HTMLElement;
  if (!element.isConnected) return false;
  const rect = element.getBoundingClientRect();
  const style = window.getComputedStyle(element);
  return rect.width > 0 && rect.height > 0 && style.visibility !== "hidden" && style.display !== "none";
}

function containsSensitiveNode(node: ParentNode): boolean {
  return Array.from(node.querySelectorAll<HTMLElement>("input,textarea")).some(isSensitiveInput);
}

function isSensitiveInput(node: HTMLElement): boolean {
  if (!(node instanceof HTMLInputElement || node instanceof HTMLTextAreaElement)) return false;

  const type = node instanceof HTMLInputElement ? node.type.toLowerCase() : "";
  const name = `${node.getAttribute("name") ?? ""} ${node.id} ${node.getAttribute("autocomplete") ?? ""}`.toLowerCase();
  return (
    type === "password" ||
    name.includes("password") ||
    name.includes("token") ||
    name.includes("cookie") ||
    name.includes("secret")
  );
}

function isNavigationText(text: string): boolean {
  return /^(next|back|previous|submit|continue|ok|cancel|下一步|上一步|提交|继续|确定|取消)$/i.test(text);
}

function findProgressText(pageText: string): string | undefined {
  const lines = pageText
    .split("\n")
    .map((line) => normalizeWhitespace(line))
    .filter(Boolean);
  return lines.find((line) => {
    if (line.length > 120) return false;
    return /(\d+\s*\/\s*\d+)|(\d+\s*%)|(progress|complete|completed|question\s+\d+|第\s*\d+\s*题|进度)/i.test(line);
  });
}

function findQuestionText(pageText: string): string | undefined {
  const lines = pageText
    .split("\n")
    .map((line) => normalizeWhitespace(line))
    .filter(Boolean)
    .filter((line) => !isNavigationText(line));

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
