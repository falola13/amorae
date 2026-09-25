import { describe, expect, it } from "vitest";

import { ApiError } from "@/lib/api/errors";
import { isAlone } from "./derive";

const apiError = (code: string) => new ApiError(409, code, "");

describe("isAlone", () => {
  it("is true with no partner, even though nothing failed", () => {
    expect(isAlone(false, undefined)).toBe(true);
  });

  it("is true when the API says so", () => {
    expect(isAlone(false, apiError("waiting_for_partner"))).toBe(true);
    expect(isAlone(true, apiError("waiting_for_partner"))).toBe(true);
  });

  it("is false once a partner is there and nothing is wrong", () => {
    expect(isAlone(true, undefined)).toBe(false);
  });

  it("does not mistake an unrelated failure for being alone", () => {
    expect(isAlone(true, apiError("internal_error"))).toBe(false);
    expect(isAlone(true, new Error("network"))).toBe(false);
  });
});
