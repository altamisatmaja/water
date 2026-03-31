# Water: Project Kickoff Guide

**Status**: Ready to Build  
**Structure**: Agent-based development (multi-developer coordination)  
**Entry Point**: This guide  

---

## 🎯 Quick Start (5 Minutes)

### 1. Read This (Right Now)
You're reading it! ✅

### 2. Read Core Docs (15 min)
```bash
# In priority order:
cat CLAUDE.md                          # Start here - project overview
cat agents/README.md                   # Understand agent roles
cat WATER_TECHNICAL_SPEC.md           # Full architecture (skim)
cat WATER_GO_IMPLEMENTATION_GUIDE.md  # Backend start (your role)
```

### 3. Understand Your Role (5 min)
Check `agents/[your-role].md`:
- `backend-agent.md` — Go developer
- `frontend-agent.md` — Svelte developer
- `devops-agent.md` — DevOps engineer
- `schema-agent.md` — Database/architecture
- `orchestrator.md` — Project manager/coordination

### 4. Start Your First Task
See "Week 1 Tasks by Role" below

---

## 📁 Project Structure Explained

```
water/
├── CLAUDE.md                          ← Project overview (start here)
├── WATER_TECHNICAL_SPEC.md           ← Full architecture
├── WATER_GO_IMPLEMENTATION_GUIDE.md   ← Backend guide
├── README.md                          ← User-facing docs (later)
│
├── agents/                            ← Role-based coordination
│   ├── README.md                      ← How agents work
│   ├── backend-agent.md               ← Go developer playbook
│   ├── frontend-agent.md              ← Svelte developer playbook
│   ├── devops-agent.md                ← DevOps playbook
│   ├── schema-agent.md                ← Database design playbook
│   └── orchestrator.md                ← PM coordination playbook
│
├── skills/                            ← Skill reference & tutorials
│   ├── go-cobra-cli.md                ← Cobra CLI tutorial
│   ├── duckdb-go.md                   ← DuckDB Go integration
│   ├── event-capture.md               ← Event streaming design
│   ├── svelte-cytoscape.md            ← Svelte + Cytoscape tutorial
│   ├── cross-compile.md               ← Build & distribution
│   └── README.md                      ← Skills index
│
├── assets/                            ← Project assets
│   └── water-icon.svg
│
├── claude_desktop_config.json         ← For Claude in desktop (multi-agent)
├── server.py                          ← Local dev server (optional)
│
└── [source code will go here]
    ├── cmd/water/
    ├── internal/
    ├── web/
    └── ...
```

---

## 👥 Agent Structure (How This Works)

This project uses **agent-based development** = each developer is a specialized "agent":

### Backend Agent 🔧
- **Role**: Go backend (Cobra CLI, DuckDB, REST API)
- **Entry**: `agents/backend-agent.md`
- **Reference**: `WATER_GO_IMPLEMENTATION_GUIDE.md`
- **Skill**: `skills/go-cobra-cli.md`, `skills/duckdb-go.md`

### Frontend Agent 🎨
- **Role**: Svelte frontend (components, stores, Cytoscape)
- **Entry**: `agents/frontend-agent.md`
- **Reference**: `WATER_SVELTE_FRONTEND_GUIDE.md` (in outputs/)
- **Skill**: `skills/svelte-cytoscape.md`

### DevOps Agent 🚀
- **Role**: Builds, GitHub Actions, Homebrew
- **Entry**: `agents/devops-agent.md`
- **Reference**: `WATER_GITHUB_LAUNCH.md`
- **Skill**: `skills/cross-compile.md`

### Schema Agent 📊
- **Role**: Database design, event schema, API contracts
- **Entry**: `agents/schema-agent.md`
- **Reference**: `WATER_TECHNICAL_SPEC.md` section 3
- **Skill**: `skills/event-capture.md`, `skills/duckdb-go.md`

### Orchestrator Agent 📋
- **Role**: Project coordination, PRs, milestones
- **Entry**: `agents/orchestrator.md`
- **Meetings**: Monday standup, Friday demo

---

## 🚀 Week 1: Getting Started

### Day 1: Setup & Understanding

#### All Team Members (30 min)
```bash
# 1. Clone the repo
git clone https://github.com/water-viz/water.git
cd water

# 2. Read project overview
cat CLAUDE.md

# 3. Understand your role
cat agents/[your-role].md

# 4. Understand structure
cat agents/README.md
```

#### Your Role-Specific Setup (30 min)
- **Backend**: Follow `agents/backend-agent.md` → "Day 1 Setup"
- **Frontend**: Follow `agents/frontend-agent.md` → "Day 1 Setup"
- **DevOps**: Follow `agents/devops-agent.md` → "Day 1 Setup"
- **Schema**: Follow `agents/schema-agent.md` → "Day 1 Setup"

### Day 2: First Implementation

#### Backend Agent
- [ ] Initialize Go module: `go mod init github.com/water-viz/water`
- [ ] Create directory structure
- [ ] Implement `internal/config/config.go`
- [ ] Implement `internal/logger/logger.go`
- [ ] Start on `internal/capture/event.go`

#### Frontend Agent
- [ ] Initialize Svelte: `npm create vite@latest web -- --template svelte`
- [ ] Install dependencies
- [ ] Create store structure (`src/stores/`)
- [ ] Start on `src/stores/graph.ts`

#### DevOps Agent
- [ ] Set up GitHub Actions directory (`.github/workflows/`)
- [ ] Create GitHub Actions workflow stubs
- [ ] Set up cross-compilation script
- [ ] Test builds locally

#### Schema Agent
- [ ] Review database schema in WATER_TECHNICAL_SPEC.md
- [ ] Create DuckDB schema SQL file
- [ ] Define API endpoint contracts
- [ ] Document event schema

### Days 3-5: Implementation Sprint

#### Backend
- Complete `internal/capture/event.go` and `stream.go`
- Implement `internal/graph/client.go` with schema
- Start on CLI commands: `cmd/water/init.go`

#### Frontend
- Complete Svelte stores
- Stub out components
- Set up Tailwind CSS
- Test store-to-component binding

#### DevOps
- Get CI/CD pipelines working (tests, lint)
- Set up cross-compilation
- Create build artifacts

#### Schema
- Finalize SQL schema
- Document API endpoints
- Create test data fixtures

### Week 1 Demo (Friday)
- Backend: `water init` command working (creates .water/ folder)
- Frontend: Stores initialized, can fetch from API
- DevOps: GitHub Actions passing tests
- Schema: All tables created in DuckDB, test data loaded

---

## 📞 How to Coordinate (Agent Communication)

### Daily Standup (15 min)
**Time**: 10 AM every weekday  
**Format**: Slack or in-person
```
[Backend Agent]
✅ Done: Implemented config.go and logger.go
🔄 Working on: event.go event streaming
⚠️ Blocked: Need schema finalized for event struct
📅 Today: Complete event.go, start graph/client.go

[Frontend Agent]
✅ Done: Project initialized, store structure created
🔄 Working on: graph.ts store implementation
⚠️ Blocked: Need API endpoint contracts from schema agent
📅 Today: Complete stores, start Graph.svelte component

[... etc ...]
```

### Code Review Process
1. Push to feature branch: `git checkout -b feature/my-feature`
2. Create Pull Request
3. Request reviews from relevant agents
4. Orchestrator merges when approved

### Blockers
- Post in Slack `#water-blockers` channel
- Assign to blocking agent
- Escalate to Orchestrator if critical

### Meetings
- **Monday 10 AM**: Sprint planning + standup
- **Wednesday 10 AM**: Standup (check progress)
- **Friday 4 PM**: Demo + retrospective

---

## 📚 How to Find Information

### "How do I implement X?"
1. Check your agent role: `agents/[your-role].md`
2. Check skill guide: `skills/[relevant-skill].md`
3. Check technical spec: `WATER_TECHNICAL_SPEC.md`
4. Check implementation guide: `WATER_GO_IMPLEMENTATION_GUIDE.md` or Svelte guide

### "What's the event schema?"
→ `WATER_TECHNICAL_SPEC.md` section 3 + `skills/event-capture.md`

### "How do I build Go code?"
→ `skills/go-cobra-cli.md`

### "How do I create Svelte components?"
→ `skills/svelte-cytoscape.md`

### "How do I set up GitHub Actions?"
→ `WATER_GITHUB_LAUNCH.md` + `skills/cross-compile.md`

### "What's my first task?"
→ `agents/[your-role].md` → "Week 1 Tasks"

---

## ✅ Week 1 Checklist

### All Team Members
- [ ] Read CLAUDE.md
- [ ] Read agents/README.md
- [ ] Read your role agent file
- [ ] Set up dev environment
- [ ] First PR submitted
- [ ] Attended standup meetings

### Backend Agent
- [ ] Go module initialized
- [ ] Directory structure created
- [ ] config.go + logger.go complete
- [ ] event.go started
- [ ] Tests passing

### Frontend Agent
- [ ] Svelte project initialized
- [ ] Dependencies installed
- [ ] Stores created
- [ ] Component structure stubbed
- [ ] Can run `npm run dev`

### DevOps Agent
- [ ] GitHub Actions workflows created
- [ ] CI/CD pipeline working
- [ ] Cross-compilation tested
- [ ] Build artifacts generated

### Schema Agent
- [ ] Database schema finalized
- [ ] Event schema documented
- [ ] API contracts defined
- [ ] Test fixtures created

### Orchestrator
- [ ] GitHub repo configured (branch protection, status checks)
- [ ] Team roles assigned
- [ ] Standup schedule set
- [ ] Blockers tracked

---

## 🎯 First 3 Weeks (MVP)

### Week 1: Foundation
**Goal**: All modules initialized, basic structure working

**Backend**: 
- ✅ config, logger, capture, graph/client modules
- ✅ `water init` command (creates .water/)
- ✅ DuckDB schema working

**Frontend**:
- ✅ Stores (graph, filters, ui)
- ✅ Component stubs
- ✅ Can fetch from backend API

**DevOps**:
- ✅ CI/CD pipeline
- ✅ Cross-platform builds
- ✅ Test framework

**Output**: `water -init` command works ✅

---

### Week 2: Integration
**Goal**: Backend + Frontend connected, basic visualization

**Backend**:
- ✅ `water serve` command (starts HTTP server)
- ✅ API endpoints: `/api/nodes`, `/api/edges`
- ✅ Event capture working (append-only JSON)

**Frontend**:
- ✅ Graph.svelte (Cytoscape visualization)
- ✅ Timeline.svelte
- ✅ Metrics.svelte
- ✅ Connected to backend API

**DevOps**:
- ✅ Embed Svelte build in Go binary
- ✅ Single binary builds

**Output**: `water serve` launches dashboard, can visualize nodes + edges ✅

---

### Week 3: Polish & Launch
**Goal**: Documentation, testing, first release

**All Agents**:
- ✅ README.md complete
- ✅ Tests passing (unit + integration)
- ✅ Code documented
- ✅ GitHub workflows passing

**DevOps**:
- ✅ Homebrew formula working
- ✅ Cross-platform binaries
- ✅ GitHub release created

**Orchestrator**:
- ✅ Beta tester program (10+ users)
- ✅ v0.1.0-alpha released

**Output**: `brew install water` works ✅, 50+ GitHub stars ⭐

---

## 🚨 Common Issues & Solutions

### "I don't understand my role"
→ Read `agents/[your-role].md` + the skill guides it references

### "Backend and frontend aren't in sync on API contracts"
→ Schema Agent mediates. Define endpoints first in WATER_TECHNICAL_SPEC.md, then implement

### "Merge conflicts"
→ Work on separate features. Merge daily to `dev` branch

### "GitHub Actions failing"
→ DevOps Agent checks logs. Post blockers in Slack

### "Need Svelte help"
→ Check `skills/svelte-cytoscape.md` or ask Frontend Agent in standup

### "Blocked on something"
→ Post in Slack #water-blockers with @relevant-agent tag

---

## 📅 Timeline Checkpoint

| Week | Goal | Status |
|------|------|--------|
| **Week 1** | Foundation (modules, setup, init command) | 📅 Mar 31 - Apr 6 |
| **Week 2** | Integration (serve, API, dashboard) | 📅 Apr 7 - Apr 13 |
| **Week 3** | Polish (docs, tests, v0.1.0-alpha release) | 📅 Apr 14 - Apr 20 |
| **Week 4-6** | Phase 2 (embeddings, clustering, analytics) | 📅 Apr 21 - May 11 |
| **Week 7-8** | Polish & Ship | 📅 May 12 - May 25 |

---

## 🎓 Resources

### For Backend Agent
- Go docs: https://go.dev
- Cobra tutorial: https://cobra.dev
- DuckDB Go: https://github.com/marcboeker/go-duckdb

### For Frontend Agent
- Svelte tutorial: https://svelte.dev/tutorial
- Cytoscape.js: https://js.cytoscape.org
- Tailwind CSS: https://tailwindcss.com

### For DevOps Agent
- GitHub Actions: https://github.com/features/actions
- Homebrew: https://docs.brew.sh

### For Schema Agent
- Database design: Check WATER_TECHNICAL_SPEC.md section 3
- Event schema: Check WATER_TECHNICAL_SPEC.md + skills/event-capture.md

---

## 🎉 Ready?

1. **Read your agent file** (`agents/[your-role].md`)
2. **Do your Day 1 setup** (in the agent file)
3. **Show up to Monday standup**
4. **Submit your first PR by Friday**

That's it. Let's ship Water! 🚀

---

## Next Actions

### Right Now (Next 10 min)
```bash
# Read your role
cat agents/[backend|frontend|devops|schema|orchestrator].md

# Understand the tech stack
cat WATER_TECHNICAL_SPEC.md | head -100
```

### Today
```bash
# Do your Day 1 setup from your agent file
# Initialize your environment
# First commit: "initial project structure"
```

### This Week
- Implement first module (from your agent file)
- Submit first PR
- Attend standup meetings
- Help teammates where blocked

---

**Questions? Ask your Orchestrator or the relevant agent. Everything else is in the docs.** 💪

**Let's build Water!** 🧠💧