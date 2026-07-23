#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const releaseWorkflow = fs.readFileSync(path.join(root, ".github/workflows/release.yml"), "utf8");
const imageWorkflow = fs.readFileSync(path.join(root, ".github/workflows/release-images.yml"), "utf8");
const npmWorkflow = fs.readFileSync(path.join(root, ".github/workflows/release-npm.yml"), "utf8");
const failures = [];

function requireText(source, text, label) {
  if (!source.includes(text)) failures.push(label);
}

function requireAbsent(source, text, label) {
  if (source.includes(text)) failures.push(label);
}

function requireCount(source, pattern, count, label) {
  const actual = [...source.matchAll(pattern)].length;
  if (actual !== count) failures.push(`${label}; expected ${count}, got ${actual}`);
}

function requireOrder(source, first, second, label) {
  const firstIndex = source.indexOf(first);
  const secondIndex = source.indexOf(second);
  if (firstIndex < 0 || secondIndex < 0 || firstIndex >= secondIndex) failures.push(label);
}

for (const image of [
  "workflow-command",
  "workflow-runtime",
  "workflow-query",
  "sync-worker",
  "frontend",
]) {
  requireText(imageWorkflow, `- name: ${image}`, `image matrix is missing ${image}`);
}

requireText(releaseWorkflow, "  push:\n    tags:", "top-level release workflow must own the tag trigger");
requireText(
  releaseWorkflow,
  "uses: ./.github/workflows/release-images.yml",
  "orchestrator must call the reusable image workflow",
);
requireText(
  releaseWorkflow,
  "uses: ./.github/workflows/release-npm.yml",
  "orchestrator must call the reusable npm workflow",
);
requireText(releaseWorkflow, "    needs: images", "npm release job must need the complete image workflow");
requireText(
  releaseWorkflow,
  "      images_verified: true",
  "orchestrator must attest the image dependency to the npm reusable workflow",
);
requireCount(
  `${releaseWorkflow}\n${imageWorkflow}\n${npmWorkflow}`,
  /^\s{2}push:\s*$/gm,
  1,
  "only the top-level release orchestrator may have a push trigger",
);
requireAbsent(imageWorkflow, "  push:\n", "reusable image workflow must not have an independent push trigger");
requireAbsent(npmWorkflow, "  push:\n", "reusable npm workflow must not have an independent push trigger");
requireText(imageWorkflow, "  workflow_call:", "image workflow must be reusable");
requireText(npmWorkflow, "  workflow_call:", "npm workflow must be reusable");
requireOrder(
  releaseWorkflow,
  "uses: ./.github/workflows/release-images.yml",
  "uses: ./.github/workflows/release-npm.yml",
  "orchestrator must declare images before npm",
);

requireText(imageWorkflow, "release-preflight:", "image workflow must define a release preflight job");
requireText(imageWorkflow, "needs: release-preflight", "image publication must depend on release preflight");
requireText(
  imageWorkflow,
  "node scripts/validate_release_version.mjs",
  "image preflight must validate its supplied release tag",
);
requireCount(
  imageWorkflow,
  /uses:\s*docker\/build-push-action@v6/g,
  1,
  "each image must be produced by exactly one build action",
);
requireText(
  imageWorkflow,
  "candidate_tag=\"candidate-${tag}-${GITHUB_SHA}\"",
  "image workflow must use a deterministic candidate reference",
);
requireText(
  imageWorkflow,
  "tags: ${{ steps.version.outputs.candidate_reference }}",
  "the image build must publish only the candidate reference",
);
requireAbsent(
  imageWorkflow,
  "tags: ${{ steps.meta.outputs.tags }}",
  "the image build must not create final release tags before scanning",
);
requireText(
  imageWorkflow,
  "image-ref: docker.io/artificialflow/${{ matrix.image.name }}@${{ steps.artifact.outputs.digest }}",
  "Trivy must scan the resolved candidate digest",
);
requireOrder(
  imageWorkflow,
  "Login to Docker Hub",
  "Inspect deterministic candidate",
  "registry authentication must precede candidate existence checks",
);
requireOrder(
  imageWorkflow,
  "Scan candidate digest",
  "Promote scanned digest to immutable final references",
  "immutable final tags must be created only after Trivy succeeds",
);
requireText(
  imageWorkflow,
  "operational failure inspecting ${reference}",
  "manifest inspection must report operational failures",
);
requireText(
  imageWorkflow,
  '"${output}" == *"manifest unknown"*',
  "manifest inspection must identify a missing manifest explicitly",
);
requireText(
  imageWorkflow,
  'if [[ "${INSPECTED_DIGEST}" != "${candidate_digest}" ]]',
  "existing immutable tags must match the candidate digest on retry",
);
requireText(
  imageWorkflow,
  'if [[ "${final_tag}" != "${moving_minor_tag}" ]]',
  "immutable tag protection must still allow the moving major.minor alias to advance",
);
requireText(
  imageWorkflow,
  "docker buildx imagetools create --tag \"${reference}\" \"${source}\"",
  "the scanned candidate digest must be promoted without rebuilding",
);
requireText(
  imageWorkflow,
  'test "${legacy_digest}" = "${canonical_digest}"',
  "published canonical and legacy aliases must assert manifest digest equality",
);
requireText(
  imageWorkflow,
  'cosign sign --yes "${canonical}"',
  "canonical image digest must be signed",
);
requireText(
  imageWorkflow,
  'cosign sign --yes "${legacy}"',
  "legacy image digest must be signed",
);
requireText(
  imageWorkflow,
  'cosign verify --certificate-identity "${identity}" --certificate-oidc-issuer "${issuer}" "${canonical}"',
  "canonical image signature must be verified",
);
requireText(
  imageWorkflow,
  'cosign verify --certificate-identity "${identity}" --certificate-oidc-issuer "${issuer}" "${legacy}"',
  "legacy image signature must be verified",
);
requireText(
  imageWorkflow,
  "verify-release-set:",
  "image workflow must verify the complete canonical and legacy set",
);
requireText(
  imageWorkflow,
  "needs: docker-images",
  "complete-set verification must wait for every image matrix job",
);

const wrapperName = ["@", "flow", "go/nodejs-sdk"].join("");
requireText(npmWorkflow, "release-preflight:", "npm workflow must define a release preflight job");
requireText(
  npmWorkflow,
  "if: ${{ inputs.publish && !inputs.images_verified }}",
  "direct npm publication must fail without the orchestrated image dependency",
);
requireText(
  npmWorkflow,
  "node scripts/validate_release_version.mjs",
  "npm preflight must validate release version consistency",
);
requireText(npmWorkflow, "needs: publish-nodejs-sdk", "compatibility wrapper must depend on canonical npm job");
requireText(
  npmWorkflow,
  "if: ${{ needs.publish-nodejs-sdk.outputs.publish == 'true' }}",
  "compatibility wrapper publication must use the canonical job decision",
);
requireText(
  npmWorkflow,
  "node scripts/validate_nodejs_sdk_package.mjs",
  "compatibility wrapper must validate exact-version dependency metadata",
);
requireText(
  npmWorkflow,
  "Refuse existing exact npm versions",
  "npm publication must reject exact-version collisions",
);
requireText(
  npmWorkflow,
  `for package in @artificialflow/nodejs-sdk ${wrapperName}`,
  "npm collision guard must cover canonical and legacy packages",
);
requireText(npmWorkflow, wrapperName, "npm workflow must name the compatibility wrapper");
requireAbsent(
  npmWorkflow,
  "imagetools inspect",
  "npm workflow must use job dependencies instead of registry polling",
);
requireAbsent(npmWorkflow, "sleep 30", "npm workflow must not poll the registry");
requireOrder(
  npmWorkflow,
  "Refuse existing exact npm versions",
  "npm publish --access public --provenance --tag \"${{ steps.release.outputs.dist_tag }}\"",
  "exact npm collision checks must run before canonical publication",
);
requireOrder(
  npmWorkflow,
  "npm publish --access public --provenance --tag \"${{ steps.release.outputs.dist_tag }}\"",
  "npm publish --access public --provenance --tag \"${{ needs.publish-nodejs-sdk.outputs.dist_tag }}\"",
  "canonical npm package must publish before its compatibility wrapper",
);

if (failures.length > 0) {
  console.error("Release workflow assertions failed:");
  failures.forEach((failure) => console.error(`- ${failure}`));
  process.exit(1);
}

console.log("Release orchestration, image promotion, and alias assertions passed");
