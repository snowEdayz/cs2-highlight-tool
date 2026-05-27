import type { ClipParameterOverrides } from "@/shared/types";

export function normalizeSpecMode(_mode: unknown): 1 {
  return 1;
}

export function normalizeClipOverrides(
  input: Partial<ClipParameterOverrides> | undefined,
): ClipParameterOverrides | undefined {
  if (!input) return undefined;
  const next: ClipParameterOverrides = {};
  if (typeof input.killer_pre_seconds === "number" && Number.isFinite(input.killer_pre_seconds)) {
    next.killer_pre_seconds = input.killer_pre_seconds;
  }
  if (typeof input.killer_post_seconds === "number" && Number.isFinite(input.killer_post_seconds)) {
    next.killer_post_seconds = input.killer_post_seconds;
  }
  if (typeof input.victim_pre_seconds === "number" && Number.isFinite(input.victim_pre_seconds)) {
    next.victim_pre_seconds = input.victim_pre_seconds;
  }
  if (typeof input.victim_post_seconds === "number" && Number.isFinite(input.victim_post_seconds)) {
    next.victim_post_seconds = input.victim_post_seconds;
  }
  if (typeof input.enable_voice === "boolean") {
    next.enable_voice = input.enable_voice;
  }
  if (typeof input.enable_spec_show_xray_zero === "boolean") {
    next.enable_spec_show_xray_zero = input.enable_spec_show_xray_zero;
  }
  return Object.keys(next).length > 0 ? next : undefined;
}

export function basename(p: string): string {
  const m = /[^\\/]+$/.exec(p);
  return m ? m[0] : p;
}
