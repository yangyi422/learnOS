import { marked } from "marked";
import DOMPurify from "dompurify";
import { safeExternalLink } from "./externalLinks";
export function localLibraryImage(src: string): boolean {
  return /^\/api\/v1\/library\/images\/[a-f0-9]{32}$/.test(src);
}
export function internalPageID(
  href: string,
  collection: number,
): number | null {
  const m = href.match(/^\/explore\/library\/(\d+)\/pages\/(\d+)$/);
  return m &&
    Number(m[1]) === collection &&
    Number(m[2]) > 0 &&
    Number(m[2]) <= 4294967295
    ? Number(m[2])
    : null;
}
export function markdownHTML(
  body: string,
  collection: number,
  links: Map<number, boolean> = new Map(),
) {
  const clean = DOMPurify.sanitize(
    marked.parse(body, { async: false, gfm: true }),
    {
      RETURN_DOM_FRAGMENT: true,
      ALLOWED_TAGS: [
        "p",
        "h1",
        "h2",
        "h3",
        "h4",
        "h5",
        "h6",
        "ul",
        "ol",
        "li",
        "blockquote",
        "pre",
        "code",
        "a",
        "img",
        "table",
        "thead",
        "tbody",
        "tr",
        "th",
        "td",
        "strong",
        "em",
        "del",
        "hr",
        "br",
        "input",
      ],
      ALLOWED_ATTR: [
        "href",
        "src",
        "alt",
        "title",
        "type",
        "checked",
        "disabled",
        "start",
      ],
      ALLOW_DATA_ATTR: false,
    },
  );
  clean.querySelectorAll("a").forEach((a) => {
    const href = a.getAttribute("href") || "";
    const id = internalPageID(href, collection);
    if (id) {
      a.setAttribute("data-library-page", String(id));
      if (links.get(id) === false) {
        const span = document.createElement("span");
        span.className = "library-broken-link";
        span.textContent = `${a.textContent}（页面已删除或不可访问）`;
        a.replaceWith(span);
      } else if (!links.has(id)) {
        a.removeAttribute("href");
        a.setAttribute("aria-disabled", "true");
        a.title = "正在检查内部链接";
      }
    } else if (safeExternalLink(href)) {
      a.setAttribute("rel", "noopener noreferrer");
      if (!href.toLowerCase().startsWith("obsidian:"))
        a.setAttribute("target", "_blank");
    } else {
      a.removeAttribute("href");
    }
  });
  clean.querySelectorAll("img").forEach((img) => {
    const src = img.getAttribute("src") || "";
    if (
      !localLibraryImage(src) &&
      (!/^https?:\/\//i.test(src) || !safeExternalLink(src))
    ) {
      img.remove();
      return;
    }
    img.setAttribute("loading", "lazy");
    img.setAttribute("tabindex", "0");
    img.setAttribute("role", "button");
    img.setAttribute(
      "aria-label",
      `查看大图：${img.getAttribute("alt") || "图片"}`,
    );
    img.setAttribute("referrerpolicy", "no-referrer");
  });
  clean.querySelectorAll("input").forEach((input) => {
    if (input.getAttribute("type") !== "checkbox") {
      input.remove();
      return;
    }
    input.setAttribute("disabled", "");
  });
  const div = document.createElement("div");
  div.append(clean);
  return div.innerHTML;
}
