import { expect, test, type Page } from "@playwright/test";

// 快速采集凭证面板：生成时只显示一次明文，列表只保留前缀，撤销后标记状态。
// 只 mock /api/v1/api-tokens 与设置页依赖的只读接口。
const USER = { id: 1, username: "probe", display_name: "Probe", role: "admin", status: "active", created_at: "2026-10-01T00:00:00Z" };

async function mockSettings(page: Page, tokens: any[]) {
  const state = { tokens: tokens.map(t => ({ ...t })), created: 0 };
  await page.route("**/api/v1/auth/me", route => route.fulfill({ json: { data: USER } }));
  await page.route("**/api/v1/system/diagnostics", route =>
    route.fulfill({
      json: {
        data: {
          app_version: "0.1.0", environment: "production", schema_version: 33,
          database_status: { status: "ok" }, database_path: "/app/data/learnos.db", database_size: 1024,
          last_backup: null, ai_provider: "mock", ai_model: "mock", ai_timeouts_seconds: {},
          course_count: 0, lesson_count: 0, learning_turn_count: 0, cognitive_state_count: 0,
          generated_at: "2026-10-29T06:00:00Z",
        },
      },
    }),
  );
  await page.route("**/api/v1/system/ai-config", route =>
    route.fulfill({
      json: {
        data: {
          provider: "mock", api_key_configured: false, base_url: "https://api.deepseek.com", model: "deepseek-chat",
          effective_provider: "mock", effective_model: "mock", last_successful_call_at: null,
        },
      },
    }),
  );
  await page.route("**/api/v1/system/backups", route => route.fulfill({ json: { data: [] } }));
  await page.route("**/api/v1/api-tokens", async route => {
    if (route.request().method() === "POST") {
      state.created += 1;
      const created = {
        id: 10 + state.created, name: "uTools 采集", prefix: "los_pat_ab12",
        created_at: "2026-10-29T06:00:00Z", last_used_at: null,
        expires_at: "2027-01-27T06:00:00Z", revoked_at: null, expired: false,
        token: "los_pat_ab12cd34ef56",
      };
      state.tokens.unshift({ ...created });
      return route.fulfill({ status: 201, json: { data: created } });
    }
    return route.fulfill({ json: { data: state.tokens } });
  });
  await page.route("**/api/v1/api-tokens/*", async route => {
    if (route.request().method() === "DELETE") {
      const id = Number(new URL(route.request().url()).pathname.split("/").pop());
      state.tokens = state.tokens.map(t => (t.id === id ? { ...t, revoked_at: "2026-10-29T06:10:00Z" } : t));
      return route.fulfill({ json: { data: { revoked: true } } });
    }
    return route.fulfill({ status: 405, json: { error: { code: "METHOD" } } });
  });
  return state;
}

test("设置页生成凭证时只显示一次明文，撤销后状态更新", async ({ page }) => {
  const state = await mockSettings(page, []);
  await page.goto("/settings");

  await expect(page.locator("#capture-token-title")).toHaveText("快速采集凭证");
  await expect(page.getByText("还没有采集凭证")).toBeVisible();

  await page.getByRole("button", { name: "生成凭证" }).click();
  await expect(page.getByText("凭证已生成，只显示这一次")).toBeVisible();
  await expect(page.locator(".token-secret code")).toHaveText("los_pat_ab12cd34ef56");

  // 列表只保留前缀，不出现明文。
  await expect(page.locator(".token-list li")).toHaveCount(1);
  await expect(page.locator(".token-list code")).toHaveText("los_pat_ab12…");
  expect(await page.locator(".token-list").innerText()).not.toContain("los_pat_ab12cd34ef56");

  await page.locator(".token-list").getByRole("button", { name: "撤销" }).click();
  await page.locator(".el-message-box__btns").getByRole("button", { name: "撤销" }).click();
  await expect.poll(() => state.tokens[0].revoked_at).not.toBeNull();
  await expect(page.locator(".token-list")).toContainText("已撤销");
});
