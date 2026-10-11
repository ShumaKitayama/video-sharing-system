# AGENTS.md

# Project Overview

This repository is an educational video-sharing platform tailored specifically for junior high school students.

The primary objective of this project is **NOT** to construct a production-grade, internet-scale public video service. Instead, it provides a safe, highly sandboxed, and easily understandable hands-on learning environment for students to explore:

- Frontend UI customization and development
- Backend API structures
- Video upload mechanics
- Local network streaming systems
- Client/server architecture interaction

The ecosystem is scoped to a single classroom. It is delivered from the project's own Vercel deployment so that hands-on sessions need **nothing installed on student machines** (Windows + Chrome only): students open `/lab` in the browser, edit the student layer through fill-in-the-blank steps or a free editor, and see the result in a live preview of the real site. Native local execution (`npm run dev`, `go run`, Docker) remains supported for teachers and developers.

## Main Technology Stack

- **Frontend:** React (v18+) + TypeScript + Vite
- **Backend:** Go + Gin Framework
- **Database:** PostgreSQL
- **Infrastructure:** Docker Compose (local), Vercel (classroom delivery: static frontend + Go API, Neon, Vercel Blob)
- **Browser Lab:** Sucrase (in-browser TSX → JS), CodeMirror 6 (free editor) — loaded only on `/lab`

---

# Core Philosophy

AI agents modifying this repository must stringently adhere to the following core tenets:

1. **Educational Clarity:** Code must remain simple, flat, and transparent. Avoid deep inheritance, over-generalized abstractions, or advanced metaprogramming that a beginner cannot easily parse.
2. **Safe Editing Experience:** Structural mechanics (networking, file system access, authentication) must be strictly isolated from visual layers so students can freely modify the UI without bricking the core system.
3. **Zero-Install Classroom:** A student needs only Chrome and a URL. Never add steps that require installing tools, copying files, or configuring student machines. Do not add third-party services or CDNs at runtime beyond the project's own Vercel deployment; keep everything bundled.
4. **Stable Classroom Operation:** Memory footprint, file handling, and concurrency models must be rock-solid. Do not introduce any code that could cause panic loops or Out-of-Memory (OOM) failures under concurrent student access.
5. **Proactive Protection:** AI agents must actively reject user or script requests that violate the architectural boundaries outlined below.

---

# Critical Design Rules & Boundaries

## 1. Strict Separation of Student and Internal Contexts

Students are permitted to modify visual components, CSS/styling, static text, layouts, and minor UI logic. They must **NEVER** encounter or be forced to modify low-level infrastructure.

- **Editable by Students:** `front/src/components/student/`, `front/src/styles/` (in class, edited through `/lab` in the browser — the files themselves are not modified)
- **Editable by Teachers:** `front/src/lab/lessons/steps.ts` (lesson data only)
- **Protected from Students:** Everything else, including API callers, routers, state management engines, the Lab runtime, and backend logic.

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

The browser Lab (`/lab`) is served by the same frontend build and must keep working in all modes; it adds no extra server.

## 5. Browser Lab Boundaries

Student code from `/lab` is compiled **in the student's own browser** and only swaps the student layer inside a same-origin preview iframe.

- Pages must use student components through `components/internal/StudentSlots.tsx` and `components/internal/studentFormatters.ts`, never by importing `components/student/` directly. The slots wrap every student component in an error boundary that falls back to the built-in component.
- Student modules may only import what `lab/runtime/moduleLoader.ts` (`hostModules`) exposes. Add to that allow-list deliberately; never expose `services/`, `hooks/` internals, `fetch`, or storage helpers.
- `lab/runtime/compileStudent.ts` rejects `fetch` / `XMLHttpRequest` / `axios` / `eval` in student code, mirroring Rule 2.
- Keep the Lab heavy dependencies (Sucrase, CodeMirror) behind dynamic imports so the normal site bundle does not grow.
- Student code is saved only in that browser's `localStorage`. Do not add endpoints that store or execute student code on the server.
- Lesson `find` strings must match the real files exactly; when you change a file in `components/student/` or `styles/index.css`, update `lab/lessons/steps.ts` accordingly.

---

# Concrete Directory Architecture & Enforcements

```txt
.
├── front/
│   ├── src/
│   │   ├── components/
│   │   │   ├── student/       <-- [EDITABLE] Only presentation layers, colors, layout variations
│   │   │   └── internal/      <-- [PROTECTED] Player engines, UI scaffolding, StudentSlots (student swap points)
│   │   ├── pages/             <-- Routing views, structural layout assemblies
│   │   ├── services/          <-- [PROTECTED] Clean API client bindings, concrete HTTP implementations
│   │   ├── hooks/             <-- [PROTECTED] Reusable state bridges and operational abstraction wrappers
│   │   ├── api/               <-- [PROTECTED] Core endpoints, configurations, Axios/Fetch instances
│   │   ├── styles/            <-- [EDITABLE] Tailwind config, CSS entry points, standard variables
│   │   ├── lab/               <-- [PROTECTED] Browser Lab (/lab): runtime/, components/, labState.ts
│   │   │   └── lessons/       <-- [TEACHER-EDITABLE] steps.ts (fill-in-the-blank lesson data)
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
1. **Deny Feature Creep:** If asked to add intricate enterprise layers (e.g., JWT rotation systems, AWS S3 storage bridges, complex microservices), remind the user that this system is tailored for a single classroom, and preserve monolithic simplicity.
2. **Explainability Metric:** Any code refactoring or function addition you execute must be structure-clean enough to be explained to a high-school or junior-high-school student during a code walkthrough.
3. **Preserve Boundaries:** If a modification request would inadvertently break the segregation of `student` and `internal` subdirectories, restructure the implementation to guarantee that structural code stays within the protected layer.
```
