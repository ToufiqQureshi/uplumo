import { EngineLogo } from "@/client/components/BrandLogo";
import { engineLabel } from "./platformLabels";

/** An AI engine's logo and name, side by side. */
export function EngineName({
  engine,
  size = 16,
}: {
  engine: string;
  size?: number;
}) {
  return (
    <span className="inline-flex min-w-0 items-center gap-1.5">
      <EngineLogo engine={engine} size={size} />
      <span className="truncate">{engineLabel(engine)}</span>
    </span>
  );
}
