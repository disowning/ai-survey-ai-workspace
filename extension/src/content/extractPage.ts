import { extractDefaultPage } from "./siteAdapters/default";
import type { ExtractedPage } from "../shared/types";

export function extractPage(): ExtractedPage {
  return extractDefaultPage();
}
