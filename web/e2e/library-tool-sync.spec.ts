import { expect, test, type Page } from "@playwright/test";

// 挖角手帐的账号存档：404 表示账号没有存档，200 表示读到已有存档，
// PUT 携带 revision 做乐观锁。只 mock /api/v1/library-tools/*，其余请求交给真实 SPA。
const TOOL_URL = "/library-tools/fefw-planner/planner.html?from=library";
const STATE_URL = "**/api/v1/library-tools/fefw-planner/state";

async function openPlanner(page: Page) {
  await page.addInitScript(() => localStorage.setItem("learnos-theme", "light"));
  await page.goto(TOOL_URL);
  await expect(page.locator("#saveText")).toBeVisible();
}

async function openOverview(page: Page) {
  await page.locator('[data-view="overview"]').click();
  await expect(page.locator("#overviewView")).toBeVisible();
}

// 切换路线会写入 state 并触发 queueSave，是最轻量的“发生编辑”方式。
async function editSomething(page: Page) {
  await page.locator(".route-tab").nth(1).click();
}

test("账号没有存档时给出上传入口，上传后写入 revision 1", async ({ page }) => {
  const puts: Array<{ state: unknown; revision: number }> = [];
  await page.route(STATE_URL, async route => {
    if (route.request().method() === "PUT") {
      puts.push(route.request().postDataJSON());
      return route.fulfill({ json: { data: { tool_key: "fefw-planner", state: puts.at(-1)!.state, revision: 1, updated_at: "2026-10-29T05:00:00Z" } } });
    }
    return route.fulfill({ status: 404, json: { error: { code: "TOOL_STATE_NOT_FOUND", message: "账号里还没有保存过这个工具的规划。", retryable: false } } });
  });

  await openPlanner(page);
  await expect(page.locator("#saveText")).toBeVisible();

  await editSomething(page);
  await openOverview(page);
  await expect(page.locator("#syncStatus")).toHaveText(/上传到账号|自动保存到账号/);

  await page.getByRole("button", { name: "上传本机到账号" }).click();
  await expect(page.locator("#syncStatus")).toHaveText(/已保存到账号/);
  expect(puts.length).toBeGreaterThan(0);
  expect(puts[0].revision).toBe(0);
  expect(puts[0].state).toMatchObject({ version: 1 });
});

test("账号已有存档时直接载入，并按 revision 保存", async ({ page }) => {
  const remoteState = {
    version: 1,
    activeRoute: "kai",
    hideSkip: false,
    search: "",
    fameFilter: "",
    plans: {},
  };
  const puts: Array<{ revision: number }> = [];
  await page.route(STATE_URL, async route => {
    if (route.request().method() === "PUT") {
      const payload = route.request().postDataJSON();
      puts.push(payload);
      return route.fulfill({ json: { data: { tool_key: "fefw-planner", state: remoteState, revision: payload.revision + 1, updated_at: "2026-10-29T05:00:00Z" } } });
    }
    return route.fulfill({ json: { data: { tool_key: "fefw-planner", state: remoteState, revision: 3, updated_at: "2026-10-29T05:00:00Z" } } });
  });

  await openPlanner(page);
  await openOverview(page);
  await expect(page.locator("#syncStatus")).toHaveText(/已载入账号存档/);

  await editSomething(page);
  await expect.poll(() => puts.length, { timeout: 5000 }).toBeGreaterThan(0);
  expect(puts.at(-1)!.revision).toBe(3);
});

test("未登录时只在本机保存，不发起写入", async ({ page }) => {
  const puts: string[] = [];
  await page.route(STATE_URL, async route => {
    if (route.request().method() === "PUT") {
      puts.push(route.request().url());
      return route.fulfill({ status: 401, json: { error: { code: "UNAUTHORIZED" } } });
    }
    return route.fulfill({ status: 401, json: { error: { code: "UNAUTHORIZED" } } });
  });

  await openPlanner(page);
  await editSomething(page);
  await openOverview(page);
  await expect(page.locator("#syncStatus")).toHaveText(/未登录/);
  expect(puts).toHaveLength(0);
});

test("revision 冲突时停止自动保存并提示载入账号版本", async ({ page }) => {
  const remoteState = { version: 1, activeRoute: "kai", hideSkip: false, search: "", fameFilter: "", plans: {} };
  const puts: number[] = [];
  await page.route(STATE_URL, async route => {
    if (route.request().method() === "PUT") {
      puts.push(route.request().postDataJSON().revision);
      return route.fulfill({ status: 409, json: { error: { code: "TOOL_STATE_CONFLICT", message: "这份规划已在别处更新，请重新载入后再修改。", retryable: false } } });
    }
    return route.fulfill({ json: { data: { tool_key: "fefw-planner", state: remoteState, revision: 5, updated_at: "2026-10-29T05:00:00Z" } } });
  });

  await openPlanner(page);
  await editSomething(page);
  await expect.poll(() => puts.length, { timeout: 5000 }).toBeGreaterThan(0);
  await openOverview(page);
  await expect(page.locator("#syncStatus")).toHaveText(/已在别处更新/);

  // 冲突后不再反复重试写入。
  const afterConflict = puts.length;
  await editSomething(page);
  await page.waitForTimeout(1500);
  expect(puts.length).toBe(afterConflict);

  // 载入账号版本后恢复可写（切换路线会回到路线视图，需要再打开总览）。
  await page.locator('[data-view="overview"]').click();
  await page.getByRole("button", { name: "载入账号版本" }).click();
  await expect(page.locator("#syncStatus")).toHaveText(/已载入账号存档/);
});

// 桌面端没有标签页：target="_blank" 会被交给系统浏览器打开，离开应用。
// 桌面 shell 用 ?shell=desktop 标记自己，工具链接改为同窗口打开。
async function mockLibrary(page: Page) {
  await page.route("**/api/v1/auth/me", route =>
    route.fulfill({
      json: { data: { id: 1, username: "probe", display_name: "Probe", role: "admin", status: "active", created_at: "2026-10-01T00:00:00Z" } },
    }),
  );
  await page.route("**/api/v1/library/collections**", route =>
    route.fulfill({ json: { data: [] } }),
  );
}

test("桌面 shell 内工具链接同窗口打开，不交给系统浏览器", async ({ page }) => {
  await mockLibrary(page);
  await page.goto("/explore/library?shell=desktop");
  const link = page.getByRole("link", { name: /万缕千丝 · 挖角手帐/ });
  await expect(link).toBeVisible();
  await expect(link).not.toHaveAttribute("target");
  // 会话标记要能跨整页跳转保留（401 → /login 之类的重定向）。
  expect(await page.evaluate(() => sessionStorage.getItem("learnos-shell"))).toBe("desktop");

  await Promise.all([page.waitForURL(/planner\.html/), link.click()]);
  await expect(page.locator("#saveText")).toBeVisible();
});

test("web 端保持新标签页行为", async ({ page }) => {
  await mockLibrary(page);
  await page.goto("/explore/library");
  await expect(page.getByRole("link", { name: /万缕千丝 · 挖角手帐/ })).toHaveAttribute(
    "target",
    "_blank",
  );
  expect(await page.evaluate(() => sessionStorage.getItem("learnos-shell"))).toBeNull();
});
