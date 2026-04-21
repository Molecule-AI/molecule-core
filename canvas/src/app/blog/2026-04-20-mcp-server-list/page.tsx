import { Metadata } from "next";
import { getMDXComponent } from "mdx-bundler";
import { getBlogPost } from "@/lib/blog";

export async function generateMetadata(): Promise<Metadata> {
  const post = await getBlogPost("2026-04-20-mcp-server-list");

  return {
    title: post.frontmatter.title,
    description: post.frontmatter.description,
    keywords: post.frontmatter.keywords,
    openGraph: {
      title: post.frontmatter.og_title ?? post.frontmatter.title,
      description: post.frontmatter.og_description ?? post.frontmatter.description,
      images: post.frontmatter.og_image ? [{ url: post.frontmatter.og_image }] : [],
      type: "article",
      publishedTime: post.frontmatter.date,
      authors: [post.frontmatter.author ?? "Molecule AI"],
    },
    twitter: {
      card: post.frontmatter.twitter_card ?? "summary_large_image",
      title: post.frontmatter.og_title ?? post.frontmatter.title,
      description: post.frontmatter.og_description ?? post.frontmatter.description,
      images: post.frontmatter.og_image ? [post.frontmatter.og_image] : [],
    },
    alternates: {
      canonical: post.frontmatter.canonical,
    },
  };
}

export default async function MCPServerListPage() {
  const post = await getBlogPost("2026-04-20-mcp-server-list");
  const Content = getMDXComponent(post.code);

  return (
    <article className="blog-post">
      <Content />
    </article>
  );
}