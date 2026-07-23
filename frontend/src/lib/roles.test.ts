import { describe, expect, it } from "vitest";
import type { IdentityResponse } from "./api";
import {
  ARTIFICIALFLOW_ADMIN_ROLE,
  ARTIFICIALFLOW_CLIENT_ROLE,
  ARTIFICIALFLOW_MODELER_ROLE,
  FLOWGO_ADMIN_ROLE,
  FLOWGO_CLIENT_ROLE,
  FLOWGO_MODELER_ROLE,
  STATIC_ARTIFICIALFLOW_ROLES,
  STATIC_FLOWGO_ROLES,
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

describe("role compatibility", () => {
  it("keeps deprecated aliases canonical for public callers", () => {
    expect(FLOWGO_ADMIN_ROLE).toBe(ARTIFICIALFLOW_ADMIN_ROLE);
    expect(FLOWGO_MODELER_ROLE).toBe(ARTIFICIALFLOW_MODELER_ROLE);
    expect(FLOWGO_CLIENT_ROLE).toBe(ARTIFICIALFLOW_CLIENT_ROLE);
    expect(STATIC_FLOWGO_ROLES).toBe(STATIC_ARTIFICIALFLOW_ROLES);
  });

  it("canonicalizes legacy standard roles and keeps custom roles", () => {
    expect(
      canonicalizeRoles([
        " FlowGo Admin ",
        ARTIFICIALFLOW_ADMIN_ROLE,
        "flowgo modeler",
        "flowgo client",
        "Finance Reviewer",
      ]),
    ).toEqual([
      ARTIFICIALFLOW_ADMIN_ROLE,
      ARTIFICIALFLOW_MODELER_ROLE,
      ARTIFICIALFLOW_CLIENT_ROLE,
      "Finance Reviewer",
    ]);
  });

  it("authorizes legacy claims through canonical role helpers", () => {
    expect(isAdmin(identity(["flowgo admin"]))).toBe(true);
    expect(isClientOnly(identity(["FLOWGO CLIENT"]))).toBe(true);
    expect(hasFlexRole(identity(["flowgo modeler"]))).toBe(false);
    expect(hasFlexRole(identity(["finance reviewer"]))).toBe(true);
  });
});
