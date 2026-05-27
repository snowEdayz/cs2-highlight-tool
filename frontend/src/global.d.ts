export {};

declare global {
  interface Window {
    go?: {
      app?: {
        App?: Record<string, (...args: unknown[]) => Promise<unknown>>;
      };
    };
  }
}

