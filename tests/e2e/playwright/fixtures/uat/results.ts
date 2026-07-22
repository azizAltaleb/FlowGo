import fs from "fs";
import path from "path";
import type { TestInfo } from "@playwright/test";
import type { UatCase } from "./cases";
import type { UatDeployment } from "./auth";

export interface UatCaseResult {
  id: string;
  title: string;
  category: string;
  deployment: UatDeployment;
  expected: string;
  status: "pass" | "fail";
  elements: string[];
  functions: string[];
  video?: string;
  error?: string;
  duration_ms: number;
}

export function runDir(): string {
  const dir = process.env.UAT_REPORT_DIR || path.resolve(__dirname, "../../../../../reports/uat-video-suite/local");
  fs.mkdirSync(dir, { recursive: true });
  fs.mkdirSync(path.join(dir, "videos"), { recursive: true });
  fs.mkdirSync(path.join(dir, "results"), { recursive: true });
  return dir;
}

export async function recordCaseResult(testInfo: TestInfo, deployment: UatDeployment, uatCase: UatCase): Promise<void> {
  await writeCaseResult(testInfo, deployment, uatCase);
}

export async function writeCaseResult(
  testInfo: TestInfo,
  deployment: UatDeployment,
  uatCase: UatCase,
  rawVideoOverride?: string,
  errorOverride?: Error,
): Promise<void> {
  const dir = runDir();
  const safeName = `${deployment}/${uatCase.id}-${slug(uatCase.title)}.webm`;
  const finalVideo = path.join(dir, "videos", safeName);
  fs.mkdirSync(path.dirname(finalVideo), { recursive: true });

  const rawVideo = rawVideoOverride || testInfo.attachments.find((attachment) => attachment.name === "video")?.path;
  if (rawVideo && fs.existsSync(rawVideo)) {
    fs.copyFileSync(rawVideo, finalVideo);
  }

  const result: UatCaseResult = {
    id: uatCase.id,
    title: uatCase.title,
    category: uatCase.category,
    deployment,
    expected: uatCase.expected,
    status: errorOverride ? "fail" : "pass",
    elements: uatCase.elements,
    functions: uatCase.functions,
    video: fs.existsSync(finalVideo) ? path.relative(dir, finalVideo) : undefined,
    error: errorOverride?.message || testInfo.error?.message,
    duration_ms: testInfo.duration,
  };

  fs.writeFileSync(
    path.join(dir, "results", `${deployment}-${uatCase.id}.json`),
    JSON.stringify(result, null, 2) + "\n",
  );
  writeManifest(dir);
}

export function writeManifest(dir = runDir()): void {
  const resultsDir = path.join(dir, "results");
  const results = fs.existsSync(resultsDir)
    ? fs.readdirSync(resultsDir)
        .filter((file) => file.endsWith(".json"))
        .map((file) => JSON.parse(fs.readFileSync(path.join(resultsDir, file), "utf8")) as UatCaseResult)
        .sort((a, b) => `${a.deployment}-${a.id}`.localeCompare(`${b.deployment}-${b.id}`))
    : [];

  const counts = results.reduce<Record<string, number>>((acc, item) => {
    const key = item.status === "pass" ? item.expected : "fail";
    acc[key] = (acc[key] || 0) + 1;
    return acc;
  }, {});

  const elementCoverage = new Map<string, string[]>();
  const functionCoverage = new Map<string, string[]>();
  for (const result of results) {
    for (const element of result.elements) {
      const cases = elementCoverage.get(element) || [];
      cases.push(`${result.deployment}:${result.id}`);
      elementCoverage.set(element, cases);
    }
    for (const fn of result.functions) {
      const cases = functionCoverage.get(fn) || [];
      cases.push(`${result.deployment}:${result.id}`);
      functionCoverage.set(fn, cases);
    }
  }

  const manifest = {
    generated_at: new Date().toISOString(),
    report_dir: dir,
    total: results.length,
    counts,
    results,
    element_coverage: Object.fromEntries([...elementCoverage.entries()].sort()),
    functionality_coverage: Object.fromEntries([...functionCoverage.entries()].sort()),
  };
  fs.writeFileSync(path.join(dir, "manifest.json"), JSON.stringify(manifest, null, 2) + "\n");
  fs.writeFileSync(path.join(dir, "summary.md"), summaryMarkdown(manifest));
}

function summaryMarkdown(manifest: {
  generated_at: string;
  total: number;
  counts: Record<string, number>;
  results: UatCaseResult[];
  element_coverage: Record<string, string[]>;
  functionality_coverage: Record<string, string[]>;
}): string {
  const lines = [
    "# FlowGo UAT Video Suite",
    "",
    `- Generated at: ${manifest.generated_at}`,
    `- Total recorded cases: ${manifest.total}`,
    `- Pass cases: ${manifest.counts.pass || 0}`,
    `- Expected rejection cases: ${manifest.counts["expected-rejection"] || 0}`,
    `- Expected skip/gap cases: ${manifest.counts["expected-skip"] || 0}`,
    `- Failed cases: ${manifest.counts.fail || 0}`,
    "",
    "## Case Results",
    "",
    "| Deployment | Case | Expected | Status | Video |",
    "| :--- | :--- | :--- | :--- | :--- |",
  ];
  for (const result of manifest.results) {
    lines.push(`| ${result.deployment} | \`${result.id}\` ${result.title} | ${result.expected} | ${result.status.toUpperCase()} | ${result.video ? `\`${result.video}\`` : ""} |`);
  }
  lines.push("", "## BPMN Element Coverage", "", "| Element | Cases |", "| :--- | :--- |");
  for (const [element, cases] of Object.entries(manifest.element_coverage)) {
    lines.push(`| \`${element}\` | ${cases.map((item) => `\`${item}\``).join(", ")} |`);
  }
  lines.push("", "## Functionality Coverage", "", "| Functionality | Cases |", "| :--- | :--- |");
  for (const [fn, cases] of Object.entries(manifest.functionality_coverage)) {
    lines.push(`| ${fn} | ${cases.map((item) => `\`${item}\``).join(", ")} |`);
  }
  return lines.join("\n") + "\n";
}

function slug(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "").slice(0, 72);
}
