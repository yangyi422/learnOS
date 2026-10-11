import { request } from "./http";
export interface Collection {
  id: number;
  name: string;
  description: string;
  category: string;
  template: "general" | "wiki" | "recipe";
  display_mode: "wiki" | "list";
  cover_url: string;
  version: number;
  created_at: string;
  updated_at: string;
  page_count?: number;
}
export interface Section {
  id: number;
  collection_id: number;
  name: string;
}
export interface Source {
  url: string;
  name: string;
  note: string;
}
export interface LibraryPage {
  id: number;
  collection_id: number;
  section_id: number | null;
  title: string;
  body: string;
  tags: string[];
  sources: Source[];
  cover_url: string;
  version: number;
  created_at: string;
  updated_at: string;
}
export interface PageRow {
  id: number;
  section_id: number | null;
  title: string;
  summary: string;
  tags: string[];
  cover_url: string;
  updated_at: string;
}
export type CollectionInput = Pick<
  Collection,
  | "name"
  | "description"
  | "category"
  | "template"
  | "display_mode"
  | "cover_url"
  | "version"
>;
export type PageInput = Pick<
  LibraryPage,
  "title" | "body" | "section_id" | "tags" | "sources" | "cover_url" | "version"
>;
async function data<T>(path: string, method = "GET", input?: unknown) {
  return (
    await request<{ data: T }>(`/api/v1/library/collections${path}`, {
      method,
      body: input === undefined ? undefined : JSON.stringify(input),
    })
  ).data;
}
export const listCollections = () => data<Collection[]>("");
export const getCollection = (id: number) =>
  data<{ collection: Collection; sections: Section[] }>(`/${id}`);
export const saveCollection = (
  id: number | undefined,
  input: CollectionInput,
) => data<Collection>(id ? `/${id}` : "", id ? "PUT" : "POST", input);
export const deleteCollection = (id: number, version: number) =>
  data(`/${id}`, "DELETE", { confirm: true, version });
export const saveSection = (
  col: number,
  id: number | undefined,
  name: string,
) =>
  data<Section>(`/${col}/sections${id ? `/${id}` : ""}`, id ? "PUT" : "POST", {
    name,
  });
export const deleteSection = (col: number, id: number, move: boolean) =>
  data(`/${col}/sections/${id}`, "DELETE", {
    confirm: true,
    move_to_root: move,
  });
export const listPages = (
  col: number,
  q: string,
  section: string,
  page: number,
) =>
  data<{ items: PageRow[]; total: number; page: number; page_size: number }>(
    `/${col}/pages?${new URLSearchParams({ q, section, page: String(page) })}`,
  );
export const getPage = (col: number, id: number) =>
  data<LibraryPage>(`/${col}/pages/${id}`);
export const savePage = (
  col: number,
  id: number | undefined,
  input: PageInput,
) =>
  data<LibraryPage>(
    `/${col}/pages${id ? `/${id}` : ""}`,
    id ? "PUT" : "POST",
    input,
  );
export const deletePage = (col: number, id: number, version: number) =>
  data(`/${col}/pages/${id}`, "DELETE", { confirm: true, version });
export async function exportCollection(id: number) {
  const response = await fetch(`/api/v1/library/collections/${id}/export`);
  if (!response.ok) {
    if (response.status === 401) window.location.assign("/login");
    throw new Error("专题导出失败，请重试。");
  }
  const url = URL.createObjectURL(await response.blob());
  const a = document.createElement("a");
  a.href = url;
  a.download = `lifeos-library-${id}.zip`;
  a.click();
  URL.revokeObjectURL(url);
}
export function pageTemplate(type: Collection["template"]) {
  return {
    general: "## 内容概要\n\n\n## 正文\n\n\n## 参考资料\n\n\n## 个人补充\n\n",
    wiki: "# 攻略标题\n\n## 适用条件\n\n路线、难度、章节等。\n\n## 核心结论\n\n\n## 详细说明\n\n\n## 参考资料\n\n\n## 我的实战心得\n\n",
    recipe:
      "# 菜名\n\n## 食材与用量\n\n- \n\n## 制作步骤\n\n1. \n\n## 注意事项\n\n\n## 保存与复热\n\n\n## 我的改良心得\n\n",
  }[type];
}

export const checkPageLinks = (col: number, ids: number[]) =>
  data<Record<string, boolean>>(`/${col}/links?ids=${ids.join(",")}`);
