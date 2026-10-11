import { describe, it, expect } from "vitest";
import { pageTemplate } from "./library";
import { internalPageID } from "../utils/libraryMarkdown";
describe("资料库模板与稳定链接", () => {
  it("模板只是可以自由编辑的 Markdown", () => {
    expect(pageTemplate("recipe")).toContain("## 保存与复热");
    expect(pageTemplate("wiki")).toContain("## 我的实战心得");
    expect(pageTemplate("general")).toContain("## 个人补充");
  });
  it("只识别同专题的稳定页面 ID", () => {
    expect(internalPageID("/explore/library/2/pages/7", 2)).toBe(7);
    for (const url of [
      "/explore/library/3/pages/7",
      "javascript:alert(1)",
      "//evil.test",
      "/explore/library/2/pages/0",
      "/explore/library/2/pages/4294967296",
    ])
      expect(internalPageID(url, 2)).toBeNull();
  });
});
