# AGENTS.md

# Project Overview

This repository is an educational video-sharing platform tailored specifically for junior high school students.

The primary objective of this project is **NOT** to construct a production-grade, internet-scale public video service. Instead, it provides a safe, highly sandboxed, and easily understandable hands-on learning environment for students to explore:

- Frontend UI customization and development
- Backend API structures
- Video upload mechanics
- Local network streaming systems
- Client/server architecture interaction

The entire ecosystem is optimized and designed to run exclusively within a restricted local area network (LAN) classroom environment.

## Main Technology Stack

- **Frontend:** React (v18+) + TypeScript + Vite
- **Backend:** Go + Gin Framework
- **Database:** PostgreSQL
- **Infrastructure:** Docker Compose

---

# Core Philosophy

AI agents modifying this repository must stringently adhere to the following core tenets:

1. **Educational Clarity:** Code must remain simple, flat, and transparent. Avoid deep inheritance, over-generalized abstractions, or advanced metaprogramming that a beginner cannot easily parse.
2. **Safe Editing Experience:** Structural mechanics (networking, file system access, authentication) must be strictly isolated from visual layers so students can freely modify the UI without bricking the core system.
3. **Local-First Architecture:** Network timeouts, asset loading, and server configurations must be tailored for local LAN environments without reliance on external cloud services.
4. **Stable Classroom Operation:** Memory footprint, file handling, and concurrency models must be rock-solid. Do not introduce any code that could cause panic loops or Out-of-Memory (OOM) failures under concurrent student access.
5. **Proactive Protection:** AI agents must actively reject user or script requests that violate the architectural boundaries outlined below.

---

# Critical Design Rules & Boundaries

## 1. Strict Separation of Student and Internal Contexts

Students are permitted to modify visual components, CSS/styling, static text, layouts, and minor UI logic. They must **NEVER** encounter or be forced to modify low-level infrastructure.

- **Editable by Students:** `src/components/student/`, `src/styles/`
- **Protected from Students:** Everything else, including API callers, routers, state management engines, and backend logic.

## 2. Mandatory API Abstraction

Frontend visual components must never initiate raw network transactions, use `fetch` directly, or manage raw multi-part form uploads. All communication networks must be completely wrapped inside clean, deterministic custom hooks or service abstractions.

- **Allowed Interaction:** Students invoke a pre-built hook like `const { uploadVideo } = useVideoUpload();` which abstracts away endpoint handling, headers, error interception, and progress calculation.
- **Prohibited:** Writing raw axios/fetch requests inside student-facing UI layers.

## 3. Mandatory Memory Efficiency (Strict OOM Prevention)

The Go backend handles video uploads and streaming requests concurrently over a local network. Memory management must remain absolute and flat.

- **CRITICAL FORBIDDEN PATTERNS:**
  - `io.ReadAll(c.Request.Body)` or `ioutil.ReadAll` for file streams.
  - `os.ReadFile(filePath)` to buffer video content entirely into RAM.
  - Allocation of byte buffers (`make([]byte, size)`) proportional to the total video size.
- **MANDATORY ENFORCED PATTERNS:**
  - Standard linear stream copying via `io.Copy` or `io.CopyBuffer` using bounded, fixed-size buffers (e.g., 32KB).
  - Explicit utilization of HTTP Range Requests (`http.ServeContent` or precise custom range slicing) to optimize chunk delivery to the frontend browser video player.
  - Immediate flushing and resource closure (`defer file.Close()`) on network connection truncation.

## 4. Double Mode Local Architecture Execution

The infrastructure must seamlessly support two parallel execution methodologies without cross-contamination:

1. Pure Local Native Native Execution (`npm run dev` + `go run ./cmd/api`)
2. Orchestrated Container Execution (`docker compose up`)

Do not introduce configuration mechanisms that break native local execution (e.g., rigid hardcoded internal Docker DNS routes without environment variable fallback loops).

---

# Concrete Directory Architecture & Enforcements

```txt
.
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── student/       <-- [EDITABLE] Only presentation layers, colors, layout variations
│   │   │   └── internal/      <-- [PROTECTED] Player engines, network fallback triggers, UI scaffolding
│   │   ├── pages/             <-- Routing views, structural layout assemblies
│   │   ├── services/          <-- [PROTECTED] Clean API client bindings, concrete HTTP implementations
│   │   ├── hooks/             <-- [PROTECTED] Reusable state bridges and operational abstraction wrappers
│   │   ├── api/               <-- [PROTECTED] Core endpoints, configurations, Axios/Fetch instances
│   │   ├── styles/            <-- [EDITABLE] Tailwind config, CSS entry points, standard variables
│   │   └── types/             <-- System-wide data contracts
│   └── package.json
└── server/
    ├── cmd/                   <-- Main entry points
    │   └── api/               <-- main.go location
    ├── internal/              <-- Core backend logic (Protected)
    │   ├── handler/           <-- HTTP router bindings, context translation, input validation
    │   ├── service/           <-- Core domain logic, validation engines
    │   ├── repository/        <-- PostgreSQL access layers, query parameters
    │   ├── streaming/         <-- Range calculators, segmented reader layers
    │   └── storage/           <-- Discrete disk storage and naming protocols
    ├── uploads/               <-- Local gitignored directory for raw binary storage
    ├── migrations/            <-- Deterministic up/down SQL schema definitions
    ├── go.mod
    └── go.sum
    ---

# Code Generation Rules & Guardrails

### Frontend (React + TypeScript)
- Use strict, deterministic functional components defined with explicit TypeScript interfaces for all `Props`.
- Do not introduce complicated React Context trees or state engines (e.g., Redux Toolkit, Zustand) inside the student domain unless explicitly isolated in internal layers.
- Fast Feedback Loop Priority: Keep dependency graphs thin to ensure Vite HMR (Hot Module Replacement) performs updates within `< 500ms`.

### Backend (Go)
- Strictly execute Layered Architecture. Do not allow SQL statements to bleed into Handlers; do not allow HTTP context headers (`gin.Context`) to bleed into Repositories.
- Avoid global mutable states or race-condition-prone variables. State must reside deterministically inside initialized database engines or context structs.
- Ensure proper logging of structural execution failures on the server terminal without swallowing error codes, while keeping client-facing messages clear and readable.

---

# AI Agent Automation Directives

When modifying this repository, you must execute inside these guardrails:
1. **Deny Feature Creep:** If asked to add intricate enterprise layers (e.g., JWT rotation systems, AWS S3 storage bridges, complex microservices), remind the user that this system is tailored for a local classroom LAN, and preserve monolithic simplicity.
2. **Explainability Metric:** Any code refactoring or function addition you execute must be structure-clean enough to be explained to a high-school or junior-high-school student during a code walkthrough.
3. **Preserve Boundaries:** If a modification request would inadvertently break the segregation of `student` and `internal` subdirectories, restructure the implementation to guarantee that structural code stays within the protected layer.
```
