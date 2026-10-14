import { expect, test, type Page } from "@playwright/test";
import type { Collection, LibraryPage, Section } from "../src/api/library";
const time = "2026-10-10T05:00:00Z";
const body =
  '## 核心结论\n\n自己实测的中文说明。\n\n|角色|建议|\n|---|---|\n|甲|先练防御|\n\n- [x] 已验证\n- [ ] 下次调整\n\n> 个人经验\n\n```go\nfmt.Println("hello")\n```';
async function mock(page: Page, empty = false) {
  const state = {
    collections: [] as Collection[],
    sections: [] as Section[],
    pages: [] as LibraryPage[],
    next: 10,
    fail: false,
    writes: 0,
  };
  if (!empty)
    state.collections = [
      {
        id: 1,
        name: "合成游戏 Wiki",
        description: "自己的实战资料",
        category: "游戏攻略",
        template: "wiki",
        display_mode: "wiki",
        cover_url: "",
        version: 1,
        created_at: time,
        updated_at: time,
      },
      {
        id: 2,
        name: "合成菜谱",
        description: "实际做过的味道",
        category: "烹饪",
        template: "recipe",
        display_mode: "list",
        cover_url: "",
        version: 1,
        created_at: time,
        updated_at: time,
      },
    ];
  await page.route("**/api/v1/**", async (route) => {
    const req = route.request(),
      url = new URL(req.url()),
      path = url.pathname,
      method = req.method();
    const reply = (data: unknown) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data }),
      });
    const error = (status: number, code: string, message: string) =>
      route.fulfill({
        status,
        contentType: "application/json",
        body: JSON.stringify({ error: { code, message } }),
      });
    if (path.endsWith("/auth/me"))
      return reply({
        id: 1,
        username: "test",
        role: "admin",
        status: "active",
      });
    if (!path.startsWith("/api/v1/library/collections")) return reply([]);
    const m = path.match(
      /^\/api\/v1\/library\/collections(?:\/(\d+)(?:\/(pages|sections|links|export)(?:\/(\d+))?)?)?$/,
    );
    if (!m) return error(404, "NOT_FOUND", "不存在");
    const col = Number(m[1]),
      kind = m[2],
      id = Number(m[3]);
    const input = ["POST", "PUT", "DELETE"].includes(method)
      ? req.postDataJSON()
      : undefined;
    if (method !== "GET") {
      state.writes++;
      if (state.fail) return error(500, "TEST_FAILED", "保存失败，正文保留");
    }
    if (!col) {
      if (method === "GET")
        return reply(
          state.collections.map((c) => ({
            ...c,
            page_count: state.pages.filter((p) => p.collection_id === c.id)
              .length,
          })),
        );
      const row = {
        ...input,
        id: state.next++,
        created_at: time,
        updated_at: time,
        version: 1,
      };
      state.collections.push(row);
      return reply(row);
    }
    let c = state.collections.find((c) => c.id === col);
    if (!c) return error(404, "LIBRARY_NOT_FOUND", "专题不可用");
    if (!kind) {
      if (method === "GET")
        return reply({
          collection: c,
          sections: state.sections.filter((s) => s.collection_id === col),
        });
      if (method === "PUT") {
        Object.assign(c, input, { version: c.version + 1 });
        return reply(c);
      }
      state.collections = state.collections.filter((c) => c.id !== col);
      state.sections = state.sections.filter((s) => s.collection_id !== col);
      state.pages = state.pages.filter((p) => p.collection_id !== col);
      return reply({ deleted: true });
    }
    if (kind === "links") {
      const states: Record<string, boolean> = {};
      url.searchParams
        .get("ids")
        ?.split(",")
        .forEach(
          (id) =>
            (states[id] = state.pages.some(
              (p) => p.id === Number(id) && p.collection_id === col,
            )),
        );
      return reply(states);
    }
    if (kind === "export")
      return route.fulfill({
        status: 200,
        contentType: "application/zip",
        headers: {
          "Content-Disposition": `attachment; filename="lifeos-library-${col}.zip"`,
        },
        body: Buffer.from("PK synthetic export"),
      });
    if (kind === "sections") {
      if (method === "POST") {
        const row = { id: state.next++, collection_id: col, ...input };
        state.sections.push(row);
        return reply(row);
      }
      if (method === "PUT") {
        const s = state.sections.find((s) => s.id === id)!;
        s.name = input.name;
        return reply(s);
      }
      const pages = state.pages.filter((p) => p.section_id === id);
      if (pages.length && !input.move_to_root)
        return error(409, "SECTION_NOT_EMPTY", "分组仍有页面");
      pages.forEach((p) => {
        p.section_id = null;
        p.version++;
      });
      state.sections = state.sections.filter((s) => s.id !== id);
      return reply({ deleted: true });
    }
    if (!id) {
      if (method === "POST") {
        const row = {
          ...input,
          id: state.next++,
          collection_id: col,
          section_id: input.section_id ?? null,
          tags: input.tags ?? [],
          sources: input.sources ?? [],
          cover_url: input.cover_url ?? "",
          version: 1,
          created_at: time,
          updated_at: time,
        };
        state.pages.push(row);
        return reply(row);
      }
      const q = url.searchParams.get("q") || "",
        section = url.searchParams.get("section"),
        p = Number(url.searchParams.get("page") || 1);
      const items = state.pages.filter(
        (p) =>
          p.collection_id === col &&
          (!section ||
            (section === "root"
              ? p.section_id === null
              : String(p.section_id) === section)) &&
          `${p.title} ${p.body} ${p.tags.join(" ")}`.includes(q),
      );
      return reply({
        items: items
          .slice((p - 1) * 50, p * 50)
          .map((p) => ({ ...p, summary: p.body.slice(0, 240) })),
        total: items.length,
        page: p,
        page_size: 50,
      });
    }
    const row = state.pages.find((p) => p.id === id && p.collection_id === col);
    if (!row) return error(404, "LIBRARY_NOT_FOUND", "页面已删除或不可访问");
    if (method === "GET") return reply(row);
    if (method === "PUT") {
      const allowed = [
        "title",
        "body",
        "section_id",
        "tags",
        "sources",
        "cover_url",
        "version",
      ];
      if (Object.keys(input).some((k) => !allowed.includes(k)))
        return error(400, "INVALID", "未知字段");
      if (input.version !== row.version)
        return error(409, "LIBRARY_CHANGED", "页面已更新");
      Object.assign(row, input, { version: row.version + 1 });
      return reply(row);
    }
    state.pages = state.pages.filter((p) => p.id !== id);
    return reply({ deleted: true });
  });
  return state;
}
function seed(
  state: Awaited<ReturnType<typeof mock>>,
  id = 3,
  col = 1,
  title = "第四章攻略",
  content = body,
  section: number | null = null,
) {
  const row: LibraryPage = {
    id,
    collection_id: col,
    section_id: section,
    title,
    body: content,
    tags: ["实测"],
    sources: [
      { url: "https://example.com/guide", name: "原始攻略", note: "仅作参考" },
    ],
    cover_url: "",
    version: 1,
    created_at: time,
    updated_at: time,
  };
  state.pages.push(row);
  return row;
}
async function createTopic(page: Page, name: string, template: string) {
  await page.getByRole("button", { name: "新建专题", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("专题名称").fill(name);
  await dialog.locator(".el-select").first().click();
  await page.getByRole("option", { name: template, exact: true }).click();
  await dialog.getByRole("button", { name: "保存专题" }).click();
  await expect(dialog).not.toBeVisible();
}
test("Cards 独立阅读、返回、刷新、编辑及前进后退保留视图与搜索", async ({ page }) => {
  const state = await mock(page);
  seed(state, 1, 2, "合成菜品", "## 做法\n\n自己的心得");
  await page.goto("/explore/library/2?q=合成");
  await expect(page).toHaveURL(/view=list/);
  await page.locator(".library-page-link").first().click();
  await expect(page.locator(".library-reader-header h2")).toHaveText("合成菜品");
  await expect(page.locator(".library-directory")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "卡片清单", exact: true })).toHaveAttribute("aria-pressed", "true");
  await page.reload();
  await expect(page.locator(".library-directory")).toHaveCount(0);
  await page.getByRole("button", { name: "编辑页面", exact: true }).click();
  await page.getByLabel("页面标题").fill("合成菜品改名");
  await page.getByRole("button", { name: "保存页面", exact: true }).click();
  await expect(page.locator(".library-reader-header h2")).toHaveText("合成菜品改名");
  await expect(page.locator(".library-directory")).toHaveCount(0);
  await page.getByRole("button", { name: "卡片清单", exact: true }).click();
  await expect(page.locator(".library-page-link")).toBeVisible();
  await expect(page).toHaveURL(/\/library\/2\?q=.*view=list/);
  await page.goBack();
  await expect(page.locator(".library-reader-header h2")).toHaveText("合成菜品改名");
  await expect(page.locator(".library-directory")).toHaveCount(0);
  await page.goForward();
  await expect(page.locator(".library-page-link")).toBeVisible();
  await page.locator(".library-page-link").click();
  await page.getByRole("button", { name: "返回卡片画廊" }).click();
  await expect(page.locator(".library-page-link")).toBeVisible();
});

test("Wiki 页面双向切换与历史来源独立于专题默认模式", async ({ page }) => {
  const state = await mock(page);
  state.sections.push({ id: 1, collection_id: 1, name: "章节" }, { id: 2, collection_id: 1, name: "角色" });
  seed(state, 1, 1, "第四章", "## 说明", 1);
  seed(state, 2, 1, "角色培养", "## 培养", 2);
  await page.goto("/explore/library/1/pages/1");
  await expect(page.locator(".library-reader-header h2")).toHaveText("第四章");
  await expect(page.locator(".library-directory")).toBeVisible();
  await page.locator(".library-page-link").filter({ hasText: "角色培养" }).click();
  await expect(page.locator(".library-reader-header h2")).toHaveText("角色培养");
  await expect(page.locator(".library-directory")).toBeVisible();
  await page.getByRole("button", { name: "卡片清单", exact: true }).click();
  await expect(page.locator(".library-reader-header")).toHaveCount(0);
  await expect(page.locator(".library-page-link")).toHaveCount(2);
  await page.goBack();
  await expect(page.locator(".library-reader-header h2")).toHaveText("角色培养");
  await expect(page.getByRole("button", { name: "Wiki 目录", exact: true })).toHaveAttribute("aria-pressed", "true");
  await page.goForward();
  await page.locator(".library-page-link").filter({ hasText: "第四章" }).click();
  await expect(page.locator(".library-directory")).toHaveCount(0);
  await page.getByRole("button", { name: "Wiki 目录", exact: true }).click();
  await expect(page.locator(".library-reader-header h2")).toHaveText("第四章");
  await expect(page.locator(".library-directory")).toBeVisible();
  await page.reload();
  await expect(page.locator(".library-directory")).toBeVisible();
});

test("Wiki 与菜谱处于同一级，空专题不自动写入页面，菜谱根目录模板可保存", async ({
  page,
}) => {
  const state = await mock(page, true);
  await page.goto("/explore/library");
  await expect(
    page.getByText("先为一个常用主题建立自己的资料库"),
  ).toBeVisible();
  await createTopic(page, "合成攻略", "Wiki 资料库");
  await createTopic(page, "合成菜谱", "个人菜谱");
  await expect(page.locator(".library-collection-card")).toHaveCount(2);
  expect(state.pages).toHaveLength(0);
  await page.getByRole("link", { name: /合成菜谱/ }).click();
  await page.getByRole("button", { name: "新建页面", exact: true }).click();
  await expect(
    page.getByRole("textbox", { name: "Markdown 正文", exact: true }),
  ).toHaveValue(/保存与复热/);
  await page.getByLabel("页面标题").fill("合成菜品");
  await page.getByRole("button", { name: "保存页面" }).click();
  await expect(page.locator(".library-reader-header h2")).toHaveText(
    "合成菜品",
  );
  expect(state.pages[0]?.section_id).toBeNull();
  await page.reload();
  await expect(
    page.locator(".library-markdown h2").filter({ hasText: "我的改良心得" }),
  ).toBeVisible();
});
test("分组下编辑攻略，表格清单来源可读，保存失败保留正文，移动和改名保持内部链接", async ({
  page,
}) => {
  const state = await mock(page);
  state.sections = [
    { id: 4, collection_id: 1, name: "章节攻略" },
    { id: 5, collection_id: 1, name: "角色培养" },
  ];
  seed(state, 3, 1, "第四章攻略", body, 4);
  seed(state, 6, 1, "新手须知", "[第四章](/explore/library/1/pages/3)");
  await page.goto("/explore/library/1/pages/3");
  await expect(page.locator(".library-markdown table")).toBeVisible();
  await expect(
    page.locator(".library-markdown input[type=checkbox]"),
  ).toHaveCount(2);
  await expect(page.locator(".library-markdown input").first()).toBeDisabled();
  await expect(page.getByRole("link", { name: "原始攻略" })).toHaveAttribute(
    "href",
    "https://example.com/guide",
  );
  await page.getByRole("button", { name: "编辑页面" }).click();
  await page.getByLabel("页面标题").fill("修改后的攻略");
  await page
    .getByRole("textbox", { name: "Markdown 正文", exact: true })
    .fill(body + "\n\n新心得");
  await page.locator(".library-editor .el-select").first().click();
  await page.getByRole("option", { name: "角色培养", exact: true }).click();
  state.fail = true;
  await page.getByRole("button", { name: "保存页面" }).click();
  await expect(page.getByText("保存失败，正文保留")).toBeVisible();
  await expect(
    page.getByRole("textbox", { name: "Markdown 正文", exact: true }),
  ).toHaveValue(body + "\n\n新心得");
  state.fail = false;
  await page.getByRole("button", { name: "保存页面" }).click();
  await expect(page.locator(".library-reader-header h2")).toHaveText(
    "修改后的攻略",
  );
  expect(state.pages.find((p) => p.id === 3)?.section_id).toBe(5);
  await page.goto("/explore/library/1/pages/6");
  await page
    .locator(".library-markdown")
    .getByRole("link", { name: "第四章", exact: true })
    .click();
  await expect(page).toHaveURL(/\/pages\/3/);
  await expect(page.locator(".library-reader-header h2")).toHaveText(
    "修改后的攻略",
  );
  await page.getByRole("button", { name: "复制内部链接" }).click();
  await expect(page.locator(".library-reader-header code")).toHaveText(
    "/explore/library/1/pages/3",
  );
});
test("非空分组明确移回根目录，中文搜索保持 URL，大量页面分页与导出可用", async ({
  page,
}) => {
  const state = await mock(page);
  state.sections = [{ id: 4, collection_id: 1, name: "章节" }];
  seed(state, 3, 1, "中文标题", body, 4);
  for (let i = 0; i < 55; i++)
    seed(state, 20 + i, 1, `批量页面${i}`, "独有中文正文", 4);
  await page.goto("/explore/library/1");
  await expect(page.locator(".library-page-link")).toHaveCount(50);
  await page.locator(".el-pagination .btn-next").click();
  await expect(page.locator(".library-page-link")).toHaveCount(6);
  await page.getByLabel("专题内搜索").fill("独有中文正文");
  await page.getByRole("button", { name: "搜索", exact: true }).click();
  await expect(page).toHaveURL(/q=/);
  await expect(page.locator(".library-page-link")).toHaveCount(50);
  await page.getByLabel("专题内搜索").fill("中文标题");
  await page.getByRole("button", { name: "搜索", exact: true }).click();
  await expect(page.locator(".library-page-link")).toHaveCount(1);
  await page.getByLabel("删除分组 章节").click();
  await page.getByRole("button", { name: "确认", exact: true }).click();
  await expect(
    page.getByText("该分组仍有页面。将全部页面移回专题根目录，然后删除分组？"),
  ).toBeVisible();
  await page
    .getByRole("dialog", { name: "保留页面并移回根目录" })
    .getByRole("button", { name: "确认", exact: true })
    .click();
  await expect(
    page.getByRole("link", { name: "章节", exact: true }),
  ).toHaveCount(0);
  expect(state.pages.every((p) => p.section_id === null)).toBe(true);
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "导出", exact: true }).click();
  expect((await download).suggestedFilename()).toBe("lifeos-library-1.zip");
  await expect(page.getByLabel("专题内搜索")).toHaveValue("中文标题");
});
test("未保存离开确认、危险来源保留输入、删除页面使内部链接明确失效", async ({
  page,
}) => {
  const state = await mock(page);
  seed(state);
  seed(state, 6, 1, "目录页", "[攻略](/explore/library/1/pages/3)");
  await page.goto("/explore/library/1/pages/3");
  await page.getByRole("button", { name: "编辑页面" }).click();
  await page
    .getByRole("textbox", { name: "Markdown 正文", exact: true })
    .fill("尚未保存");
  await page.getByRole("link", { name: "我的资料库", exact: true }).click();
  await page.getByRole("button", { name: "继续编辑", exact: true }).click();
  await expect(
    page.getByRole("textbox", { name: "Markdown 正文", exact: true }),
  ).toHaveValue("尚未保存");
  await page.getByLabel("来源 1 URL").fill("javascript:alert(1)");
  await page.getByRole("button", { name: "保存页面" }).click();
  await expect(
    page.getByText(
      "请检查来源地址，支持 HTTP、HTTPS 或有效的 Obsidian 打开链接。",
    ),
  ).toBeVisible();
  await page.getByRole("button", { name: "取消编辑" }).click();
  await page.getByRole("button", { name: "放弃修改", exact: true }).click();
  await page.getByRole("button", { name: "删除页面", exact: true }).click();
  await page.getByRole("button", { name: "确认", exact: true }).click();
  await page.getByRole("link", { name: /目录页/ }).click();
  await expect(page.getByText("攻略（页面已删除或不可访问）")).toBeVisible();
  await expect(page.locator(".library-markdown a")).toHaveCount(0);
});
for (const width of [1440, 390])
  test(`资料库 ${width}px Markdown 安全渲染与阅读预览无溢出`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 900 });
    const state = await mock(page);
    seed(
      state,
      3,
      1,
      "很长的攻略名称".repeat(8),
      body +
        '\n\n<script>window.__wikiXSS=1</script>\n\n<img src="javascript:alert(1)" onerror="window.__wikiXSS=1">\n\n[危险](javascript:alert(1))\n\n<iframe src="https://example.com"></iframe>',
    );
    await page.goto("/explore/library/1/pages/3");
    await expect(page.locator(".library-markdown table")).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth + 1,
      ),
    ).toBe(true);
    expect(
      await page.evaluate(
        () => (window as unknown as Record<string, unknown>).__wikiXSS,
      ),
    ).toBeUndefined();
    await expect(
      page.locator(
        ".library-markdown script,.library-markdown iframe,.library-markdown [onerror]",
      ),
    ).toHaveCount(0);
    await expect(
      page.locator('.library-markdown a[href^="javascript:"]'),
    ).toHaveCount(0);
    await page.getByRole("button", { name: "编辑页面" }).click();
    await page.getByText("阅读预览", { exact: true }).click();
    await expect(page.locator(".library-markdown table")).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth + 1,
      ),
    ).toBe(true);
    if (process.env.CAPTURE_VISUALS)
      await page.screenshot({
        path: info.outputPath(`library-${width}.png`),
        fullPage: true,
      });
  });

test("Wiki 新建分组及分组页面，模板可删改，切换展示模式不改变内容", async ({
  page,
}) => {
  const state = await mock(page);
  await page.goto("/explore/library/1");
  await page.getByRole("button", { name: "新建分组" }).click();
  await page.getByPlaceholder("分组名称").fill("新章节");
  await page
    .getByRole("dialog", { name: "新建分组" })
    .getByRole("button", { name: "创建", exact: true })
    .click();
  await page.getByRole("link", { name: "新章节", exact: true }).click();
  await page.getByRole("button", { name: "新建页面", exact: true }).click();
  await page.getByLabel("页面标题").fill("新攻略");
  await expect(
    page.getByRole("textbox", { name: "Markdown 正文", exact: true }),
  ).toHaveValue(/我的实战心得/);
  await page
    .getByRole("textbox", { name: "Markdown 正文", exact: true })
    .fill("自己的自由正文");
  await page.getByRole("button", { name: "保存页面" }).click();
  await expect(page.locator(".library-reader-header h2")).toHaveText("新攻略");
  expect(state.pages[0]?.section_id).toBe(state.sections[0]?.id);
  await page.getByText("卡片清单", { exact: true }).click();
  await expect(page.locator(".library-topic-layout")).toHaveClass(/is-list/);
  expect(state.pages[0]?.body).toBe("自己的自由正文");
  await page.getByLabel("重命名分组 新章节").click();
  const prompt = page.getByRole("dialog", { name: "重命名分组" });
  await prompt.getByRole("textbox").fill("调整后的章节");
  await prompt.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByRole("link", { name: "调整后的章节", exact: true }),
  ).toBeVisible();
  expect(state.pages[0]?.body).toBe("自己的自由正文");
});

test("深色主题菜谱清单无需分组，URL 图片与专题封面可阅读", async ({
  page,
}, info) => {
  await page.addInitScript(() => localStorage.setItem("learnos-theme", "dark"));
  await page.route("https://example.com/image.png", (r) =>
    r.fulfill({
      contentType: "image/png",
      body: Buffer.from(
        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jpV8AAAAASUVORK5CYII=",
        "base64",
      ),
    }),
  );
  const state = await mock(page);
  const row = seed(
    state,
    4,
    2,
    "合成菜品",
    "## 食材与用量\n\n- 测试食材\n\n## 制作步骤\n\n1. 合成步骤\n\n![制作示意](https://example.com/image.png)\n\n## 我的改良心得\n\n自己的经验",
  );
  row.cover_url = "https://example.com/image.png";
  await page.goto("/explore/library/2");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.locator(".library-page-link")).toHaveCount(1);
  await page.locator(".library-page-link").click();
  await expect(page.locator(".library-markdown img")).toHaveAttribute(
    "src",
    "https://example.com/image.png",
  );
  await expect(page.locator(".library-markdown img")).toHaveAttribute(
    "referrerpolicy",
    "no-referrer",
  );
  await expect(
    page.locator(".library-markdown h2").filter({ hasText: "我的改良心得" }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth + 1,
    ),
  ).toBe(true);
  if (process.env.CAPTURE_VISUALS)
    await page.screenshot({
      path: info.outputPath("library-recipe-dark.png"),
      fullPage: true,
    });
});

for (const width of [1920, 1440, 390]) {
  test(`gallery reading keeps covers, content and navigation distinct at ${width}px`, async ({ page }, info) => {
    await page.setViewportSize({ width, height: 960 });
    const state = await mock(page);
    const image = 'https://example.com/gallery.png';
    await page.route(image, (route) => route.fulfill({ contentType: 'image/png', body: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jpV8AAAAASUVORK5CYII=', 'base64') }));
    const original = `# 菜名\n\n![菜品](${image})\n\n## 食材与用量\n\n- 番茄\n- 鸡蛋\n\n|用料|分量|\n|---|---|\n|番茄|两个|`;
    const row = seed(state, 31, 2, '番茄炒鸡蛋', original);
    row.cover_url = image;
    seed(state, 32, 2, '无封面的菜谱', '## 不该成为卡片摘要的标题');
    await page.goto('/explore/library/2?view=list');
    const cards = page.locator('.library-page-link');
    await expect(cards).toHaveCount(2);
    await expect(cards.locator('p')).toHaveCount(0);
    await expect(cards.nth(1)).toContainText('暂无封面');
    const first = await cards.first().boundingBox();
    expect(first!.width).toBeLessThanOrEqual(320);
    if (width > 800) {
      const second = await cards.nth(1).boundingBox();
      expect(second!.y).toBe(first!.y);
      expect(first!.x).toBe((await page.locator('.library-directory').boundingBox())!.x);
    }
    if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: info.outputPath(`gallery-${width}.png`), fullPage: true });
    await cards.first().focus();
    await page.keyboard.press('Enter');
    await expect(page.locator('.library-markdown table')).toBeVisible();
    await expect(page.locator('.library-directory')).toHaveCount(0);
    await expect(page.locator('.library-reader-cover')).toHaveCount(0);
    await expect(page.locator('.library-markdown img')).toHaveCount(1);
    expect(row.body).toBe(original);
    await page.reload();
    await expect(page.locator('.library-markdown img')).toHaveCount(1);
    if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: info.outputPath(`recipe-${width}.png`), fullPage: true });
    await page.getByRole('button', { name: '返回卡片画廊' }).click();
    await expect(cards).toHaveCount(2);
    await page.goBack();
    await expect(page.locator('.library-markdown table')).toBeVisible();
    await page.getByRole('button', { name: 'Wiki 目录', exact: true }).click();
    await expect(page.locator('.library-directory')).toBeVisible();
    await expect(page.locator('.library-markdown table')).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
    if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: info.outputPath(`wiki-${width}.png`), fullPage: true });
    await page.getByRole('button', { name: '卡片清单', exact: true }).click();
    await expect(cards).toHaveCount(2);
    await expect(page.locator('.library-reader')).toHaveCount(0);
  });
}

test('failed gallery cover keeps a stable frame and reference-style body cover is not duplicated', async ({ page }) => {
  const state = await mock(page);
  const image = 'https://example.com/missing-cover.png';
  await page.route(image, route => route.fulfill({ status: 404 }));
  const original = `![正文图片][cover]\n\n[cover]: ${image}`;
  const row = seed(state, 41, 2, '封面失败仍可阅读', original);
  row.cover_url = image;
  await page.goto('/explore/library/2?view=list');
  const card = page.locator('.library-page-link');
  await expect(card).toContainText('封面暂时无法加载');
  const frame = await card.locator('.library-card-cover').boundingBox();
  expect(frame!.width / frame!.height).toBeCloseTo(4 / 3, 1);
  await card.click();
  await expect(page.locator('.library-reader-cover')).toHaveCount(0);
  await expect(page.locator('.library-image-failed')).toBeVisible();
  expect(row.body).toBe(original);
});
