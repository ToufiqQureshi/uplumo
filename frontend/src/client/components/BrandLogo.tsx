import openai from "@lobehub/icons-static-svg/icons/openai.svg";
import gemini from "@lobehub/icons-static-svg/icons/gemini-color.svg";
import claude from "@lobehub/icons-static-svg/icons/claude-color.svg";
import perplexity from "@lobehub/icons-static-svg/icons/perplexity-color.svg";
import google from "@lobehub/icons-static-svg/icons/google-color.svg";
import {
  siBookingdotcom,
  siGoogleanalytics,
  siGooglemaps,
  siGooglesearchconsole,
  siInstagram,
  siReddit,
  siTripadvisor,
  siWhatsapp,
  siWikipedia,
  siWordpress,
  siYoutube,
  type SimpleIcon,
} from "simple-icons";

/**
 * Real brand marks for the AI engines and services Uplumo reads. Full colour
 * AI marks come from @lobehub/icons-static-svg (MIT), single colour marks from
 * simple-icons (CC0). They name a service; they are never Uplumo's brand.
 */
type Logo =
  | { image: string; label: string; invertInDark?: boolean }
  | { icon: SimpleIcon };

const LOGOS = {
  chatgpt: { image: openai, label: "ChatGPT", invertInDark: true },
  gemini: { image: gemini, label: "Gemini" },
  claude: { image: claude, label: "Claude" },
  perplexity: { image: perplexity, label: "Perplexity" },
  google: { image: google, label: "Google" },
  search_console: { icon: siGooglesearchconsole },
  google_analytics: { icon: siGoogleanalytics },
  google_maps: { icon: siGooglemaps },
  whatsapp: { icon: siWhatsapp },
  instagram: { icon: siInstagram },
  tripadvisor: { icon: siTripadvisor },
  booking: { icon: siBookingdotcom },
  wordpress: { icon: siWordpress },
  youtube: { icon: siYoutube },
  reddit: { icon: siReddit },
  wikipedia: { icon: siWikipedia },
} satisfies Record<string, Logo>;

type Brand = keyof typeof LOGOS;

// Engine ids used by the API, mapped to their logo.
const ENGINE_BRANDS: Partial<Record<string, Brand>> = {
  chat_gpt: "chatgpt",
  gemini: "gemini",
  claude: "claude",
  perplexity: "perplexity",
  google: "google",
  google_ai_overview: "google",
};

/** The logo of an API engine id (chat_gpt, gemini, ...), or nothing. */
export function EngineLogo({
  engine,
  size,
}: {
  engine: string;
  size?: number;
}) {
  const brand = ENGINE_BRANDS[engine];
  return brand ? <BrandLogo brand={brand} size={size} /> : null;
}

export function BrandLogo({
  brand,
  size = 16,
  decorative = true,
}: {
  brand: Brand;
  size?: number;
  /** True next to the brand's written name: screen readers skip it. */
  decorative?: boolean;
}) {
  const logo: Logo = LOGOS[brand];
  if ("image" in logo) {
    return (
      <img
        src={logo.image}
        alt={decorative ? "" : logo.label}
        width={size}
        height={size}
        loading="lazy"
        className={`inline-block shrink-0${logo.invertInDark ? " dark:invert" : ""}`}
      />
    );
  }
  return (
    <svg
      role={decorative ? undefined : "img"}
      aria-hidden={decorative ? true : undefined}
      aria-label={decorative ? undefined : logo.icon.title}
      viewBox="0 0 24 24"
      width={size}
      height={size}
      className="inline-block shrink-0"
      fill={`#${logo.icon.hex}`}
    >
      <path d={logo.icon.path} />
    </svg>
  );
}
