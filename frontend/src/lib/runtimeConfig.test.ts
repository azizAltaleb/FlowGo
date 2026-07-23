import { describe, expect, it } from "vitest";
import { resolveRuntimeConfig } from "./runtimeConfig";

describe("resolveRuntimeConfig", () => {
  it("prefers canonical runtime values", () => {
    expect(
      resolveRuntimeConfig(
        {
          apiUrl: " https://api.artificialflow.example.io ",
          oidcAuthority: "https://iam.artificialflow.example.io",
          oidcClientId: "artificialflow-frontend",
        },
        {
          apiUrl: "https://legacy.example/api",
          oidcAuthority: "https://legacy.example/iam",
          oidcClientId: "legacy-client",
        },
      ),
    ).toEqual({
      apiUrl: "https://api.artificialflow.example.io",
      oidcAuthority: "https://iam.artificialflow.example.io",
      oidcClientId: "artificialflow-frontend",
    });
  });

  it("falls back to legacy values per field", () => {
    expect(
      resolveRuntimeConfig(
        { apiUrl: "/api" },
        {
          apiUrl: "/legacy-api",
          oidcAuthority: "https://legacy.example/iam",
          oidcClientId: "legacy-client",
        },
      ),
    ).toEqual({
      apiUrl: "/api",
      oidcAuthority: "https://legacy.example/iam",
      oidcClientId: "legacy-client",
    });
  });
});
