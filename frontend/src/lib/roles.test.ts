import { describe, expect, it } from "vitest";
import type { IdentityResponse } from "./api";
import {
  ARTIFICIALFLOW_ADMIN_ROLE,
  ARTIFICIALFLOW_CLIENT_ROLE,
  ARTIFICIALFLOW_MODELER_ROLE,
  canonicalizeRoles,
  hasFlexRole,
  isAdmin,
  isClientOnly,
} from "./roles";

function identity(roles: string[]): IdentityResponse {
  return {
    authenticated: true,
    principal: {
      subject: "user-1",
      roles,
    },
  };
}

describe("role helpers", () => {
  it("canonicalizes standard roles and keeps custom roles", () => {
    expect(
      canonicalizeRoles([
        " ArtificialFlow Admin ",
        ARTIFICIALFLOW_ADMIN_ROLE,
        "artificialflow modeler",
        "artificialflow client",
        "Finance Reviewer",
      ]),
    ).toEqual([
      ARTIFICIALFLOW_ADMIN_ROLE,
      ARTIFICIALFLOW_MODELER_ROLE,
      ARTIFICIALFLOW_CLIENT_ROLE,
      "Finance Reviewer",
    ]);
  });

  it("authorizes claims through canonical role helpers", () => {
    expect(isAdmin(identity(["artificialflow admin"]))).toBe(true);
    expect(isClientOnly(identity(["ARTIFICIALFLOW CLIENT"]))).toBe(true);
    expect(hasFlexRole(identity(["artificialflow modeler"]))).toBe(false);
    expect(hasFlexRole(identity(["finance reviewer"]))).toBe(true);
  });
});
