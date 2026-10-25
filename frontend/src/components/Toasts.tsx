import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from "react";
import styles from "./Toasts.module.css";

export type ToastKind = "error" | "success" | "info";
type Toast = { id: number; kind: ToastKind; message: string };

const ToastsContext = createContext<{ push: (message: string, kind?: ToastKind) => void }>({
  push: () => {},
});

export const useToasts = () => useContext(ToastsContext);

/** Shorten a server message for a toast: first sentence, capped. */
export function shortError(err: unknown, fallback = "Something went wrong."): string {
  const raw = err instanceof Error ? err.message : fallback;
  const first = raw.split(/(?<=[.!?])\s/)[0] ?? raw;
  return first.length > 110 ? `${first.slice(0, 107)}…` : first;
}

export function ToastsProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const nextID = useRef(0);

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const push = useCallback(
    (message: string, kind: ToastKind = "info") => {
      const id = ++nextID.current;
      setToasts((prev) => [...prev.slice(-3), { id, kind, message }]);
      setTimeout(() => dismiss(id), 6000);
    },
    [dismiss],
  );

  return (
    <ToastsContext.Provider value={{ push }}>
      {children}
      <div className={styles.viewport} aria-live="polite">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={`${styles.toast} ${styles[toast.kind]}`}
            role={toast.kind === "error" ? "alert" : "status"}
          >
            <span>{toast.message}</span>
            <button type="button" aria-label="Dismiss" onClick={() => dismiss(toast.id)}>
              ×
            </button>
          </div>
        ))}
      </div>
    </ToastsContext.Provider>
  );
}
