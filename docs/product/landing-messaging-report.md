# Molecule AI Landing Messaging Report

Last updated: 2026-04-09

## 1. Executive Summary

基于当前 `main` 分支，Molecule AI 最适合对外讲的，不是“又一个 agent framework”，也不是“又一个 workflow builder”，而是一个更高层、更接近生产系统的类别：

> **Molecule AI is the org-native control plane for heterogeneous AI agent teams.**

它解决的核心问题不是“单个 agent 怎么更聪明”，而是当企业开始真正运行一支 AI 团队时，如何把这些 agent 组织起来、治理起来、观察起来、恢复起来，并且允许不同 runtime 在同一个组织系统里协作。

从当前仓库能被严格支持的叙事看，Molecule AI 已经具备以下清晰卖点：

1. **Workspace 是角色，不是任务节点**
2. **组织结构本身就是协作拓扑**
3. **不同 agent runtime 可以共存于同一控制平面**
4. **记忆边界沿组织边界流动，而不是全局混写**
5. **平台具备真实 control plane 能力，而不是停留在 demo orchestration**
6. **系统已经开始形成 memory -> skill -> operational improvement 的复利闭环**

因此，landing page 的主叙事应该从“agent 很强”转向“AI 团队可被设计、治理、扩张、运行和恢复”。

---

## 2. 基于最新版本的类别定义

## 2.1 一句话定义

Molecule AI 是一个面向 **heterogeneous AI agent teams** 的 **org-native control plane**。

更直白一点：

- 它不是只负责 prompt 编排
- 不是只负责某一种 agent runtime
- 也不是只负责画流程图

它负责把一整支 AI 团队作为一个可运行、可治理、可扩张的组织系统来管理。

## 2.2 它填补的类别空白

当前市场里大多数产品大致分成四类：

1. **聊天型 AI 产品**
   - 强在单用户交互
   - 弱在组织结构、治理、运行边界

2. **workflow builders**
   - 强在任务流程编排
   - 节点通常代表 task / API / tool
   - 弱在角色抽象、长期团队形态、组织治理

3. **agent frameworks**
   - 强在 agent loop、tool use、planning
   - 弱在 control plane、生命周期治理、跨团队运营

4. **coding agents / CLI agents**
   - 强在真实执行
   - 弱在团队组织层、层级协作、统一运维面

Molecule AI 的定位更像：

> **The missing operational and organizational layer above agent runtimes.**

这使它天然适合被定义为：

- AI 团队控制平面
- AI 组织操作层
- heterogeneous runtimes 的统一治理层

## 2.3 我们真正卖的不是 agent，而是组织能力

Molecule AI 对外卖的不是“一个更强的 agent”，而是：

- 一种构建 AI 组织的方法
- 一套治理 AI 团队的控制平面
- 一个允许多种 runtime 在统一规则下共存的组织层
- 一个让 AI 从 demo 走向 production operations 的平台底座

---

## 3. 当前 `main` 可以明确宣传的产品事实

这一部分只写当前主分支有文档支撑、可以安全对外表达的内容。

## 3.1 Workspace 是角色容器，不是任务节点

当前产品最核心的抽象是 **workspace**。

在 Molecule AI 中，一个 workspace 同时是：

- 一个组织角色
- 一个 agent runtime 容器
- 一个带 Agent Card 的 A2A 服务端点
- 一个可以递归扩展为团队的组织单元

这意味着用户在画布上搭建的不是 workflow DAG，而是 AI 组织图。

这个抽象带来的对外价值非常强：

- 模型可换，但角色不变
- runtime 可换，但角色身份不变
- 单 agent 可以扩展成团队，但对外接口不变

适合 landing page 的表达是：

> Start with one agent. Expand into a team. Keep the same organizational identity.

## 3.2 组织图就是拓扑

Molecule AI 不是通过手动画边来表达协作关系。当前系统的默认协作表面由 hierarchy 决定：

- parent -> child 可以委派
- child -> parent 可以汇报
- siblings 可以协作
- 团队外不能直接访问私有子工作区

这意味着组织图不是“装饰性 UI”，而是系统运行逻辑的一部分。

对外讲法可以明确到：

> The org chart is the topology.

这句话在当前版本是成立的，因为通信边界、team expansion、discoverability 和 private scope 都围绕层级关系实现。

## 3.3 当前 `main` 已经形成 heterogeneous runtime 叙事

当前 `main` 已合并并文档化的 runtime surface 是 6 个 adapter：

- LangGraph
- DeepAgents
- Claude Code
- CrewAI
- AutoGen
- OpenClaw

这里需要特别注意边界：

- **NemoClaw 目前不是 `main` 已合并能力**
- 它只应当被视为分支级 WIP / roadmap，不应写成 current product proof

因此当前最准确的对外表达是：

> Standardize governance without standardizing runtimes.

这也是 Molecule AI 一个极强的对外差异点。因为它不要求用户放弃底层 runtime 选择，只要求团队把治理和组织标准提升到上层。

## 3.4 HMA 已经是可以成立的深度概念

当前版本的 memory 叙事，不应再写成泛泛的“agent memory”。

最新文档显示，Molecule AI 的记忆模型已经明确区分：

- `LOCAL`
- `TEAM`
- `GLOBAL`

并且当前实现里存在多类 memory surface：

1. `agent_memories`
   - 面向 HMA 的 durable scoped memory

2. `workspace_memory`
   - 适合 UI 配置和结构化状态的 key/value memory

3. `session-search`
   - 最近活动与记忆回溯

4. awareness-backed persistence
   - 当 awareness 配置存在时，memory 会进入 workspace-scoped namespace

所以现在适合宣传的不是“记得更多”，而是：

> Memory is treated like infrastructure, not a flat vector dump.

这句话与当前 `main` 是匹配的。

## 3.5 Skill evolution 是 Molecule AI 的复利点

当前 README 和 runtime docs 已经把以下路径讲清楚：

1. 任务执行沉淀 durable insight
2. 重复成功形成信号
3. 经验被提升为 reusable skill
4. skill hot-reload 回 runtime

这意味着 Molecule AI 的能力叙事不只是 memory，而是 memory 和 skills 的协同：

- memory 用于存事实、上下文、长期知识
- skills 用于存可复用 procedure

这是一个比“memory feature”更有平台感的产品点，因为它暗示系统能够把团队经验转化为可运行能力。

## 3.6 当前平台已经具备真实 control plane 轮廓

根据最新 README、canvas、quickstart 和 edit history，当前 `main` 已经可以对外讲这些 control plane 能力：

- workspace CRUD 与 provisioning
- registry + heartbeat
- pause / resume / restart
- health sweep + auto-restart
- activity logs
- current task reporting
- Agent Card refresh
- WebSocket fanout
- browser-safe A2A proxy
- terminal access
- files access
- traces
- templates
- bundles
- global secrets with workspace override

这意味着 Molecule AI 不是只会“把 agent 放在画布上”，而是已经在形成一个真正的运营面板。

## 3.7 当前 canvas 已是运营 UI，而不是展示 UI

最新版本的 canvas 文档和 quickstart 已经把这一点坐实：

- 空画布部署入口
- onboarding wizard
- drag-to-nest team building
- 10-tab side panel
- WebSocket-first chat response delivery
- hydration retry
- app-wide error boundary

所以 landing page 可以明确表达：

> Molecule AI is not just a visualizer. It is the operational UI for AI teams.

## 3.8 Global secrets 是企业化落地的重要卖点

最新主分支已经有：

- platform-wide secrets
- workspace-level overrides
- Config UI 中可视化 scope

这使得对外可以讲：

- 企业不需要给每个 workspace 手动重复配置 provider key
- 平台可以集中管理基础凭证
- 特殊角色仍可局部覆盖

这是一个非常适合技术负责人和平台团队的现实卖点。

## 3.9 Runtime tiers 可以作为治理与风险分级叙事

当前 `workspace-tiers.md` 文档是 4-tier 模型：

- T1 Sandboxed
- T2 Standard
- T3 Privileged
- T4 Full Access

它最适合用于表达“不同角色拥有不同执行权限与风险边界”，而不是泛泛地讲安全。

更好的对外表述是：

- 所有 agent 不应该在同一权限层运行
- AI 团队内部也应有风险分级
- 执行能力应该与角色责任匹配

---

## 4. 产品哲学与理念

## 4.1 角色比任务更稳定

Molecule AI 的核心抽象不是 task node，而是 role-native workspace。

这是一个很重要的产品哲学判断：

- task 会变
- tool 会变
- model 会变
- runtime 会变
- 但组织中的角色职责更稳定

例如：

- Research Lead
- Developer PM
- QA Engineer
- Marketing Lead
- DevOps

这些更像企业真实组织结构，而不是一次性的 DAG step。

因此，Molecule AI 更适合讲“长期可运行的 AI 组织”，而不是“临时拼装的自动化流程”。

## 4.2 组织边界就是治理边界

在 Molecule AI 的理念里：

- 组织结构决定通信关系
- 组织结构决定 team scope
- 组织结构决定 memory sharing surface
- 组织结构影响 runtime 风险分层

这让治理和组织不再是两套独立配置，而是同一张图的不同投影。

这是 Molecule AI 最强的哲学表达之一：

> Governance is not bolted on later. It is encoded into the organizational model.

## 4.3 Memory 应服从组织边界，而不是追求全局共享

多数 agent 系统会默认“共享越多越好”，但企业现实并不是这样。

真实企业需要的是：

- 正确的人看到正确的信息
- 共享发生在合适的组织边界内
- 全局知识可读，但高风险写入有约束

HMA 在当前产品中的真正价值不是“更强记忆”，而是：

- 组织隔离
- 协作 handoff
- 结构化 recall
- 治理可解释性

## 4.4 Agent 需要被运行和治理，而不是被神化

Molecule AI 的整体产品气质不是“全自动 AI 乌托邦”。

它更接近：

- 可部署
- 可观察
- 可恢复
- 可暂停
- 可检查
- 可约束

这使 Molecule AI 更像企业级 operating layer，而不是 consumer AI assistant。

---

## 5. 技术差异化优势

## 5.1 相比 workflow builders，Molecule AI 的节点语义完全不同

传统 workflow builders 通常是：

- 节点 = task / tool / API
- 核心问题 = 执行顺序与分支逻辑

Molecule AI 是：

- 节点 = 组织角色 / workspace
- 核心问题 = hierarchy、governance、lifecycle、team structure

所以它并不是“更复杂的流程图工具”，而是另一种系统抽象。

## 5.2 相比 agent frameworks，Molecule AI 不打 agent loop 正面战

LangGraph、CrewAI、AutoGen、DeepAgents 等的价值主要在：

- reasoning / planning
- tool use
- runtime semantics
- collaboration primitives

Molecule AI 不需要和它们在这一层竞争。它的定位是：

> The operational and organizational layer above heterogeneous agent runtimes.

这使得它可以吸纳 runtime 生态，而不是被 runtime 生态替代。

## 5.3 相比 coding agents，Molecule AI 把单兵能力升级为团队基础设施

Claude Code 这类运行时擅长真实执行，但单独使用时更像个人 agent。

Molecule AI 带来的额外价值是：

- workspace identity
- hierarchy-aware collaboration
- A2A delegation
- shared control plane
- memory scopes
- operational lifecycle

换句话说，Molecule AI 把优秀的单兵 runtime 变成可编排、可治理的团队成员。

## 5.4 Recursive team expansion 是非常强的结构性优势

当前 team expansion 机制具备非常强的产品表达力：

- 一个 workspace 可以扩展成内部团队
- 对外仍然保持同一个角色接口
- team lead 作为唯一外部桥接面
- 子团队在内部递归协作，对外保持封装

这非常接近现实组织的扩张方式，也是平台未来模板化和 bundle 化的基础。

## 5.5 Awareness namespace 让 memory boundary 从理念进入实现

过去讲 HMA 很容易被理解成架构概念。现在最新版本已经更具体：

- runtime 工具接口稳定
- awareness 开启后，memory 进入 workspace namespace
- 没有 awareness 时也能保持兼容回退

这说明 Molecule AI 正在把“组织边界内的 memory”从理念做成稳定实现路径。

## 5.6 WebSocket-first operational UX 提升了“真实系统”感

当前系统已经不是“发请求后等刷新页面”这种 demo 交互。

现在已经形成：

- WebSocket-first A2A response delivery
- current task 实时更新
- AGENT_CARD_UPDATED 实时刷新
- AGENT_MESSAGE 主动推送
- error boundary + hydration retry

这使 landing page 可以更有底气地讲：

> Molecule AI is built to operate live systems, not static demos.

---

## 6. 商业差异化与市场价值

## 6.1 我们卖的是组织能力，不是 prompt 技巧

很多 AI 产品卖的是：

- 模型效果
- prompt 包装
- 单任务效率

Molecule AI 卖的是：

- 企业如何拥有一支 AI 团队
- 如何让不同 agent 作为真实角色协作
- 如何在组织边界内给 AI 放权
- 如何把 agent 从实验对象变成可治理资产

这使它天然更适合：

- CTO / AI platform 团队
- 内部自动化平台
- 需要长期运行 agent 组织的公司
- 希望同时支持多种 runtime 的技术组织

## 6.2 平台属性比单点功能更强

workflow 工具容易被新节点替代。  
聊天产品容易被模型替代。  
单 agent 产品容易被更强 agent 替代。

但 Molecule AI 绑定的是更高层的东西：

- 组织结构
- runtime 治理
- memory boundary
- lifecycle operations
- templates / bundles / reusable team patterns

一旦进入企业内部流程，它更接近基础设施，而不是单点功能。

## 6.3 异构 runtime 兼容提升了平台议价能力

如果平台要求所有团队都迁移到同一种 runtime，企业 adoption 会很慢。

Molecule AI 的商业价值恰恰在于：

- 不强制 runtime 统一
- 允许团队保留底层偏好
- 只要求在 governance 和 operations 层达成统一

这会显著降低 adoption friction。

## 6.4 Bundles、templates、skills 为未来产品化打开空间

当前版本已经有：

- templates
- bundle import/export
- skills hot reload

这意味着未来非常自然的商业路径包括：

- 行业 Bot Team / Agent Team 模板
- 可复用组织能力包
- 团队级最佳实践分发
- 面向企业的平台增值能力

即使当前还不应该把 marketplace 当成“已上线能力”去宣传，它也已经是非常自然的 next layer。

---

## 7. Why Now：为什么现在是这个类别成立的时点

这一部分是 landing page 很值得强化的融资叙事。

今天行业已经不缺：

- 单个强 agent
- workflow automation
- coding agent demo

真正缺的是：

- 让不同 agent 以组织角色存在
- 让它们在边界内协作
- 让它们被统一运营
- 让团队能 live、recover、inspect、govern

随着 agent 开始进入真实工作流，新的瓶颈不再只是模型本身，而是：

- 谁负责什么
- 谁能调谁
- 谁能看什么
- 哪个 agent 能执行高风险动作
- 故障怎么恢复
- 如何把团队经验沉淀为可复用能力

Molecule AI 的类别价值，恰恰诞生在这里。

---

## 8. Landing Page 最值得重点宣传的叙事结构

## 8.1 第一层：类别定义

先说清楚：

- Molecule AI 不是另一个 workflow builder
- 不是另一个单 runtime 框架
- 它是 AI agent teams 的 org-native control plane

目标是抢到类别定义权。

## 8.2 第二层：理念与产品哲学

接着讲：

- the node is a role, not a task
- the org chart is the topology
- memory follows organizational boundaries
- governance is built in, not added later

目标是建立 worldview。

## 8.3 第三层：当前产品 proof

然后给出当前 `main` 能撑住的 proof：

- six runtime adapters on `main`
- HMA-style memory scopes + awareness namespaces
- recursive team expansion
- global secrets with local override
- WebSocket-first operational UX
- restart / pause / resume / health sweep / auto-restart

目标是建立“这不是概念页”的可信度。

## 8.4 第四层：商业与平台价值

再解释为什么这对企业重要：

- heterogeneous runtime teams 需要统一治理
- AI 团队需要控制平面，不只是 prompt layer
- 平台级 adoption 比单 agent feature 更难被替代

## 8.5 第五层：未来愿景

最后才讲更远的 vision：

- terminal agents
- browser agents
- device agents
- embodied systems
- bot teams

这样可以在不夸大现状的前提下把想象空间拉高。

---

## 9. 适合 landing page 的理念表达

## 9.1 品牌级表达

- Build AI organizations, not fragile agent demos.
- The node is a role, not a task.
- The org chart is the topology.
- Standardize governance without standardizing runtimes.
- Memory should follow organizational boundaries.
- Operate live AI teams, not brittle orchestration graphs.

## 9.2 对技术负责人的表达

- 把 agent 从 runtime 选择题，升级成统一治理问题
- 把不同团队的 runtime 差异留在底层，把 control plane 拉到上层统一
- 把 memory 从平面共享改造成组织基础设施
- 把 AI 团队纳入 pause / resume / restart / inspect / trace 的真实运营体系

## 9.3 对企业决策者的表达

- 你不是在部署一堆 bot，而是在设计一支 AI 团队
- 你不需要先统一所有 runtime，才能获得统一治理
- 你可以像管理组织一样管理 AI
- 你可以从一个角色开始，再扩展成一个团队，而不必重建整个系统

---

## 10. 未来愿景：从 Agent Teams 走向 Bot Teams

这一层必须明确是 **vision layer**，不是 current main proof。

## 10.1 为什么这个愿景与当前产品方向一致

Molecule AI 当前已经成立的抽象有几个很关键的前提：

- workspace 是组织角色容器
- runtime 是可替换执行载体
- A2A 是协作接口
- hierarchy 是治理边界
- tiers 是风险分级执行模型

这些前提并不要求执行体一定是“纯软件里的 LLM agent”。

因此，Molecule AI 的未来愿景可以自然延伸为：

> Today, Molecule AI organizes AI agent teams.  
> Tomorrow, it can organize bot teams across software, terminals, devices, and embodied systems.

## 10.2 从 software agents 到 terminal/device executors

未来进入 Molecule AI 组织层的“成员”，可以不只是当前意义上的 agent。

它还可以包括：

- terminal bots
- browser bots
- desktop operation bots
- mobile execution bots
- device-connected agents
- robot or embodied execution systems

Molecule AI 的角色不是替代这些执行体本身，而是给它们提供：

- 组织关系
- 协作边界
- 记忆边界
- 风险分层
- 审计与恢复能力

## 10.3 为什么“Bot Team”是合理延展，而不是空想

这个愿景不是凭空跳跃，原因在于当前产品已经接受了三个关键前提：

1. **异构运行时是前提，不是例外**
2. **对外统一的是 workspace contract，而不是内部实现**
3. **治理层高于 runtime 层**

一旦这些前提成立，未来接入新的执行体类型就是产品边界外扩，而不是范式重写。

## 10.4 通用问题与复杂问题的长期分工

未来非常适合对外讲的愿景结构是：

- 通用问题由通用 Bot Team 自主解决
- 复杂问题由多角色、多执行体、层级化组织协同解决

这会把 Molecule AI 的终局表达，从“多 agent 协作”升级为：

> an organizational layer for autonomous problem-solving systems

中文版可以表达为：

> Molecule AI 正在构建自治问题解决系统的组织层。

---

## 11. 对外表达时需要谨慎处理的边界

## 11.1 NemoClaw 不能写成 current main support

当前主分支只应宣传 6 个 runtime adapter。

NemoClaw 可以出现在 roadmap / future direction / branch-level experimentation，但不能当作已合并能力写到 current proof 里。

## 11.2 Bot / terminal / robot 只能写成愿景层

这些方向可以大胆写，但必须标注成：

- future direction
- natural extension
- long-term platform vision

不能写成当前已经全面落地支持。

## 11.3 不要把 memory 讲成“我们已经有完整企业知识中枢”

当前可以讲的是：

- HMA 思路
- scoped memory
- workspace awareness namespaces
- memory-to-skill promotion path

不宜夸大成已经完成全面企业知识治理平台。

## 11.4 不要把 tiers 讲成“完整合规体系”

当前 runtime tiers 很适合表达风险分级和执行边界，但不应该直接等同于大型企业合规认证能力。

---

## 12. 可直接压缩成 landing 文案的结论

如果把最新版本压缩成最值得对外讲的几句话：

1. Molecule AI 不是一个 workflow builder，而是 heterogeneous AI agent teams 的 org-native control plane。
2. 在 Molecule AI 里，workspace 是角色，不是任务节点；组织图本身就是协作拓扑。
3. Molecule AI 允许 LangGraph、DeepAgents、Claude Code、CrewAI、AutoGen 和 OpenClaw 在同一控制平面下协作。
4. Molecule AI 把 memory 当作组织基础设施来设计，而不是扁平共享上下文。
5. Molecule AI 已经具备运行一支 AI 团队所需的关键 control plane 能力，包括 lifecycle、observability、secrets、WebSocket-first ops 和 team expansion。
6. 长期看，Molecule AI 不只适用于 software agent teams，也天然指向 software, terminal, device, robotics 组成的 bot teams。

---

## 13. 建议对外的品牌终局表达

如果需要一句最能承载平台 ambition 的表达，当前最稳妥的是：

> **Molecule AI is building the organizational layer for autonomous teams.**

如果希望更偏未来愿景，也可以写：

> **From AI agent teams to bot teams, Molecule AI is building the control plane for autonomous problem-solving organizations.**

中文版建议：

> **Molecule AI 正在构建自治型团队的组织层。**

或：

> **从 AI Agent Team 到 Bot Team，Molecule AI 正在构建自治问题解决组织的控制平面。**

---

## 14. Source Basis

This report is aligned to the current `main` branch and grounded primarily in:

- `README.md`
- `docs/index.md`
- `docs/quickstart.md`
- `docs/product/core-concepts.md`
- `docs/architecture/architecture.md`
- `docs/architecture/memory.md`
- `docs/architecture/workspace-tiers.md`
- `docs/agent-runtime/workspace-runtime.md`
- `docs/agent-runtime/cli-runtime.md`
- `docs/agent-runtime/team-expansion.md`
- `docs/frontend/canvas.md`
- `docs/edit-history/2026-04-08.md`
- `docs/edit-history/2026-04-09.md`

This version intentionally separates:

- **current main product claims**
- **strategic narrative inferred from the architecture**
- **forward-looking vision**

so the landing page can be ambitious without blurring the boundary between shipped reality and future direction.
