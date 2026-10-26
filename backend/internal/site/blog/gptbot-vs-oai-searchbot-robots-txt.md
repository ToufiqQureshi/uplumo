---
title: GPTBot vs OAI-SearchBot: Which OpenAI Crawler Should Your Site Allow?
description: Understand OpenAI’s crawler tokens for search and model training, verify access in logs, and set robots.txt rules that match your actual publishing policy.
date: 2026-10-15
updated: 2026-10-26
---

**Short answer:** OpenAI documents separate crawler controls: OAI-SearchBot is used to surface websites in ChatGPT search features, while GPTBot relates to content that may be used to train foundation models. Decide what your organization permits, use the documented product tokens in robots.txt, and monitor verified requests. Allowing a crawler does not guarantee a citation or inclusion.

![A practical troubleshooting flow for gptbot vs oai-searchbot: which openai crawler should your site allow?](/site/blog/gptbot-vs-oai-searchbot-robots-txt.svg)

## Why this problem happens

A broad “block all AI bots” rule can prevent a site from being discovered for a search product even when the publisher only meant to restrict training use. Conversely, allowing every bot without understanding the policy may not fit licensing or content requirements. Robots rules are public instructions, not access control.

## Check these signals before changing the site

1. Read OpenAI’s current crawler documentation and identify the exact user-agent token and purpose. Do not assume every bot with “GPT” in its name has the same use.
2. Review robots.txt for specific and wildcard rules, including matching user-agent groups and path rules. Confirm the file returns 200 and is served from the correct host.
3. Check server logs and verify crawler IPs using the verification method OpenAI publishes. A claimed user-agent is not proof of identity.
4. Ask legal/content owners whether your licensing, privacy, and syndication terms permit the relevant crawler. Respect private areas through authentication, not robots.txt.

## Fix it in a safe order

### 1. Write an explicit policy

Choose whether to allow search discovery, training-related crawling, both, or neither. Record the reason and owner so later robots changes do not undo the decision.

### 2. Use precise robots rules

Add the documented product token rules, check the rules in the matching user-agent group, and keep the file syntactically valid. Do not block CSS or essential public resources accidentally.

### 3. Test from each public host

Check robots.txt on the apex and subdomains where content is hosted. Confirm the intended crawler can fetch public pages and cannot access protected areas.

### 4. Measure outcomes cautiously

Review verified crawl logs and any available referral/citation data. Test the same public pages over time; AI answers can change and no crawler directive guarantees visibility.

## How to tell the fix worked

Keep dated copies of robots.txt and crawler logs. Confirm each rule’s effect on representative URLs and review OpenAI documentation before changing the policy, since product behavior may evolve.

## Mistakes that make this worse

Do not promise that allowing OAI-SearchBot guarantees ChatGPT citations. Do not expose private content on the assumption that robots rules enforce confidentiality.

## Example: allow search discovery while declining training crawls

For a publisher whose policy permits ChatGPT search discovery but declines GPTBot training-related crawling, this illustrates separate controls:

```text
User-agent: OAI-SearchBot
Allow: /
Disallow: /account/
Disallow: /api/

User-agent: GPTBot
Disallow: /
```

Adapt the protected paths and review the rest of the file. A specific matching user-agent group does not automatically inherit every rule in the `*` group; ensure its own rules express the intended policy. Authentication must still protect private resources even when a crawler ignores robots.txt.

| Token | Documented purpose | What the setting means |
| --- | --- | --- |
| `OAI-SearchBot` | Automatic crawling for ChatGPT search | Search-discovery policy, not a citation guarantee |
| `GPTBot` | Crawling that may support foundation-model training | Separate training-related preference |
| `ChatGPT-User` | User-triggered visits | Not the automatic search crawler |

OpenAI says robots.txt rules may not apply to user-initiated `ChatGPT-User` actions. A rule for that token is therefore not a substitute for access control, and it is not the switch that determines automatic search inclusion. Do not infer “ChatGPT cannot cite us” merely because that token appears in a block list.

After a change, inspect the robots.txt actually served through the CDN, not just the repository copy. Managed bot settings can prepend rules. Check verified requests from the published IP ranges, requested path, HTTP response and any WAF challenge. A locally spoofed user-agent tests your server's string handling; it does not demonstrate access from OpenAI infrastructure.

Keep retrieval permission, training preference, indexed inclusion and observed citations as four separate facts in your report.

## FAQ

**Does GPTBot control ChatGPT search visibility?**

OpenAI documents OAI-SearchBot for search discovery and GPTBot for potential training use. Check the current official documentation for the latest distinctions.

## Related Uplumo guides

- [What is answer engine optimisation (AEO)? A guide for small businesses](/blog/what-is-answer-engine-optimisation)
- [How to get your business cited in AI search answers](/blog/get-cited-in-ai-search-results)
- [How to track AI Overview visibility in Search Console](/blog/track-ai-overview-visibility-search-console)

## Sources

- [OpenAI crawlers documentation](https://developers.openai.com/api/docs/bots)
- [Google robots.txt overview](https://developers.google.com/search/docs/crawling-indexing/robots/intro)
