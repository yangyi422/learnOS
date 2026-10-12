import { describe, expect, it } from "vitest";
import { libraryCardImageURL, localLibraryImage } from "./libraryMarkdown";

describe("library card images", () => {
  it("uses a fixed thumbnail only for canonical private image references", () => {
    const original = `/api/v1/library/images/${"a".repeat(32)}`;
    expect(libraryCardImageURL(original)).toBe(`${original}?variant=card-v1`);
    expect(localLibraryImage(original)).toBe(true);
    expect(localLibraryImage(libraryCardImageURL(original))).toBe(false);
    for (const url of ["", "https://example.com/photo.png", "/api/v1/library/images/invalid", `${original}?token=secret`]) {
      expect(libraryCardImageURL(url)).toBe(url);
    }
  });
});
