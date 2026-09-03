"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";

import { Toast, type ToastData, type ToastTone } from "@/components/ui/Toast";

interface ToastContextValue {
  showToast: (toast: { title: string; description?: string; tone?: ToastTone }) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

// Toda notificação some sozinha depois de 6s, além de poder ser fechada
// manualmente pelo botão "X" do Toast (ver dismiss abaixo).
const AUTO_DISMISS_MS = 6000;

// Provedor da pilha de toasts, montado uma vez no DashboardShell. Guarda
// a lista de toasts ativos em memória (não em Zustand/Redux — o estado é
// puramente local a esta árvore de componentes) e expõe showToast() via
// contexto para qualquer componente descendente disparar uma notificação
// visual, seja a partir de uma ação do usuário (IntegrationCard) ou de um
// evento de WebSocket (NotificationCenter).
export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastData[]>([]);
  const idRef = useRef(0);
  // P-05: guarda os timers de auto-dismiss para poder cancelá-los — sem
  // isto um `setTimeout` órfão chamava setState após o unmount do provider.
  const timersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  const dismiss = useCallback((id: string) => {
    const timer = timersRef.current.get(id);
    if (timer) {
      clearTimeout(timer);
      timersRef.current.delete(id);
    }
    setToasts((current) => current.filter((t) => t.id !== id));
  }, []);

  const showToast = useCallback<ToastContextValue["showToast"]>(
    ({ title, description, tone = "info" }) => {
      idRef.current += 1;
      const id = `toast-${idRef.current}`;
      setToasts((current) => [...current, { id, title, description, tone }]);
      timersRef.current.set(
        id,
        setTimeout(() => dismiss(id), AUTO_DISMISS_MS),
      );
    },
    [dismiss],
  );

  // Limpa todos os timers pendentes no unmount.
  useEffect(() => {
    const timers = timersRef.current;
    return () => {
      timers.forEach(clearTimeout);
      timers.clear();
    };
  }, []);

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      <div
        aria-live="polite"
        role="status"
        className="pointer-events-none fixed bottom-4 right-4 z-50 flex flex-col gap-2"
      >
        {toasts.map((toast) => (
          <div key={toast.id} className="pointer-events-auto">
            <Toast toast={toast} onDismiss={dismiss} />
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used within a ToastProvider");
  }
  return ctx;
}
