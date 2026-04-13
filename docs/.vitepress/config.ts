import { defineConfig } from "vitepress";

export default defineConfig({
  title: "Molecule AI",
  description: "The Organizational Operating System for AI Agents",
  head: [["link", { rel: "icon", href: "/assets/branding/molecule-icon.png" }]],

  themeConfig: {
    logo: "/assets/branding/molecule-icon.png",

    nav: [
      { text: "Guide", link: "/quickstart" },
      { text: "Architecture", link: "/architecture/architecture" },
      { text: "API", link: "/api-protocol/platform-api" },
      { text: "GitHub", link: "https://github.com/molecule-monorepo/molecule" },
    ],

    sidebar: [
      {
        text: "Getting Started",
        items: [
          { text: "Overview", link: "/product/overview" },
          { text: "Quickstart", link: "/quickstart" },
          { text: "Core Concepts", link: "/product/core-concepts" },
        ],
      },
      {
        text: "Architecture",
        collapsed: false,
        items: [
          { text: "System Architecture", link: "/architecture/architecture" },
          { text: "Technical Documentation", link: "/architecture/molecule-technical-doc" },
          { text: "Database Schema", link: "/architecture/database-schema" },
          { text: "Workspace Tiers", link: "/architecture/workspace-tiers" },
          { text: "Provisioner", link: "/architecture/provisioner" },
          { text: "Memory System", link: "/architecture/memory" },
          { text: "Event Log", link: "/architecture/event-log" },
          { text: "Technology Choices", link: "/architecture/technology-choices" },
        ],
      },
      {
        text: "API & Protocols",
        collapsed: false,
        items: [
          { text: "Platform API", link: "/api-protocol/platform-api" },
          { text: "A2A Protocol", link: "/api-protocol/a2a-protocol" },
          { text: "Communication Rules", link: "/api-protocol/communication-rules" },
          { text: "WebSocket Events", link: "/api-protocol/websocket-events" },
          { text: "Registry & Heartbeat", link: "/api-protocol/registry-and-heartbeat" },
        ],
      },
      {
        text: "Agent Runtime",
        collapsed: false,
        items: [
          { text: "Workspace Runtime", link: "/agent-runtime/workspace-runtime" },
          { text: "CLI Runtime", link: "/agent-runtime/cli-runtime" },
          { text: "Config Format", link: "/agent-runtime/config-format" },
          { text: "Agent Card", link: "/agent-runtime/agent-card" },
          { text: "Skills System", link: "/agent-runtime/skills" },
          { text: "Team Expansion", link: "/agent-runtime/team-expansion" },
          { text: "System Prompt", link: "/agent-runtime/system-prompt-structure" },
          { text: "Bundle System", link: "/agent-runtime/bundle-system" },
        ],
      },
      {
        text: "Frontend",
        collapsed: true,
        items: [
          { text: "Canvas Engine", link: "/frontend/canvas" },
        ],
      },
      {
        text: "Development",
        collapsed: true,
        items: [
          { text: "Local Development", link: "/development/local-development" },
          { text: "Build Order", link: "/development/build-order" },
          { text: "Observability", link: "/development/observability" },
          { text: "Code Sandbox", link: "/development/code-sandbox" },
          { text: "Constraints & Rules", link: "/development/constraints-and-rules" },
        ],
      },
      {
        text: "Product",
        collapsed: true,
        items: [
          { text: "Product Narrative", link: "/product/molecule-product-doc" },
          { text: "Landing Messaging Report", link: "/product/landing-messaging-report" },
          { text: "PRD", link: "/product/PRD" },
          { text: "SaaS Upgrade Path", link: "/product/saas-upgrade" },
        ],
      },
    ],

    search: {
      provider: "local",
    },

    editLink: {
      pattern: "https://github.com/molecule-monorepo/molecule/edit/main/docs/:path",
      text: "Edit this page on GitHub",
    },

    footer: {
      message: "Released under the MIT License.",
      copyright: "Copyright 2026 Molecule AI",
    },

    socialLinks: [
      { icon: "github", link: "https://github.com/molecule-monorepo/molecule" },
    ],
  },
});
