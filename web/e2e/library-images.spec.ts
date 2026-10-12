import { expect, test, type Page } from "@playwright/test";
const time = "2026-10-12T08:00:00Z";
const imageID = "a".repeat(32),
  imageURL = `/api/v1/library/images/${imageID}`;
const png = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAGAAAABACAIAAABqVuVZAAAAeklEQVR4nO3QMQ0AIADAMFTgFLeIwQEPBzuaTMDSMdfWpfH9IB4gQIAAAQoHCBAgQIDCAQIECBCgcIAAAQIEKBwgQIAAAQoHCBAgQIDCAQIECBCgcIAAAQIEKBwgQIAAAQoHCBAgQIDCAQIECBCgcIAAAQIEKBwgQG8d+Mep8KRt20gAAAAASUVORK5CYII=",
  "base64",
);
async function mock(page: Page) {
  const collection = {
    id: 1,
    name: "合成图片专题",
    description: "",
    category: "",
    template: "wiki",
    display_mode: "wiki",
    cover_url: "",
    version: 1,
    created_at: time,
    updated_at: time,
  };
  const saved = {
    id: 1,
    collection_id: 1,
    title: "地图资料",
    body: "原有正文",
    section_id: null,
    tags: [],
    sources: [],
    cover_url: "",
    version: 1,
    created_at: time,
    updated_at: time,
  };
  const state = {
    collection,
    uploads: 0,
    fail: false,
    delay: false,
    missing: false,
    saved,
  };
  await page.route("**/api/v1/**", async (route) => {
    const req = route.request(),
      path = new URL(req.url()).pathname;
    const reply = (data: unknown) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data }),
      });
    if (path.endsWith("/auth/me"))
      return reply({
        id: 1,
        username: "test",
        role: "admin",
        status: "active",
      });
    if (path === "/api/v1/library/images" && req.method() === "POST") {
      state.uploads++;
      expect(req.headers()["content-type"]).toContain(
        "multipart/form-data; boundary=",
      );
      if (state.delay) await new Promise((resolve) => setTimeout(resolve, 600));
      if (state.fail)
        return route.fulfill({
          status: 500,
          contentType: "application/json",
          body: JSON.stringify({
            error: { code: "TEST", message: "图片写入失败，正文未改变。" },
          }),
        });
      return reply({
        id: imageID,
        url: imageURL,
        mime: "image/png",
        size: png.length,
        width: 96,
        height: 64,
        original_name: "map.png",
        created_at: time,
      });
    }
    if (path === imageURL)
      return route.fulfill({
        status: state.missing ? 404 : 200,
        contentType: "image/png",
        body: state.missing ? "" : png,
      });
    if (path === "/api/v1/library/collections/1")
      return reply({ collection, sections: [] });
    if (path === "/api/v1/library/collections/1/pages")
      return reply({
        items: [{ ...saved, summary: saved.body.slice(0, 50) }],
        total: 1,
        page: 1,
      });
    if (path === "/api/v1/library/collections/1/pages/1") {
      if (req.method() === "PUT")
        Object.assign(saved, req.postDataJSON(), {
          version: saved.version + 1,
        });
      return reply(saved);
    }
    return reply([]);
  });
  return state;
}
async function edit(page: Page) {
  await page.goto("/explore/library/1/pages/1");
  await page.getByRole("button", { name: "编辑页面", exact: true }).click();
  await expect(
    page.getByRole("textbox", { name: "Markdown 正文", exact: true }),
  ).toHaveValue("原有正文");
}
async function pasteImage(page: Page) {
  await page
    .getByRole("textbox", { name: "Markdown 正文", exact: true })
    .evaluate((el) => {
      const canvas = document.createElement("canvas");
      canvas.width = 3;
      canvas.height = 2;
      return new Promise<void>((resolve) =>
        canvas.toBlob((blob) => {
          const data = new DataTransfer();
          data.items.add(new File([blob!], "截图.png", { type: "image/png" }));
          el.dispatchEvent(
            new ClipboardEvent("paste", {
              clipboardData: data,
              bubbles: true,
              cancelable: true,
            }),
          );
          resolve();
        }, "image/png"),
      );
    });
}

test("click upload inserts at caret, preview and saved reading show authenticated image and viewer", async ({
  page,
}) => {
  const state = await mock(page);
  await edit(page);
  const body = page.getByRole("textbox", {
    name: "Markdown 正文",
    exact: true,
  });
  await body.focus();
  await body.evaluate((el: HTMLTextAreaElement) => el.setSelectionRange(2, 2));
  await page.getByRole("button", { name: "上传图片", exact: true }).click();
  await page
    .locator(".library-image-editor input[type=file]")
    .setInputFiles({ name: "map.png", mimeType: "image/png", buffer: png });
  await expect(body).toHaveValue(`原有\n![map.png](${imageURL})\n正文`);
  await page.getByText("阅读预览", { exact: true }).click();
  const img = page.locator(".library-markdown img");
  await expect(img).toHaveAttribute("src", imageURL);
  await expect
    .poll(() => img.evaluate((i: HTMLImageElement) => i.naturalWidth))
    .toBe(96);
  await img.click();
  await expect(page.locator(".el-image-viewer__wrapper")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.locator(".el-image-viewer__wrapper")).toHaveCount(0);
  await page.getByRole("button", { name: "保存页面", exact: true }).click();
  expect(state.saved.body).toContain(imageURL);
  await page.reload();
  await expect(page.locator(".library-markdown img")).toHaveAttribute(
    "src",
    imageURL,
  );
  expect(state.uploads).toBe(1);
});

test("card directory requests one lazy async rendition and does not download its original", async ({ page }) => {
  const state = await mock(page);
  state.collection.display_mode = "list";
  state.saved.cover_url = imageURL;
  const requests: string[] = [];
  page.on("request", (request) => {
    if (new URL(request.url()).pathname === imageURL) requests.push(new URL(request.url()).search);
  });
  await page.goto("/explore/library/1");
  const cover = page.locator(".library-page-link img");
  await expect(cover).toHaveAttribute("src", `${imageURL}?variant=card-v1`);
  await expect(cover).toHaveAttribute("loading", "lazy");
  await expect(cover).toHaveAttribute("decoding", "async");
  await expect.poll(() => cover.evaluate((image) => (image as HTMLImageElement).complete && (image as HTMLImageElement).naturalWidth > 0)).toBe(true);
  expect(requests).toEqual(["?variant=card-v1"]);
});

test("paste image is uploaded once, concurrent typing is preserved, plain clipboard text retains native behavior", async ({
  page,
}) => {
  const state = await mock(page);
  await edit(page);
  state.delay = true;
  const body = page.getByRole("textbox", {
    name: "Markdown 正文",
    exact: true,
  });
  await body.focus();
  await body.evaluate((el: HTMLTextAreaElement) => el.setSelectionRange(2, 2));
  await pasteImage(page);
  await expect(
    page.getByText("图片上传中，正文仍可编辑；上传成功后插入引用。"),
  ).toBeVisible();
  await pasteImage(page);
  await body.press("End");
  await body.pressSequentially("补充");
  await expect(body).toHaveValue(`原有\n![截图.png](${imageURL})\n正文补充`);
  expect(state.uploads).toBe(1);
  const prevented = await body.evaluate((el) => {
    const data = new DataTransfer();
    data.setData("text/plain", "普通文字");
    const event = new ClipboardEvent("paste", {
      clipboardData: data,
      bubbles: true,
      cancelable: true,
    });
    el.dispatchEvent(event);
    return event.defaultPrevented;
  });
  expect(prevented).toBe(false);
  expect(state.uploads).toBe(1);
});

test("drag upload fails without changing body, retry works, invalid files are rejected before uploading", async ({
  page,
}) => {
  const state = await mock(page);
  await edit(page);
  state.fail = true;
  const body = page.getByRole("textbox", {
    name: "Markdown 正文",
    exact: true,
  });
  await body.evaluate(
    (el, bytes) => {
      const data = new DataTransfer();
      data.items.add(
        new File([new Uint8Array(bytes)], "drop.png", { type: "image/png" }),
      );
      el.dispatchEvent(
        new DragEvent("drop", {
          dataTransfer: data,
          bubbles: true,
          cancelable: true,
        }),
      );
    },
    [...png],
  );
  await expect(page.getByRole("alert")).toContainText("图片写入失败");
  await expect(body).toHaveValue("原有正文");
  state.fail = false;
  await page.getByRole("button", { name: "重试上传", exact: true }).click();
  await expect(body).toHaveValue(new RegExp(imageURL));
  await page.locator(".library-image-editor input[type=file]").setInputFiles({
    name: "unsafe.svg",
    mimeType: "image/svg+xml",
    buffer: Buffer.from("<svg/>"),
  });
  await expect(page.getByRole("alert")).toContainText("仅支持 PNG、JPEG、WebP");
  expect(state.uploads).toBe(2);
});

test("failed private image has readable fallback and can retry after login/resource recovery", async ({
  page,
}) => {
  const state = await mock(page);
  state.saved.body = `![地图](${imageURL})`;
  state.missing = true;
  await page.goto("/explore/library/1/pages/1");
  await expect(
    page.getByText("地图：无法加载，请确认登录或图片是否存在。"),
  ).toBeVisible();
  state.missing = false;
  await page.getByRole("button", { name: "重试", exact: true }).click();
  await expect
    .poll(() =>
      page
        .locator(".library-markdown img")
        .evaluate((i: HTMLImageElement) => i.naturalWidth),
    )
    .toBe(96);
});

test("native Chromium clipboard Ctrl+V supports ordinary text and a copied PNG screenshot", async ({
  page,
}) => {
  const state = await mock(page);
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  await edit(page);
  const body = page.getByRole("textbox", {
    name: "Markdown 正文",
    exact: true,
  });
  await body.focus();
  await body.press("End");
  await page.evaluate(() => navigator.clipboard.writeText("普通文字"));
  await body.press("Control+V");
  await expect(body).toHaveValue("原有正文普通文字");
  await page.evaluate(async () => {
    const canvas = document.createElement("canvas");
    canvas.width = 96;
    canvas.height = 64;
    canvas.getContext("2d")!.fillRect(0, 0, 96, 64);
    const blob = await new Promise<Blob>((resolve) =>
      canvas.toBlob((blob) => resolve(blob!), "image/png"),
    );
    await navigator.clipboard.write([new ClipboardItem({ "image/png": blob })]);
  });
  await body.press("Control+V");
  await expect(body).toHaveValue(new RegExp(imageURL));
  expect(state.uploads).toBe(1);
});

for (const width of [1440, 390]) {
  test(`local image reading and upload controls fit ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 900 });
    const state = await mock(page);
    state.saved.body = `## 地图\n\n![合成地图](${imageURL})\n\n自己的说明。`;
    await page.goto("/explore/library/1/pages/1");
    const image = page.locator(".library-markdown img");
    await expect
      .poll(() => image.evaluate((i: HTMLImageElement) => i.naturalWidth))
      .toBe(96);
    expect(
      await image.evaluate(
        (i: HTMLImageElement) => i.clientWidth <= i.naturalWidth,
      ),
    ).toBe(true);
    if (process.env.CAPTURE_VISUALS)
      await page.screenshot({
        path: info.outputPath("local-image-reading.png"),
        fullPage: true,
      });
    await page.getByRole("button", { name: "编辑页面", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "上传图片", exact: true }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth > innerWidth,
      ),
    ).toBe(false);
    if (process.env.CAPTURE_VISUALS)
      await page.screenshot({
        path: info.outputPath("image-editor.png"),
        fullPage: true,
      });
  });
}

test("cover upload previews, saves and refreshes in reader and page cards; failure keeps cover", async ({
  page,
}) => {
  const state = await mock(page);
  state.collection.display_mode = "list";
  await edit(page);
  await page.getByRole("button", { name: "上传封面", exact: true }).click();
  await page
    .getByLabel("选择封面文件")
    .setInputFiles({ name: "cover.png", mimeType: "image/png", buffer: png });
  await expect(page.getByLabel("页面封面地址")).toHaveValue(imageURL);
  await expect(page.getByAltText("页面封面预览")).toHaveAttribute(
    "src",
    imageURL,
  );
  state.fail = true;
  await page
    .getByLabel("选择封面文件")
    .setInputFiles({ name: "new.png", mimeType: "image/png", buffer: png });
  await expect(page.getByRole("alert")).toContainText("原有封面已保留");
  await expect(page.getByLabel("页面封面地址")).toHaveValue(imageURL);
  state.fail = false;
  await page.getByRole("button", { name: "保存页面", exact: true }).click();
  await expect(page.getByAltText("页面封面", { exact: true })).toHaveAttribute(
    "src",
    imageURL,
  );
  await expect(page.locator(".library-directory")).toHaveCount(0);
  await page.reload();
  await expect(page.getByAltText("页面封面", { exact: true })).toHaveAttribute(
    "src",
    imageURL,
  );
  expect(state.saved.cover_url).toBe(imageURL);
  expect(state.saved.body).toBe("原有正文");
  await page.getByRole("button", { name: "返回卡片画廊" }).click();
  await expect(page.locator(".library-page-link img")).toHaveAttribute("src", `${imageURL}?variant=card-v1`);
});

test("body image cover reuses stable ID without upload, clear preserves body, HTTPS cover remains supported", async ({
  page,
}) => {
  const state = await mock(page);
  state.saved.body = `![合成地图](${imageURL})\n![duplicate](${imageURL})\n\n\`![code](${imageURL})\``;
  await page.goto("/explore/library/1/pages/1");
  await page.getByRole("button", { name: "编辑页面", exact: true }).click();
  await page.getByRole("button", { name: "使用正文图片", exact: true }).click();
  await expect(page.locator(".library-cover-options button")).toHaveCount(1);
  await page
    .getByRole("button", { name: "使用正文图片：合成地图", exact: true })
    .click();
  await expect(page.getByLabel("页面封面地址")).toHaveValue(imageURL);
  expect(state.uploads).toBe(0);
  await page.getByRole("button", { name: "清除封面", exact: true }).click();
  await expect(page.getByLabel("页面封面地址")).toHaveValue("");
  await expect(
    page.getByRole("textbox", { name: "Markdown 正文", exact: true }),
  ).toHaveValue(state.saved.body);
  await page.route("https://example.com/cover.png", (route) =>
    route.fulfill({ contentType: "image/png", body: png }),
  );
  await page.getByLabel("页面封面地址").fill("https://example.com/cover.png");
  await expect(page.getByAltText("页面封面预览")).toHaveAttribute(
    "src",
    "https://example.com/cover.png",
  );
  await page.getByRole("button", { name: "保存页面", exact: true }).click();
  await expect(page.getByAltText("页面封面", { exact: true })).toHaveAttribute(
    "src",
    "https://example.com/cover.png",
  );
  expect(state.uploads).toBe(0);
});
