export const DEFAULT_REGISTRY_URL =
  "https://raw.githubusercontent.com/mvanhorn/printing-press-library/main/registry.json";

export interface MCPBlock {
  binary: string;
  transport: string;
  tool_count: number;
  public_tool_count?: number;
  auth_type: string;
  env_vars: string[];
  mcp_ready?: string;
}

export interface RegistryEntry {
  name: string;
  category: string;
  api: string;
  description: string;
  path: string;
  mcp?: MCPBlock;
}

export interface Registry {
  schema_version: number;
  entries: RegistryEntry[];
}

export function parseRegistry(value: unknown): Registry {
  if (!isRecord(value)) {
    throw new Error("registry payload must be an object");
  }
  if (value.schema_version !== 1) {
    throw new Error(`unsupported registry schema_version: ${String(value.schema_version)}`);
  }
  if (!Array.isArray(value.entries)) {
    throw new Error("registry entries must be an array");
  }

  return {
    schema_version: 1,
    entries: value.entries.map(parseRegistryEntry),
  };
}

export async function fetchRegistry(
  url = DEFAULT_REGISTRY_URL,
  fetchImpl: typeof fetch = fetch,
): Promise<Registry> {
  const response = await fetchImpl(url);
  if (!response.ok) {
    throw new Error(`failed to fetch registry: HTTP ${response.status}`);
  }
  return parseRegistry(await response.json());
}

export function lookupByName(registry: Registry, name: string): RegistryEntry | null {
  const normalized = normalizeName(name);
  return (
    registry.entries.find((entry) => {
      const entryName = normalizeName(entry.name);
      return entryName === normalized || normalizeName(entry.api) === normalized;
    }) ?? null
  );
}

export function cliSkillName(entry: RegistryEntry): string {
  return `pp-${entry.name.replace(/-pp-cli$/, "")}`;
}

function parseRegistryEntry(value: unknown): RegistryEntry {
  if (!isRecord(value)) {
    throw new Error("registry entry must be an object");
  }

  const entry = {
    name: requiredString(value, "name"),
    category: requiredString(value, "category"),
    api: requiredString(value, "api"),
    description: requiredString(value, "description"),
    path: requiredString(value, "path"),
  };

  return isRecord(value.mcp)
    ? {
        ...entry,
        mcp: {
          binary: requiredString(value.mcp, "binary"),
          transport: requiredString(value.mcp, "transport"),
          tool_count: requiredNumber(value.mcp, "tool_count"),
          public_tool_count:
            typeof value.mcp.public_tool_count === "number" ? value.mcp.public_tool_count : undefined,
          auth_type: requiredString(value.mcp, "auth_type"),
          env_vars: Array.isArray(value.mcp.env_vars) ? value.mcp.env_vars.map(String) : [],
          mcp_ready: typeof value.mcp.mcp_ready === "string" ? value.mcp.mcp_ready : undefined,
        },
      }
    : entry;
}

function normalizeName(value: string): string {
  return value.toLowerCase().replace(/^pp-/, "").replace(/-pp-cli$/, "").replace(/[^a-z0-9]+/g, "-");
}

function requiredString(value: Record<string, unknown>, key: string): string {
  if (typeof value[key] !== "string" || value[key].trim() === "") {
    throw new Error(`registry entry missing string field: ${key}`);
  }
  return value[key];
}

function requiredNumber(value: Record<string, unknown>, key: string): number {
  if (typeof value[key] !== "number") {
    throw new Error(`registry entry missing number field: ${key}`);
  }
  return value[key];
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
