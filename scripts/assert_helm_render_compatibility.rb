#!/usr/bin/env ruby
# frozen_string_literal: true

require "yaml"

abort "usage: #{$PROGRAM_NAME} <fresh-external> <fresh-internal> <legacy-external> <legacy-internal>" unless ARGV.length == 4

def documents(path)
  YAML.load_stream(File.read(path, encoding: "UTF-8")).compact.select { |doc| doc.is_a?(Hash) && doc["kind"] }
end

def identity_set(docs)
  docs.map { |doc| "#{doc.fetch("kind")}/#{doc.dig("metadata", "name")}" }.sort
end

def assert_selectors!(docs, expected_name, expected_instance, label)
  docs.each do |doc|
    selectors =
      case doc["kind"]
      when "Service"
        selector = doc.dig("spec", "selector")
        selector ? [selector] : []
      when "Deployment", "StatefulSet"
        [
          doc.dig("spec", "selector", "matchLabels"),
          doc.dig("spec", "template", "metadata", "labels"),
        ].compact
      else
        []
      end

    selectors.each do |selector|
      actual = [
        selector["app.kubernetes.io/name"],
        selector["app.kubernetes.io/instance"],
      ]
      expected = [expected_name, expected_instance]
      abort "#{label}: #{doc["kind"]}/#{doc.dig("metadata", "name")} selector #{actual.inspect}, expected #{expected.inspect}" unless actual == expected
    end
  end
end

def assert_keep_policy!(docs, label)
  docs.select { |doc| %w[PersistentVolumeClaim Secret].include?(doc["kind"]) }.each do |doc|
    policy = doc.dig("metadata", "annotations", "helm.sh/resource-policy")
    abort "#{label}: #{doc["kind"]}/#{doc.dig("metadata", "name")} is missing helm.sh/resource-policy=keep" unless policy == "keep"
  end

  docs.select { |doc| doc["kind"] == "StatefulSet" }.each do |doc|
    retention = doc.dig("spec", "persistentVolumeClaimRetentionPolicy")
    expected = { "whenDeleted" => "Retain", "whenScaled" => "Retain" }
    abort "#{label}: #{doc.dig("metadata", "name")} must retain StatefulSet claims" unless retention == expected
  end
end

def assert_canonical_images!(docs, label)
  application_containers = %w[workflow-command workflow-runtime workflow-query sync-worker frontend]
  docs.each do |doc|
    containers = doc.dig("spec", "template", "spec", "containers") || []
    containers.each do |container|
      next unless application_containers.include?(container["name"])

      image = container["image"].to_s
      abort "#{label}: #{container["name"]} uses non-canonical image #{image}" unless image.start_with?("artificialflow/")
    end
  end
end

def resource!(docs, kind, name, label)
  docs.find { |doc| doc["kind"] == kind && doc.dig("metadata", "name") == name } ||
    abort("#{label}: missing #{kind}/#{name}")
end

def container_env!(docs, workload_kind, workload_name, container_name, label)
  workload = resource!(docs, workload_kind, workload_name, label)
  container = (workload.dig("spec", "template", "spec", "containers") || [])
    .find { |candidate| candidate["name"] == container_name }
  abort "#{label}: missing container #{container_name} in #{workload_kind}/#{workload_name}" unless container

  (container["env"] || []).to_h { |entry| [entry["name"], entry["value"]] }
end

def assert_ingress_identity!(docs, name, host, tls_secret, label)
  ingress = resource!(docs, "Ingress", name, label)
  actual_hosts = (ingress.dig("spec", "rules") || []).map { |rule| rule["host"] }.uniq
  actual_tls = ingress.dig("spec", "tls", 0)
  abort "#{label}: Ingress/#{name} hosts #{actual_hosts.inspect}, expected #{host.inspect}" unless actual_hosts == [host]
  abort "#{label}: Ingress/#{name} TLS host changed" unless actual_tls&.fetch("hosts", nil) == [host]
  abort "#{label}: Ingress/#{name} TLS secret changed" unless actual_tls["secretName"] == tls_secret
end

fresh_external = documents(ARGV[0])
fresh_internal = documents(ARGV[1])
legacy_external = documents(ARGV[2])
legacy_internal = documents(ARGV[3])

[["fresh external", fresh_external], ["fresh internal", fresh_internal]].each do |label, docs|
  names = docs.map { |doc| doc.dig("metadata", "name") }.compact
  unexpected = names.grep(/\Aflowgo(?:-|\z)/)
  abort "#{label}: fresh render contains legacy resource names: #{unexpected.join(", ")}" unless unexpected.empty?
  assert_selectors!(docs, "artificialflow", "artificialflow", label)
  assert_keep_policy!(docs, label)
  assert_canonical_images!(docs, label)
end

expected_legacy_external = %w[
  ConfigMap/flowgo-env
  ConfigMap/flowgo-gateway-nginx
  Deployment/flowgo-command
  Deployment/flowgo-frontend
  Deployment/flowgo-gateway
  Deployment/flowgo-query
  Deployment/flowgo-runtime
  Deployment/flowgo-sync-worker
  Ingress/flowgo
  Secret/flowgo-env
  Service/flowgo-command
  Service/flowgo-frontend
  Service/flowgo-gateway
  Service/flowgo-query
  Service/flowgo-sync-worker
  ServiceAccount/flowgo
].sort

expected_legacy_internal = (expected_legacy_external + %w[
  ConfigMap/flowgo-zitadel-bootstrap-script
  Deployment/flowgo-zitadel-api
  Deployment/flowgo-zitadel-login
  Ingress/flowgo-zitadel
  PersistentVolumeClaim/flowgo-flowgo-auth
  PersistentVolumeClaim/flowgo-flowgo-bootstrap
  PersistentVolumeClaim/flowgo-zitadel-bootstrap
  Secret/flowgo-zitadel
  Service/flowgo-zitadel-api
  Service/flowgo-zitadel-login
  Service/flowgo-zitadel-postgresql
  StatefulSet/flowgo-zitadel-postgresql
]).sort

{
  "legacy external" => [legacy_external, expected_legacy_external],
  "legacy internal" => [legacy_internal, expected_legacy_internal],
}.each do |label, (docs, expected)|
  actual = identity_set(docs)
  unless actual == expected
    abort "#{label}: resource identities differ\nmissing: #{(expected - actual).join(", ")}\nunexpected: #{(actual - expected).join(", ")}"
  end
  assert_selectors!(docs, "flowgo", "flowgo", label)
  assert_keep_policy!(docs, label)
  assert_canonical_images!(docs, label)
end

legacy_external_config = resource!(legacy_external, "ConfigMap", "flowgo-env", "legacy external").fetch("data")
{
  "AUTH_ISSUER_INTERNAL_URL" => "https://login.example.com",
  "AUTH_ISSUER_PUBLIC_URL" => "https://login.example.com",
  "FRONTEND_AUTH_OIDC_AUTHORITY" => "https://login.example.com",
  "ALLOWED_ORIGINS" => "https://flowgo.example.com",
}.each do |key, expected|
  actual = legacy_external_config[key]
  abort "legacy external: #{key}=#{actual.inspect}, expected #{expected.inspect}" unless actual == expected
end
assert_ingress_identity!(legacy_external, "flowgo", "flowgo.example.com", "flowgo-tls", "legacy external")

legacy_internal_config = resource!(legacy_internal, "ConfigMap", "flowgo-env", "legacy internal").fetch("data")
{
  "AUTH_ISSUER_INTERNAL_URL" => "http://flowgo-zitadel-api:8080",
  "AUTH_ISSUER_PUBLIC_URL" => "https://iam.flowgo.example.com",
  "FRONTEND_AUTH_OIDC_AUTHORITY" => "https://iam.flowgo.example.com",
  "ARTIFICIALFLOW_OIDC_AUTHORITY" => "https://iam.flowgo.example.com",
  "FRONTEND_AUTH_OIDC_CLIENT_ID_FILE" => "/flowgo/bootstrap/flowgo-frontend-client-id",
  "ARTIFICIALFLOW_OIDC_CLIENT_ID_FILE" => "/flowgo/bootstrap/flowgo-frontend-client-id",
  "ALLOWED_ORIGINS" => "https://flowgo.example.com",
}.each do |key, expected|
  actual = legacy_internal_config[key]
  abort "legacy internal: #{key}=#{actual.inspect}, expected #{expected.inspect}" unless actual == expected
end

assert_ingress_identity!(legacy_internal, "flowgo", "flowgo.example.com", "flowgo-tls", "legacy internal")
assert_ingress_identity!(
  legacy_internal,
  "flowgo-zitadel",
  "iam.flowgo.example.com",
  "flowgo-iam-tls",
  "legacy internal",
)

zitadel_api_env = container_env!(
  legacy_internal,
  "Deployment",
  "flowgo-zitadel-api",
  "zitadel-api",
  "legacy internal",
)
{
  "ZITADEL_EXTERNALDOMAIN" => "iam.flowgo.example.com",
  "ZITADEL_EXTERNALPORT" => "443",
  "ZITADEL_EXTERNALSECURE" => "true",
  "ZITADEL_DEFAULTINSTANCE_FEATURES_LOGINV2_BASEURI" => "https://iam.flowgo.example.com/ui/v2/login/",
}.each do |key, expected|
  actual = zitadel_api_env[key]
  abort "legacy internal: #{key}=#{actual.inspect}, expected #{expected.inspect}" unless actual == expected
end

zitadel_login_env = container_env!(
  legacy_internal,
  "Deployment",
  "flowgo-zitadel-login",
  "zitadel-login",
  "legacy internal",
)
{
  "ZITADEL_PUBLIC_URL" => "https://iam.flowgo.example.com",
  "ARTIFICIALFLOW_FRONTEND_URL" => "https://flowgo.example.com",
  "ARTIFICIALFLOW_FRONTEND_CLIENT_ID_FILE" => "/flowgo/bootstrap/flowgo-frontend-client-id",
}.each do |key, expected|
  actual = zitadel_login_env[key]
  abort "legacy internal: #{key}=#{actual.inspect}, expected #{expected.inspect}" unless actual == expected
end

puts "helm fresh and legacy resource identity assertions passed"
