import { useState } from "react";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import type { TrackingConfig } from "../trackingApi";
import { addQuestions, starterQuestions } from "../starterQuestions";

/** Suggests local customer questions from what the business sells and where. */
export function QuestionStarter({
  config,
  onChange,
}: {
  config: TrackingConfig;
  onChange: (prompts: TrackingConfig["prompts"]) => void;
}) {
  const [business, setBusiness] = useState("");
  const [city, setCity] = useState("");
  const full = config.prompts.filter((p) => p.text.trim()).length >= 30;
  return (
    <div className="space-y-2 rounded-lg bg-muted p-3">
      <p className="text-sm font-medium">
        Not sure what to ask? Track the questions your customers ask.
      </p>
      <div className="grid gap-2 md:grid-cols-[2fr_2fr_auto]">
        <Input
          aria-label="What you sell"
          placeholder="What you sell, e.g. hotel, dental clinic, CA firm"
          value={business}
          maxLength={80}
          onChange={(e) => setBusiness(e.target.value)}
        />
        <Input
          aria-label="City or area"
          placeholder="City or area, e.g. Kemps Corner, Mumbai"
          value={city}
          maxLength={80}
          onChange={(e) => setCity(e.target.value)}
        />
        <Button
          variant="outline"
          disabled={!business.trim() || full}
          onClick={() =>
            onChange(
              addQuestions(
                config.prompts,
                starterQuestions({
                  brand: config.brand,
                  business,
                  city,
                  competitors: config.competitors.map((c) => c.name),
                }),
              ),
            )
          }
        >
          Suggest questions
        </Button>
      </div>
    </div>
  );
}
