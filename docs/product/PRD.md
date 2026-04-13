# Molecule AI — 产品需求文档 (PRD)

> **Product Name:** Molecule AI  
> **Tagline:** *"构建你的 AI 组织 — 让每一个智能体成为团队，让每一个团队成为公司"*  
> **Version:** 1.0  
> **Date:** 2026-04-01  
> **Author:** Molecule AI Product Team  
> **Status:** Draft — Pending Review

---

## 目录

1. [产品愿景](#1-产品愿景)
2. [市场分析与竞争格局](#2-市场分析与竞争格局)
3. [核心差异化优势](#3-核心差异化优势)
4. [目标用户画像](#4-目标用户画像)
5. [核心功能需求](#5-核心功能需求)
6. [用户旅程 (User Journeys)](#6-用户旅程-user-journeys)
7. [技术架构概要](#7-技术架构概要)
8. [非功能性需求](#8-非功能性需求)
9. [分阶段交付计划](#9-分阶段交付计划)
10. [成功指标 (KPIs)](#10-成功指标-kpis)
11. [风险分析与缓解](#11-风险分析与缓解)
12. [附录](#12-附录)

---

## 1. 产品愿景

### 1.1 一句话定义

> Molecule AI 是一个**可视化 AI Agent Team 编排平台**。用户在画布上构建 AI 组织架构图 —— 拖拽角色、嵌套团队、配置技能 —— 每个节点背后运行一个真实的 AI 智能体。平台自动处理部署、发现、通信和可观测性。

### 1.2 核心信念

| 信念 | 解释 |
|------|------|
| **角色 > 任务** | 竞品节点代表"任务"或"工具"。Molecule AI 的节点代表"角色"（Role），即组织中的一个岗位。内部的 AI 模型可以随时替换，但角色位置、层级关系、技能配置不变。 |
| **组织架构即访问控制** | 无需手动连线。通信拓扑从 parent/child 层级结构自动派生。组织结构图就是安全策略。 |
| **分形递归** | 任何一个智能体节点都可以"展开"为一整个团队。从外部看，它仍然是同一个 A2A 端点。内部结构对外完全不透明 —— 与真实企业组织的分工逻辑一致。 |
| **标准协议** | 基于 Google/Linux Foundation 的 A2A 协议（Agent-to-Agent），任何符合 A2A 标准的智能体都可以直接接入，零供应商锁定。 |

### 1.3 产品不是什么

| ❌ Molecule AI 不是 | 说明 |
|-------------------|------|
| 工作流自动化工具（如 n8n） | 节点是角色，不是任务步骤 |
| 聊天界面 | 智能体间通过 A2A 协议程序化通信 |
| 模型供应商 | 用户自带 API Key (BYOK) |
| LangGraph 的替代品 | LangGraph 是每个 Workspace 内部的运行引擎 |
| 托管服务（MVP 阶段） | 自托管、开源优先 |

---

## 2. 市场分析与竞争格局

### 2.1 行业趋势 (2026)

1. **从单智能体到多智能体系统（Multi-Agent Systems, MAS）** —— 企业已不再满足单个 chatbot，需要多个专业化智能体协同完成复杂任务。
2. **"持久执行"成为金标准** —— 复杂、长周期的智能体流程需要状态持久化、人工审批门控和跨重启恢复。
3. **Human-on-the-Loop > Full Autonomy** —— 最成功的企业采用人类监督模式，而非完全自主。
4. **互操作性需求爆发** —— A2A 等开放协议的出现开始打破供应商锁定。
5. **运维成熟度分水岭** —— 成本监控、审计日志和失败熔断成为生产准入门槛。

### 2.2 竞品对比矩阵

| 维度 | **Molecule AI** | CrewAI | AutoGen | LangGraph | n8n / Flowise | Sim.ai |
|------|-------------|--------|---------|-----------|---------------|--------|
| **核心抽象** | 角色 (Role) | 角色 (Role) | 对话 (Chat) | 状态图 (Graph) | 任务 (Task) | 任务 (Task) |
| **递归团队** | ✅ 无限嵌套 | ❌ 扁平 | ❌ 扁平 | ❌ 手动编排 | ❌ 无 | ❌ 无 |
| **可视化画布** | ✅ 核心产品力 | ❌ 纯代码 | ❌ 纯代码 | ❌ 纯代码 | ✅ 画布 | ✅ 画布 |
| **通信协议** | A2A (标准) | 内部 API | 内部 API | 内部 API | HTTP Webhook | 内部 API |
| **分布式部署** | ✅ 多机 | ❌ 单进程 | ❌ 单进程 | ❌ 单进程 | ❌ 单机 | ❌ 单机 |
| **模型无关** | ✅ BYOK | ✅ | ✅ | ✅ | 部分 | 部分 |
| **安全隔离** | 4 级 Tier | ❌ 无 | ❌ 无 | ❌ 无 | ❌ 无 | ❌ 无 |
| **可观测性** | Langfuse 全链追踪 | 基础 | 基础 | LangSmith (SaaS) | 基础 | 基础 |
| **Bundle 市场** | ✅ (规划中) | ❌ | ❌ | Hub (社区) | 模板 | ❌ |
| **开源** | ✅ MIT | ✅ | ✅ | ✅ | ✅ Community | ❌ |

### 2.3 竞争核心结论

> [!IMPORTANT]
> **Molecule AI 的本质差异不在于"又一个多智能体框架"，而在于它是唯一一个将"组织架构"作为第一公民抽象的可视化平台。** 它不竞争 LangGraph 的底层引擎能力，而是将 LangGraph 包装在一个可视化、可嵌套、可分发的组织层中。

---

## 3. 核心差异化优势

### 3.1 差异化金字塔

```
                    ┌───────────────────────┐
                    │   Bundle Marketplace   │  ← 商业壁垒
                    │  (角色的 App Store)     │
                    ├───────────────────────┤
                    │   递归分形团队展开      │  ← 产品体验壁垒
                    │   (节点展开为子团队)    │
                    ├───────────────────────┤
                    │   组织即拓扑            │  ← 架构壁垒
                    │   (层级 = 访问控制)     │
                    ├───────────────────────┤
                    │   A2A 标准协议          │  ← 生态壁垒
                    │   (零锁定，可互操作)    │
                    └───────────────────────┘
```

### 3.2 九大差异化要点

| # | 差异化 | 竞品现状 | Molecule AI 方案 |
|---|--------|---------|---------------|
| **D1** | **角色抽象 vs 任务抽象** | 节点 = 一个 API 调用或工具 | 节点 = 一个组织岗位，内部 AI 可随时热替换 |
| **D2** | **递归团队展开** | 扁平节点列表，无嵌套 | 任何节点可展开为子团队，子团队可再展开，无限递归 |
| **D3** | **组织即拓扑** | 手动连线 / 白名单 | 拖入嵌套自动建立通信关系，zero wiring |
| **D4** | **分布式 A2A 通信** | 单进程内部调用 | 节点可分布在不同机器，通过 A2A JSON-RPC 2.0 直连 |
| **D5** | **4 级安全隔离** | 所有节点共享同一运行时 | Tier 1-3 容器隔离，Tier 4 提供完整宿主机访问能力 |
| **D6** | **层级审批链** | 扁平人工介入 | 智能体沿组织层级逐级上报，直到根节点暴露给人类 |
| **D7** | **跨 Workspace 全链可观测** | 单节点 tracing | 统一 Langfuse 实例，跨所有 Workspace 的 LLM 调用链 |
| **D8** | **Bundle 可分发/可交易** | 无便携格式 | `.bundle.json` 标准格式，未来支持市场化售卖 |
| **D9** | **层次化记忆架构 (HMA)** | 全局共享向量库（越权、噪音大） | 按汇报线隔离的 Local/Team/Global 三级记忆存储 |

---

## 4. 目标用户画像

### 4.1 Persona 1 — 技术架构师 (Technical Architect)

| 属性 | 描述 |
|------|------|
| 背景 | 10+ 年后端经验，熟悉分布式系统和 DevOps |
| 痛点 | 需要编排多个 AI 智能体协作，但现有框架要么太底层（LangGraph），要么不支持分布式（CrewAI） |
| 需求 | 可视化编排 + 代码级控制、生产级部署、安全隔离 |
| 价值主张 | "在画布上拖拽即可构建企业级多智能体系统，不牺牲任何工程控制力" |

### 4.2 Persona 2 — 业务运营者 (Business Operator)

| 属性 | 描述 |
|------|------|
| 背景 | 业务管理者，非技术背景，但理解团队组织管理 |
| 痛点 | 想自动化复杂的多步骤业务流程（如 SEO + 内容 + 投放），但没有编程能力 |
| 需求 | 像搭建组织架构一样部署 AI 团队，无需写代码 |
| 价值主张 | "像管理人类团队一样管理 AI 团队 — 定义角色、分配技能、设置汇报线" |

### 4.3 Persona 3 — Workspace 作者 (Workspace Author / Skill Developer)

| 属性 | 描述 |
|------|------|
| 背景 | AI/LLM 应用开发者，熟悉 Prompt Engineering 和工具开发 |
| 痛点 | 开发的智能体/技能缺乏可复用的分发渠道 |
| 需求 | 将自己的专业知识打包为可复用、可交易的 Workspace Bundle |
| 价值主张 | "把你的 AI 专业能力打包成产品，一键分发或在市场上售卖" |

### 4.4 Persona 4 — 企业管理员 (Enterprise Admin)

| 属性 | 描述 |
|------|------|
| 背景 | IT 部门负责人，关注安全合规、成本管控 |
| 痛点 | AI 智能体运行不透明、无审计轨迹、成本不可控 |
| 需求 | 全链可观测、层级审批机制、秘钥加密管理、多租户隔离 |
| 价值主张 | "每一次 LLM 调用都有迹可循，每一次危险操作都需层级审批" |

---

## 5. 核心功能需求

### F1 — 可视化 Org Canvas (Visual Org Canvas)

**目标：** 提供类 Figma 的画布体验，以组织架构图的形式创建和管理 AI 智能体。

| ID | 功能点 | 优先级 | 描述 |
|----|--------|--------|------|
| F1.1 | 画布渲染 | P0 | 基于 React Flow 的无限画布，支持缩放、平移、小地图 |
| F1.2 | WorkspaceNode 组件 | P0 | 每个节点展示：名称、状态指示灯（green/gray/yellow/red）、Tier 徽章、技能列表、活跃任务计数器 |
| F1.3 | 层级边自动渲染 | P0 | parent/child 关系自动生成连线，无需手动连接 |
| F1.4 | 拖拽嵌套 | P0 | 将节点拖入另一个节点即建立 parent/child 关系。拖出到画布取消嵌套 |
| F1.5 | 模板面板 (Template Palette) | P0 | 侧边栏展示可用 Workspace 模板，点击即可配置并部署 |
| F1.6 | 快速配置弹窗 | P0 | 选择模板后弹出：名称、模型选择、父节点选择，预填默认值 |
| F1.7 | 节点位置持久化 | P0 | 拖拽停止后 PATCH 到后端 Postgres，跨浏览器一致 |
| F1.8 | 视口记忆 | P1 | 保存画布缩放/平移状态，再次打开恢复上次视角 |
| F1.9 | 实时状态同步 | P0 | WebSocket 推送所有状态变更，节点颜色/徽章实时更新 |
| F1.10 | 节点右键菜单 | P0 | 导出 Bundle、复制节点、展开为团队、删除 |
| F1.11 | 团队缩放视图 | P1 | 展开的节点支持"Zoom-in"查看子 Workspace 内部结构 |
| F1.12 | Bundle 拖放导入 | P1 | 拖拽 `.bundle.json` 文件到画布即可导入并部署 |
| F1.13 | 连接断裂可视化 | P2 | 委派失败时，两节点之间的边显示警告指示 |

### F2 — 递归团队展开 (Recursive Team Expansion)

**目标：** 任何 Workspace 节点可展开为一个子团队，递归无限深度。

| ID | 功能点 | 优先级 | 描述 |
|----|--------|--------|------|
| F2.1 | 展开操作 | P0 | 右键 → "展开为团队" 或 API `POST /workspaces/:id/expand` |
| F2.2 | 子节点自动部署 | P0 | 读取 config.yaml 的 `sub_workspaces` 字段，自动 provision 子容器 |
| F2.3 | 团队领导保留 | P0 | 展开后，原节点的 Agent 保留为协调员（Team Lead），接收上游消息并分发 |
| F2.4 | 作用域隔离 | P0 | 子节点只能与同级 sibling 和 Team Lead 通信，不能直接联系外部 |
| F2.5 | 折叠操作 | P1 | `POST /workspaces/:id/collapse` 停止子节点，Team Lead 恢复为独立执行 |
| F2.6 | 删除前拖出 | P1 | 删除团队时可先将子节点拖出保留，再级联删除剩余 |
| F2.7 | 事件广播 | P0 | `WORKSPACE_EXPANDED` / `WORKSPACE_COLLAPSED` 事件触发画布更新 |

### F3 — A2A 标准通信 (Agent-to-Agent Protocol)

**目标：** 所有 Workspace 间通过开放标准 A2A 协议直接通信，Platform 只做服务发现。

| ID | 功能点 | 优先级 | 描述 |
|----|--------|--------|------|
| F3.1 | Agent Card 发布 | P0 | 每个 Workspace 在 `/.well-known/agent-card.json` 发布身份文档 |
| F3.2 | 按需发现 | P0 | 委派时才查询 Platform 解析目标 URL，不预推送拓扑 |
| F3.3 | 层级访问检查 | P0 | `CanCommunicate()` 基于 parent_id 层级强制执行 403 Forbidden |
| F3.4 | Peer 发现 API | P0 | `GET /registry/:id/peers` 返回当前节点可达的所有 Workspace |
| F3.5 | 同步/流式调用 | P0 | `message/send` (同步短任务) 和 `message/sendSubscribe` (SSE 流式长任务) |
| F3.6 | 任务生命周期 | P0 | submitted → working → completed/failed/canceled，含 `input-required` 中间态 |
| F3.7 | 委派失败处理 | P0 | 3 次重试 + 指数退避 + 可选 fallback workspace + LLM 自主决策 |
| F3.8 | 签名令牌 (Post-MVP) | P2 | Platform 颁发短生命周期签名令牌，目标 Workspace 验证每次 A2A 请求 |

### F4 — Skills 生态系统 (Skills Ecosystem)

**目标：** 模块化的技能包系统，支持热加载、ClawHub 生态兼容、MCP 工具集成。

| ID | 功能点 | 优先级 | 描述 |
|----|--------|--------|------|
| F4.1 | Skill 包格式 | P0 | `SKILL.md` (YAML frontmatter + Markdown 指令) + `tools/` (LangChain @tool) |
| F4.2 | 运行时热加载 | P0 | 文件监控 → 2s 去抖 → 重新扫描 skills → 重建 Agent Card → 广播 AGENT_CARD_UPDATED |
| F4.3 | 画布拖放技能 | P1 | 从技能面板拖拽技能到节点，自动复制文件到容器 volume |
| F4.4 | ClawHub 兼容 | P1 | `npx clawhub@latest install <skill-name>` 安装社区技能 |
| F4.5 | 三种技能类型 | P0 | 纯上下文（仅 SKILL.md）/ 混合（SKILL.md + tools）/ 纯工具（仅 tools） |
| F4.6 | 环境依赖声明 | P1 | frontmatter 中声明 `requires.env` / `requires.bins`，启动时校验 |

### F5 — Bundle 便携系统 (Bundle System)

**目标：** Workspace 的完整可移植格式，支持导出/导入/复制/未来市场交易。

| ID | 功能点 | 优先级 | 描述 |
|----|--------|--------|------|
| F5.1 | 导出 | P0 | 右键 → 导出为 `.bundle.json`，内含 system prompt、所有 skill 文件、工具配置、递归子 Workspace |
| F5.2 | 导入 | P0 | 拖放到画布 → `POST /bundles/import` → 递归部署 |
| F5.3 | 复制 | P1 | 导出 + 全新 ID 重新导入，两个实例完全独立 |
| F5.4 | 秘钥隔离 | P0 | Bundle 绝不包含 API Key 或密码。导入方自带凭据 |
| F5.5 | 部分失败处理 | P1 | 子节点部署失败不阻塞父节点，失败节点标红提供重试按钮 |
| F5.6 | 来源追溯 | P1 | `source_bundle_id` 字段记录实例来源模板 |
| F5.7 | 市场流通 (Future) | P3 | 卖家上架 Bundle + 定价 → 买家购买 → 平台在买方环境部署 |

### F6 — 层级审批链 (Hierarchical Human-in-the-Loop)

**目标：** 智能体遇到需要审批的操作时，沿组织层级逐级上报，Root 节点暴露给人类。

| ID | 功能点 | 优先级 | 描述 |
|----|--------|--------|------|
| F6.1 | LangGraph 暂停集成 | P0 | 利用 LangGraph 原生 interrupt 机制暂停执行 |
| F6.2 | 层级上报 | P0 | 子节点 → 父节点，父节点可 approve / deny / 继续上报 |
| F6.3 | Root 节点人工界面 | P0 | 根节点收到审批请求时，在画布 UI 上弹出审批卡片 |
| F6.4 | 审批结果回传 | P0 | 审批结果沿层级向下传递，触发子节点恢复/中止 |
| F6.5 | 可配置审批规则 | P1 | 在 system prompt 中定义哪些操作需要审批（破坏性操作、高成本操作等） |

### F7 — 跨 Workspace 可观测性 (Cross-Workspace Observability)

**目标：** 统一 Langfuse 追踪平台，跨所有 Workspace 查看完整 LLM 调用链。

| ID | 功能点 | 优先级 | 描述 |
|----|--------|--------|------|
| F7.1 | 自动 LLM Tracing | P0 | LangGraph 检测 Langfuse 环境变量自动注入追踪，零配置 |
| F7.2 | 追踪内容 | P0 | LLM 调用 (prompt/output/tokens/cost)、工具调用、规划步骤、错误堆栈 |
| F7.3 | A2A 委派跨度 | P1 | 手动 span 链接：`parent_task_id` 将子 Workspace trace 关联到父链路 |
| F7.4 | 统一视图 | P0 | 所有 Workspace 报告到同一 Langfuse 实例，提供全局调用树 |
| F7.5 | 画布内联追踪 (Future) | P3 | 点击节点直接查看最近 LLM 调用摘要，无需切换到 Langfuse |

### F8 — 分级安全隔离 (Tiered Security Isolation)

**目标：** 4 级安全分级，不同角色获得不同的系统权限和隔离程度。

| ID | 功能点 | 优先级 | 描述 |
|----|--------|--------|------|
| F8.1 | Tier 1 — 无特权容器 | P0 | 只读文件系统，纯文本/数据处理 |
| F8.2 | Tier 2 — 标准容器 | P0 | 资源受限 Docker + `/workspace` 挂载，适合大多数开发/协调型智能体 |
| F8.3 | Tier 3 — 特权容器 | P1 | `--privileged` + host PID，保留 Docker 网络，适合高权限开发操作 |
| F8.4 | Tier 4 — 完整宿主机访问 | P2 | privileged + host PID + host network + Docker socket，适合 DevOps / orchestrator 类工作区 |
| F8.5 | 秘钥加密存储 | P0 | AES-256 应用层加密，`SECRETS_ENCRYPTION_KEY` 环境变量管理密钥 |
| F8.6 | 代码沙箱 | P2 | Tier 3+ 的代码执行在一次性容器中运行（网络禁用、内存限制、执行后销毁） |

---

## 6. 用户旅程 (User Journeys)

### Journey 1 — 首次上手：部署一个 SEO Agent 团队

```
用户打开 Molecule AI Canvas (localhost:3000)
      │
      ▼
空白画布 — 左侧 Template Palette 展示可用模板
      │
      ▼
用户点击 "SEO Agent" 模板
      │
      ▼
弹出快速配置：
  • 名称: "Reno Stars SEO Agent"
  • 模型: Claude Sonnet (下拉选择)
  • 父节点: 无 (根级别)
      │
      ▼
用户确认 → POST /workspaces
      │
      ▼
画布出现新节点 🔄 (provisioning)
      │
      ▼
~30 秒后节点变绿 🟢 (online)
技能徽章: [Generate SEO Page] [Audit SEO Page]
      │
      ▼
用户右键节点 → "展开为团队"
      │
      ▼
节点展开：SEO Lead + Keyword Agent + Writer Agent + QA Agent
子节点依次上线，边自动渲染
      │
      ▼
用户在 Langfuse 中看到完整的跨 Agent 调用链 ✅
```

**验收标准 (Acceptance Criteria):**
- [ ] 从选择模板到节点上线 < 60 秒
- [ ] 节点状态从 provisioning 到 online 实时更新，无刷新
- [ ] 团队展开后子节点自动部署并建立正确的层级关系
- [ ] 子节点间只能与 sibling 和 Team Lead 通信，直接联系外部返回 403
- [ ] Langfuse 中可看到从 SEO Lead 到各子 Agent 的完整调用树

### Journey 2 — 构建多层组织：AI 软件开发公司

```
用户已有一个顶层 "Business Core" 节点
      │
      ▼
从模板添加: Marketing Agent, Developer PM, Operations Agent
拖入 Business Core 建立 parent/child 关系
      │
      ▼
画布自动渲染 Business Core → 三个子节点的组织架构
      │
      ▼
用户展开 Developer PM 为团队:
  Developer PM (协调员)
    ├── Frontend Agent
    ├── Backend Agent
    └── QA PM
      │
      ▼
进一步展开 QA PM 为团队:
  QA PM (协调员)
    ├── Auto Test Agent
    └── Manual Review Agent
      │
      ▼
三层组织架构完成 ✅
通信规则自动生效:
  • Frontend ↔ Backend ↔ QA PM (siblings) ✅
  • Frontend → Developer PM (up to parent) ✅
  • Frontend → Business Core (skip level) ❌ 403
  • Frontend → Marketing (cross-team) ❌ 403
```

**验收标准:**
- [ ] 三层嵌套结构正确建立，画布正确渲染所有层级
- [ ] 通信访问控制严格按照层级规则执行
- [ ] 拖出子节点到画布根级别后，该节点失去原 parent/child 关系
- [ ] 删除 QA PM 时，弹出警告列出所有将被级联删除的子节点

### Journey 3 — Bundle 分发：导出并在另一环境复现

```
用户右键 Developer PM 节点 → "导出为 Bundle"
      │
      ▼
下载 developer-pm.bundle.json
内含: system prompt + 3 个子 Workspace 的完整定义（递归）
不含: API keys
      │
      ▼
用户将文件分享给同事
      │
      ▼
同事打开自己的 Molecule AI，拖拽 .bundle.json 到画布
      │
      ▼
POST /bundles/import → 递归创建 4 个 Workspace
全新 ID，保留 source_bundle_id 追溯
      │
      ▼
同事配置自己的 API Key
      │
      ▼
完整的 Developer PM 团队在同事环境中上线 ✅
```

### Journey 4 — 层级审批：危险操作上报人类

```
Frontend Agent 决定需要删除生产数据库的旧表
      │
      ▼
Frontend Agent 暂停 (LangGraph interrupt)
发送审批请求 → Developer PM (parent)
      │
      ▼
Developer PM 的 LLM 评估：
"这是破坏性操作，超出我的权限"
      │
      ▼
Developer PM 继续上报 → Business Core (its parent)
      │
      ▼
Business Core 是根节点 →
审批请求通过画布 UI 暴露给人类用户
      │
      ▼
人类在画布上看到审批卡片：
"Frontend Agent 请求删除 production.legacy_table"
[批准] [拒绝]
      │
      ▼
人类点击 [拒绝]
      │
      ▼
拒绝信号沿链路回传：
Business Core → Developer PM → Frontend Agent
      │
      ▼
Frontend Agent 收到拒绝，采取替代方案 ✅
```

---

## 7. 技术架构概要

### 7.1 系统全景图

```
┌─────────────────────────────────────────────────────────────┐
│                    Molecule AI Canvas                           │
│           Next.js 15 · React Flow · Zustand · TailwindCSS   │
│    ┌──────────┐  ┌──────────┐  ┌──────────┐                │
│    │WorkspaceNode│ │TemplatePanel│ │ApprovalCard│ ...         │
│    └────┬─────┘  └────┬─────┘  └─────┬────┘                │
│         │ HTTP REST    │              │ WebSocket             │
└─────────┼──────────────┼──────────────┼─────────────────────┘
          │              │              │
┌─────────▼──────────────▼──────────────▼─────────────────────┐
│                   Molecule AI Platform                          │
│              Go (Gin) · REST API · WebSocket Hub             │
│                                                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │ Registry │ │Provisioner│ │ Bundler  │ │ Broadcaster│      │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘       │
│       │             │            │             │              │
│  ┌────▼─────┐  ┌────▼────┐                                  │
│  │ Postgres │  │  Redis  │     Events Flow:                  │
│  │  (SoT)   │  │ (Cache) │     Action → DB Insert → Redis   │
│  └──────────┘  └─────────┘     Pub/Sub → WebSocket Hub →    │
│                                 Canvas + Workspace Clients   │
└─────────────────────────────────────────────────────────────┘
          ↕ A2A JSON-RPC 2.0 (Direct, P2P)
┌─────────────────────────────────────────────────────────────┐
│              Workspace Runtime (per instance)                │
│         Python · Deep Agents · LangGraph · a2a-sdk        │
│                                                             │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        │
│  │ Agent   │ │  Skills  │ │ Heartbeat│ │  Memory  │         │
│  │(LangGraph)││(SKILL.md)│ │(30s loop)│ │(fs/pg/s3)│        │
│  └────┬────┘ └─────┬────┘ └────┬─────┘ └──────────┘        │
│       │ Traces      │           │                            │
│  ┌────▼─────────────▼───────────▼──────────────────┐        │
│  │              Langfuse (Observability)            │        │
│  │          Unified cross-workspace tracing         │        │
│  └─────────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

### 7.2 技术栈确认

| 层级 | 技术 | 版本 | 选型理由 |
|------|------|------|---------|
| **Canvas** | Next.js | 15 | App Router + React Server Components + 易于代理后端 |
| **Canvas** | React Flow (@xyflow/react) | v12 | 业界标准可视化画布库 |
| **Canvas** | Zustand | latest | 轻量状态管理，契合 React Flow 受控模式 |
| **Canvas** | TailwindCSS | v4 | 快速 UI 开发，暗色主题支持 |
| **Platform** | Go + Gin | Go 1.22+ | 高并发心跳/WebSocket，goroutine 模型 |
| **Platform** | PostgreSQL | 16 | 事实源，append-only 事件日志，JSONB Agent Card |
| **Platform** | Redis | 7 | TTL 活跃检测 + Pub/Sub 事件广播 + URL 缓存 |
| **Runtime** | Python | 3.11+ | Deep Agents / LangGraph 的原生语言 |
| **Runtime** | Deep Agents | 0.4+ | Agent 封装（TODO 规划、子 Agent、文件系统记忆） |
| **Runtime** | LangGraph | latest | Agent 循环、状态持久化、流式处理、Human-in-the-Loop |
| **Runtime** | a2a-sdk | latest | A2A 服务端包装（JSON-RPC 路由 + Agent Card 自动发布） |
| **Observability** | Langfuse | 3.x (self-hosted) | 开源可自托管，LangGraph 原生集成 |
| **Infra** | Docker Compose | 2.x | 本地开发全栈启动 |

### 7.3 数据库核心模型

| 表 | 用途 | 关键字段 |
|----|------|---------|
| `workspaces` | Workspace 注册表（当前态） | `id`, `name`, `status`, `parent_id`, `agent_card` (JSONB), `tier`, `url` |
| `agents` | Agent 分配历史 | `workspace_id`, `model`, `status` |
| `structure_events` | 不可变事件日志 (APPEND-ONLY) | `event_type`, `workspace_id`, `payload` (JSONB) |
| `workspace_secrets` | 加密凭据 | `workspace_id`, `key`, `encrypted_value` (AES-256) |
| `canvas_layouts` | 节点画布位置 | `workspace_id`, `x`, `y`, `collapsed` |

### 7.4 实时数据流

```
1. 操作发生 (register / heartbeat / expand / ...)
      │
      ▼
2. broadcaster.RecordAndBroadcast()
   → INSERT INTO structure_events (append-only)
   → PUBLISH to Redis pub/sub channel
      │
      ▼
3. Redis subscriber relay → WebSocket Hub
      │
      ▼
4. Hub broadcasts:
   → Canvas clients: 所有事件 (更新全局视图)
   → Workspace clients: 过滤后事件 (仅 CanCommunicate 可达节点)
      │
      ▼
5. Canvas: Zustand applyEvent() → React Flow re-render
   Workspace: 重建 system prompt (如需)
```

---

## 8. 非功能性需求

### 8.1 性能

| 指标 | 目标 |
|------|------|
| Workspace 上线时间 (Provisioning → Online) | < 60s (Tier 1-3), < 180s (Tier 4) |
| 心跳间隔 | 30s |
| 心跳处理吞吐 | > 1000 次/秒 (Platform) |
| WebSocket 事件延迟 | < 200ms (操作到画布更新) |
| 画布渲染节点数 | > 100 节点流畅 |
| A2A 发现延迟 | < 50ms (Redis 命中) |

### 8.2 可靠性

| 指标 | 目标 |
|------|------|
| Redis 丢失恢复 | 下次心跳自动重建状态 |
| WebSocket 断连恢复 | 指数退避重连 + 全量 re-hydrate |
| Bundle 导入部分失败 | 成功节点保持运行，失败节点提供重试 |
| 委派失败 | 3 次重试 + 退避 + fallback + LLM 决策 |

### 8.3 安全

| 要求 | 实现 |
|------|------|
| Workspace 间认证 (MVP) | 发现时验证 `CanCommunicate()`，直连无认证 |
| Workspace 间认证 (Post-MVP) | Platform 签发短效签名令牌 |
| 秘钥存储 | Postgres + AES-256 应用层加密 |
| Bundle 安全 | 不序列化任何凭据 |
| Tier 4 隔离 | 完整宿主机访问级别的 Docker 配置 |
| Docker 网络 | 所有容器在 `molecule-monorepo-net` 私有网络内 |

### 8.4 可扩展性

| 方向 | 设计支持 |
|------|---------|
| 多机部署 | A2A 协议天然跨机器，节点可在任何主机运行 |
| 多租户 (Future) | Schema 预留 `org_id` 扩展位 |
| Marketplace (Future) | Bundle 格式已标准化，可直接挂载商业层 |
| 自定义 Provider | LangChain 兼容字符串格式，支持 Anthropic/OpenAI/Ollama/本地模型 |

---

## 9. 分阶段交付计划

### Phase 1 — Foundation (基石期) · 8 周

> **目标：** 证明核心循环 —— Workspace 注册 → 画布显示 → 心跳保活 → 离线检测 → 画布变灰。

| 周次 | 里程碑 | 交付物 |
|------|--------|--------|
| W1-2 | 基础设施 + 数据库 | Docker Compose (Postgres/Redis/Langfuse) + 5 个 Migration 文件 |
| W2-3 | Platform API 骨架 | Go/Gin 服务启动，CORS，连接 PG/Redis |
| W3-4 | Registry 端点 | register / heartbeat / update-card + Redis TTL 活跃检测 |
| W4-5 | Workspace Runtime | Python 模板 + 最小 Echo Agent + A2A 包装 + 心跳 |
| W5-6 | Canvas 骨架 | Next.js + React Flow + Zustand + WorkspaceNode + 初始加载 |
| W6-7 | WebSocket 实时更新 | 事件广播 + 画布实时节点状态更新 |
| W7-8 | 第一个真实 Workspace | SEO Agent 配置完成，端到端从启动到画布可见 |

**Phase 1 完成标准：** 一个 SEO Agent Workspace 从容器启动到画布显示为绿色节点，心跳停止后变为灰色，全流程端到端通过。

### Phase 2 — Growth (增长期) · 6 周

> **目标：** 组织架构能力 + 通信 + Bundle 系统，用户可以构建多层 AI 组织。

| 周次 | 里程碑 | 交付物 |
|------|--------|--------|
| W9-10 | 层级 & 通信 | `CanCommunicate()` + Peer 发现 + 画布拖拽嵌套 |
| W10-11 | 团队展开/折叠 | expand/collapse API + 递归子 Workspace 部署 |
| W11-12 | Bundle 导入/导出 | exporter + importer + 画布拖放 |
| W12-13 | 模板面板 | 侧边栏模板列表 + 快速配置弹窗 |
| W13-14 | 层级审批 | Human-in-the-loop 层级上报 + 画布审批卡片 |

**Phase 2 完成标准：** 用户能构建 3 层组织架构，通信规则正确执行，Bundle 可导出/导入，审批链从叶节点上报到人类。

### Phase 3 — Enterprise (企业期) · 6 周

> **目标：** 安全隔离、分级部署、高级可观测性、SaaS 扩展准备。

| 周次 | 里程碑 | 交付物 |
|------|--------|--------|
| W15-16 | Tier 2-3 部署 | Playwright / Xvfb 容器配置 |
| W17-18 | Tier 4 Full-Host 部署 | Host network / Docker socket / 特权运行时策略 |
| W18-19 | 代码沙箱 | Tier 3+ Docker-in-Docker 沙箱 |
| W19-20 | SaaS 准备 | Auth 抽象层 + org_id 扩展 + Stripe 集成点 |

---

## 10. 成功指标 (KPIs)

### 10.1 产品指标

| 指标 | Phase 1 目标 | Phase 2 目标 | Phase 3 目标 |
|------|-------------|-------------|-------------|
| 从模板到节点上线时间 | < 120s | < 60s | < 30s |
| 画布流畅节点数量 | 20+ | 50+ | 100+ |
| Bundle 导入成功率 | — | > 95% | > 99% |
| 委派成功率 (首次) | — | > 90% | > 95% |
| WebSocket 重连恢复时间 | < 10s | < 5s | < 3s |

### 10.2 社区指标 (开源)

| 指标 | 6 个月 | 12 个月 |
|------|--------|---------|
| GitHub Stars | 1,000 | 5,000 |
| 社区 Workspace Bundle 数量 | 10 | 50 |
| 月活跃 Self-Hosted 部署 | 50 | 500 |
| ClawHub 上架 Skills 数量 | 20 | 100 |

---

## 11. 风险分析与缓解

| # | 风险 | 影响 | 概率 | 缓解策略 |
|---|------|------|------|---------|
| R1 | **A2A 协议尚未广泛采用** | 生态兼容性受限 | 中 | Molecule AI 本身推动 A2A 落地，提供参考实现；保留 HTTP fallback |
| R2 | **LangGraph/Deep Agents 版本迭代** | Runtime 适配成本 | 高 | 抽象 Agent 接口层，隔离底层框架变更 |
| R3 | **画布性能瓶颈 (100+ 节点)** | 复杂组织架构下 UX 降级 | 中 | 虚拟化渲染 + 团队折叠；外层只看单个团队节点 |
| R4 | **MVP 无认证的安全隐患** | 如果用户暴露到公网 | 低 | 文档明确标注 "仅限可信网络"；Post-MVP 优先加签名令牌 |
| R5 | **多 AI Provider 成本不可控** | 用户不知道花了多少钱 | 中 | Langfuse 自带 Token/Cost 追踪；画布节点展示累计 cost |
| R6 | **递归团队深度过大** | 延迟爆炸 + 调试困难 | 低 | 默认建议 ≤ 4 层深度，超出时 UI 警告 |

---

## 12. 附录

### 12.1 术语表

| 术语 | 定义 |
|------|------|
| **Workspace** | Molecule AI 的基本单元。一个组织角色，内含一个 AI Agent，对外提供 A2A 端点 |
| **Agent** | Workspace 内部的 AI 模型实例，可热替换 |
| **Agent Card** | 发布在 `/.well-known/agent-card.json` 的身份文档，描述能力和技能 |
| **Bundle** | `.bundle.json` 可移植文件，包含 Workspace 完整配置（递归含子 Workspace） |
| **Skill** | 可加载的技能包 (SKILL.md + tools/)，赋予 Agent 特定能力 |
| **Tier** | 安全等级 (1-4)，决定 Workspace 的隔离程度和部署方式 |
| **A2A** | Agent-to-Agent 协议，JSON-RPC 2.0 over HTTP，Workspace 间直连通信 |
| **Team Expansion** | 将单个 Workspace 展开为包含子 Workspace 的团队 |
| **Platform** | Go 后端控制平面，负责注册、发现、事件广播、部署 |
| **Canvas** | Next.js 前端可视化画布，用户在此构建和管理 AI 组织 |

### 12.2 关键设计约束

> [!CAUTION]
> 以下约束在任何情况下不得违反：

1. **Platform 永远不路由 Agent 消息** — A2A 消息是点对点 (P2P) 的
2. **Postgres 是事实源，Redis 是临时缓存** — Redis 丢失可自动恢复
3. **`structure_events` 表永远只 INSERT** — 不 UPDATE，不 DELETE
4. **`workspace-template` 不含业务逻辑** — 所有业务在 `workspace-configs-templates/`
5. **Bundle 绝不包含秘钥** — API Key / 密码禁止序列化
6. **层级即拓扑** — 无手动连线，通信关系从 `parent_id` 派生

### 12.3 相关文档索引

| 文档 | 路径 |
|------|------|
| 系统架构 | [architecture.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/architecture.md) |
| 核心概念 | [core-concepts.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/core-concepts.md) |
| 通信规则 | [communication-rules.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/communication-rules.md) |
| 平台 API | [platform-api.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/platform-api.md) |
| Workspace 运行时 | [workspace-runtime.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/workspace-runtime.md) |
| Canvas UI | [canvas.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/canvas.md) |
| Skills 系统 | [skills.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/skills.md) |
| Bundle 系统 | [bundle-system.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/bundle-system.md) |
| 数据库 Schema | [database-schema.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/database-schema.md) |
| 部署器 | [provisioner.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/provisioner.md) |
| 安全等级 | [workspace-tiers.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/workspace-tiers.md) |
| WebSocket 事件 | [websocket-events.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/websocket-events.md) |
| 可观测性 | [observability.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/observability.md) |
| 构建顺序 | [build-order.md](file:///Users/cuizhanlin/Desktop/Molecule AI-Agent-Team/docs/build-order.md) |

---

> [!NOTE]
> 本 PRD 基于现有 Molecule AI 架构文档编写，覆盖了从技术架构到产品体验的完整产品定义。所有功能需求均与现有代码库中的设计文档对齐，并在此基础上增加了用户旅程、验收标准和商业化路径。

---

*Molecule AI — 点燃你的 AI 组织*
