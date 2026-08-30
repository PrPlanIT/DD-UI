// ui/src/components/ChunkFallback.tsx
import { Loader2 } from "lucide-react";

// Shown while a lazily-loaded chunk (monaco editor, xterm terminal) is fetched.
export default function ChunkFallback({ label = "Loading…" }: { label?: string }) {
  return (
    <div className="flex h-full w-full items-center justify-center gap-2 p-6 text-sm text-neutral-400">
      <Loader2 className="h-4 w-4 animate-spin" />
      <span>{label}</span>
    </div>
  );
}
