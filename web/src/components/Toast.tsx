"use client";

export function Toast({
  message,
  onDismiss,
}: {
  message: string | null;
  onDismiss: () => void;
}) {
  if (!message) return null;
  return (
    <div className="fixed bottom-6 left-1/2 z-50 -translate-x-1/2">
      <button
        type="button"
        onClick={onDismiss}
        className="rounded border border-rose-500/40 bg-rose-950/90 px-4 py-2 text-lg text-rose-200 shadow-lg"
      >
        {message}
      </button>
    </div>
  );
}
