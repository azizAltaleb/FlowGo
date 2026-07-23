import { describe, expect, it } from "vitest";
import { resolveRuntimeConfig } from "./runtimeConfig";

describe("resolveRuntimeConfig", () => {
  it("trims canonical runtime values", () => {
    expect(
      resolveRuntimeConfig({
        apiUrl: " https://api.artificialflow.example.io ",
        oidcAuthority: "https://iam.artificialflow.example.io",
        oidcClientId: "artificialflow-frontend",
      }),
    ).toEqual({
      apiUrl: "https://api.artificialflow.example.io",
      oidcAuthority: "https://iam.artificialflow.example.io",
      oidcClientId: "artificialflow-frontend",
    });
  });

  it("returns empty strings for missing fields", () => {
    expect(resolveRuntimeConfig({ apiUrl: "/api" })).toEqual({
      apiUrl: "/api",
      oidcAuthority: "",
      oidcClientId: "",
    });
  });
});
