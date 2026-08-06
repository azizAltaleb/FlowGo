import { describe, expect, it } from "vitest";
import type { IdentityConfigResponse, IdentityResponse } from "./api";
import {
  canAccessIdentityConsole,
  identityConsoleDeniedMessage,
  isDashboardPathAllowed,
  isIdentityConsolePath,
} from "./dashboardAccess";

function identity(roles: string[]): IdentityResponse {
  return {
    authenticated: true,
    principal: { subject: "user-1", roles },
  };
}

function config(mode: string): IdentityConfigResponse {
  return {
    deployment_mode: mode,
    configuration_source: "test",
    provider_name: "test",
    auth_enabled: true,
    frontend_auth_enabled: true,
    frontend_oidc_authority: "http://localhost:9180",
    frontend_oidc_client_id: "console",
    token_validation_mode: "jwt",
    internal_issuer_url: "",
    external_issuer_url: "",
    client_id: "",
    introspection_url: "",
    introspection_client_id: "",
    introspection_auth_method: "",
    enforce_audience: false,
    allow_insecure_issuer: false,
    claim_subject_path: "",
    claim_roles_path: "",
    claim_scopes_path: "",
    claim_tenant_path: "",
    claim_email_path: "",
    claim_name_path: "",
    standard_roles: [],
  };
}

describe("dashboardAccess", () => {
  it("recognizes identity console paths", () => {
    expect(isIdentityConsolePath("/identity")).toBe(true);
    expect(isIdentityConsolePath("/sdk-clients")).toBe(true);
    expect(isIdentityConsolePath("/processes")).toBe(false);
  });

  it("allows zitadel admins on /identity and /sdk-clients", () => {
    const admin = identity(["artificialflow admin"]);
    const zitadel = config("zitadel");
    expect(canAccessIdentityConsole(admin, zitadel)).toBe(true);
    expect(isDashboardPathAllowed({ path: "/identity", identity: admin, config: zitadel })).toBe(true);
    expect(isDashboardPathAllowed({ path: "/sdk-clients", identity: admin, config: zitadel })).toBe(true);
  });

  it("denies identity console when admin detection fails even if other roles exist", () => {
    const modeler = identity(["artificialflow modeler"]);
    const zitadel = config("zitadel");
    expect(isDashboardPathAllowed({ path: "/identity", identity: modeler, config: zitadel })).toBe(false);
    expect(identityConsoleDeniedMessage(modeler, zitadel)).toMatch(/artificialflow admin/i);
  });

  it("denies identity console outside bundled zitadel mode", () => {
    const admin = identity(["artificialflow admin"]);
    const external = config("external");
    expect(isDashboardPathAllowed({ path: "/identity", identity: admin, config: external })).toBe(false);
    expect(identityConsoleDeniedMessage(admin, external)).toMatch(/bundled ZITADEL/i);
  });

  it("hides incidents paths when the frontend incidents UI is disabled", () => {
    const admin = identity(["artificialflow admin"]);
    const zitadel = config("zitadel");
    expect(isDashboardPathAllowed({ path: "/incidents", identity: admin, config: zitadel })).toBe(false);
    expect(
      isDashboardPathAllowed({
        path: "/incidents",
        identity: admin,
        config: zitadel,
        showIncidentsUi: true,
      }),
    ).toBe(true);
  });
});
