import { computed, ref } from "vue";
import { t } from "@/shared/i18n";

const path = ref("");
const errorMessage = ref("");
const validated = ref(false);
const submitting = ref(false);

async function callBackend<T>(method: string, ...args: unknown[]): Promise<T> {
  const api = window.go?.app?.App as
    | Record<string, (...a: unknown[]) => Promise<unknown>>
    | undefined;
  const fn = api?.[method];
  if (!fn) throw new Error(`Wails API not loaded: ${method}`);
  return fn(...args) as Promise<T>;
}

export function useWorkspaceInit() {
  const canSubmit = computed(
    () => !submitting.value && path.value.trim().length > 0 && validated.value && !errorMessage.value,
  );

  function reset(): void {
    path.value = "";
    errorMessage.value = "";
    validated.value = false;
    submitting.value = false;
  }

  async function validate(): Promise<void> {
    const trimmed = path.value.trim();
    if (!trimmed) {
      errorMessage.value = "";
      validated.value = false;
      return;
    }
    try {
      // Backend signature: ValidateWorkspaceDir(path) (ok bool, errorMessage string).
      // Wails serializes multi-return as an array; we still defensively handle an
      // object shape in case the binding generator changes in the future.
      const result = await callBackend<unknown>("ValidateWorkspaceDir", trimmed);
      let ok = false;
      let errMsg = "";
      if (Array.isArray(result)) {
        ok = Boolean(result[0]);
        const second = result[1];
        errMsg = typeof second === "string" ? second : "";
      } else if (result && typeof result === "object") {
        const obj = result as Record<string, unknown>;
        ok = Boolean(obj.ok);
        errMsg = typeof obj.errorMessage === "string" ? obj.errorMessage : "";
      }
      validated.value = ok;
      errorMessage.value = ok ? "" : errMsg || t("workspace.validate.generic");
    } catch (err) {
      validated.value = false;
      errorMessage.value = String(err);
    }
  }

  async function pick(): Promise<void> {
    try {
      const selected = (await callBackend<string>("PickWorkspaceDir")) || "";
      const trimmed = String(selected).trim();
      if (!trimmed) return;
      path.value = trimmed;
      await validate();
    } catch (err) {
      errorMessage.value = String(err);
      validated.value = false;
    }
  }

  async function confirm(): Promise<boolean> {
    if (!canSubmit.value) return false;
    submitting.value = true;
    try {
      await callBackend<void>("SetWorkspaceDir", path.value.trim());
      // Reset on success so a later reset → modal reopen starts clean.
      reset();
      return true;
    } catch (err) {
      errorMessage.value = String(err);
      validated.value = false;
      return false;
    } finally {
      submitting.value = false;
    }
  }

  async function exitApp(): Promise<void> {
    try {
      await callBackend<void>("ExitApp");
    } catch {
      // Ignore: process is supposed to terminate.
    }
  }

  return {
    t,
    path,
    errorMessage,
    submitting,
    canSubmit,
    pick,
    validate,
    confirm,
    exitApp,
    reset,
  };
}
