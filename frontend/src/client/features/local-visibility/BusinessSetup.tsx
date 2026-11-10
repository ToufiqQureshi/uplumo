import { type FormEvent, useState } from "react";
import { Button } from "@/client/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/client/components/ui/card";
import { Input } from "@/client/components/ui/input";
import { Label } from "@/client/components/ui/label";
import {
  type Business,
  type Place,
  useFindBusiness,
  useSaveBusiness,
} from "./localVisibilityApi";
import { PaidButton } from "./RunPanel";

/**
 * Picks the project's business from a Google Maps search, and the search
 * customers use to find it. Searching is paid; picking and saving are free.
 */
export function BusinessSetup({
  projectId,
  current,
  findCredits,
  onDone,
}: {
  projectId: string;
  current: Business | null;
  findCredits: number;
  onDone?: () => void;
}) {
  const find = useFindBusiness(projectId);
  const save = useSaveBusiness(projectId);
  const [query, setQuery] = useState("");
  const [picked, setPicked] = useState<Place | null>(null);
  const [term, setTerm] = useState(current?.searchTerm ?? "");
  const places = find.data?.places;

  const search = (event: FormEvent) => {
    event.preventDefault();
    if (query.trim().length < 2 || find.isPending) return;
    setPicked(null);
    find.mutate(query.trim());
  };
  const pick = (place: Place) => {
    setPicked(place);
    if (!term) setTerm(place.category);
  };
  const cid = picked?.cid ?? current?.cid;
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (!cid) return;
    save.mutate({ cid, searchTerm: term }, { onSuccess: () => onDone?.() });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          <h2>
            {current ? "Change business or search" : "Find your business"}
          </h2>
        </CardTitle>
        <CardDescription>
          Search Google Maps for your business by name. Add the city if the name
          is common.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <form onSubmit={search} className="flex flex-col gap-2 sm:flex-row">
          <Input
            aria-label="Business name"
            placeholder="Business name and city"
            value={query}
            maxLength={120}
            onChange={(e) => setQuery(e.target.value)}
          />
          <PaidButton
            label="Search"
            credits={findCredits}
            pending={find.isPending}
            disabled={query.trim().length < 2}
            type="submit"
          />
        </form>
        {find.error ? (
          <p role="alert" className="text-sm text-destructive">
            {find.error.message}
          </p>
        ) : null}
        {places && places.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            Google Maps found nothing for that name. Try the name as it appears
            on Google, with the city.
          </p>
        ) : null}
        {places && places.length > 0 ? (
          <ul className="divide-y rounded-lg border" aria-label="Results">
            {places.map((place) => (
              <li key={place.cid}>
                <button
                  type="button"
                  aria-pressed={picked?.cid === place.cid}
                  onClick={() => pick(place)}
                  className="w-full px-3 py-2 text-left text-sm hover:bg-muted aria-pressed:bg-primary/10"
                >
                  <span className="font-medium">{place.title}</span>
                  <span className="block text-xs text-muted-foreground">
                    {[
                      place.category,
                      place.address,
                      place.rating !== null
                        ? `★ ${place.rating} (${place.reviews})`
                        : "",
                    ]
                      .filter(Boolean)
                      .join(" · ")}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        ) : null}
        {cid ? (
          <form onSubmit={submit} className="space-y-2">
            <Label htmlFor="local-search-term">
              What do customers search on Maps to find a business like{" "}
              {picked?.title ?? current?.title}?
            </Label>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Input
                id="local-search-term"
                value={term}
                maxLength={80}
                onChange={(e) => setTerm(e.target.value)}
              />
              <Button
                type="submit"
                disabled={save.isPending || term.trim().length < 2}
              >
                Save
              </Button>
            </div>
            {save.error ? (
              <p role="alert" className="text-sm text-destructive">
                {save.error.message}
              </p>
            ) : null}
          </form>
        ) : null}
      </CardContent>
    </Card>
  );
}
