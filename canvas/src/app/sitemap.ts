import type { MetadataRoute } from "next";

/**
 * app/sitemap.ts — Next.js 15 app-router sitemap
 *
 * Next.js automatically uses this as `/sitemap.xml` at serve time.
 * No pipeline change required — the framework handles it at build time.
 *
 * PRODUCTION OVERRIDE:
 *   Set NEXT_PUBLIC_CANVAS_SITE_URL in the build environment to
 *   reflect the actual deployed origin (e.g. https://app.moleculesai.app).
 *   The hardcoded default is safe for local dev / docker-compose.
 *
 *   In publish-canvas-image.yml, pass it as:
 *     build-args: NEXT_PUBLIC_CANVAS_SITE_URL=https://app.moleculesai.app
 *   and add to the Dockerfile:
 *     ARG NEXT_PUBLIC_CANVAS_SITE_URL=https://localhost:3000
 *     ENV NEXT_PUBLIC_CANVAS_SITE_URL=$NEXT_PUBLIC_CANVAS_SITE_URL
 */
const SITE_URL =
  process.env.NEXT_PUBLIC_CANVAS_SITE_URL ?? "https://app.moleculesai.app";

export default function sitemap(): Promise<MetadataRoute.Sitemap> {
  return Promise.resolve([
    {
      url: SITE_URL,
      lastModified: new Date(),
      changeFrequency: "daily",
      priority: 1.0,
    },
    {
      url: `${SITE_URL}/orgs`,
      lastModified: new Date(),
      changeFrequency: "weekly",
      priority: 0.9,
    },
    {
      url: `${SITE_URL}/pricing`,
      lastModified: new Date(),
      changeFrequency: "weekly",
      priority: 0.8,
    },
  ]);
}
