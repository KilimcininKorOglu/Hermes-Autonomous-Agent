# Hermes v3.0 - Web Interface

## Overview

Hermes v3.0 replaces the terminal-based TUI with a modern web interface, enabling remote access, real-time collaboration, and a richer visual experience for task management and AI execution monitoring.

## Current State (v1.x - v2.x)

```
Terminal TUI (bubbletea)
├── Single user
├── Local access only
├── Text-based UI
├── No persistence of views
└── Limited visualization
```

## Proposed State (v3.0)

```
Web Interface
├── Multi-user support
├── Remote access (browser)
├── Modern React UI
├── Real-time updates (WebSocket)
├── Rich visualizations (graphs, charts)
├── Mobile responsive
└── Session persistence
```

## Key Features

### 1. Dashboard

Real-time overview of project status.

```
┌─────────────────────────────────────────────────────────────────────────┐
│  HERMES                                    user@project    ⚙️  🔔  👤   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   TASKS     │  │  COMPLETED  │  │ IN PROGRESS │  │   BLOCKED   │    │
│  │     12      │  │      5      │  │      3      │  │      1      │    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
│                                                                         │
│  ┌─────────────────────────────────┐  ┌─────────────────────────────┐  │
│  │  PROGRESS                       │  │  ACTIVITY                   │  │
│  │  ████████████░░░░░░░░░░░ 58%   │  │  ┌─┐                         │  │
│  │                                 │  │  │ │ ┌─┐     ┌─┐            │  │
│  │  Feature 1: ████████████ 100%  │  │  │ │ │ │ ┌─┐ │ │            │  │
│  │  Feature 2: ████████░░░░  67%  │  │  └─┴─┴─┴─┴─┴─┴─┘            │  │
│  │  Feature 3: ████░░░░░░░░  33%  │  │  Mon Tue Wed Thu Fri        │  │
│  └─────────────────────────────────┘  └─────────────────────────────┘  │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  RECENT ACTIVITY                                                 │   │
│  │  ● T005 completed - User Authentication API         2 min ago  │   │
│  │  ● T006 started - Frontend Login Component          5 min ago  │   │
│  │  ● Feature 1 completed - Database Layer            15 min ago  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2. Task Board (Kanban)

Drag-and-drop task management.

```
┌─────────────────────────────────────────────────────────────────────────┐
│  TASK BOARD                                    Filter ▼  Search 🔍     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  NOT STARTED      IN PROGRESS       COMPLETED         BLOCKED          │
│  ┌───────────┐    ┌───────────┐    ┌───────────┐    ┌───────────┐     │
│  │ T007      │    │ T005  🔄  │    │ T001  ✓  │    │ T009  ⚠️  │     │
│  │ API Tests │    │ Auth API  │    │ DB Schema │    │ Deploy    │     │
│  │ P2  2d    │    │ P1  3d    │    │ P1  1d    │    │ P1  1d    │     │
│  │ F002      │    │ F002      │    │ F001      │    │ F003      │     │
│  └───────────┘    └───────────┘    └───────────┘    │ Blocked by│     │
│  ┌───────────┐    ┌───────────┐    ┌───────────┐    │ T005      │     │
│  │ T008      │    │ T006  🔄  │    │ T002  ✓  │    └───────────┘     │
│  │ E2E Tests │    │ Login UI  │    │ User CRUD │                      │
│  │ P3  1d    │    │ P2  2d    │    │ P1  2d    │                      │
│  │ F002      │    │ F002      │    │ F001      │                      │
│  └───────────┘    └───────────┘    └───────────┘                      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3. Live Execution View

Real-time AI output streaming.

```
┌─────────────────────────────────────────────────────────────────────────┐
│  EXECUTION - T005: User Authentication API                    ⏸️  ⏹️   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Status: IN_PROGRESS          Duration: 00:05:23               │   │
│  │  Provider: Claude             Loop: 3/10                       │   │
│  │  Circuit: CLOSED              Branch: hermes/T005              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  AI OUTPUT                                              [Live] │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │  > Creating auth middleware...                                  │   │
│  │  > Writing file: internal/auth/middleware.go                   │   │
│  │                                                                 │   │
│  │  ```go                                                          │   │
│  │  package auth                                                   │   │
│  │                                                                 │   │
│  │  func AuthMiddleware(next http.Handler) http.Handler {         │   │
│  │      return http.HandlerFunc(func(w http.ResponseWriter...     │   │
│  │  ```                                                            │   │
│  │                                                                 │   │
│  │  > Running tests...                                             │   │
│  │  > ✓ TestAuthMiddleware passed                                 │   │
│  │  █                                                              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌───────────────────────────────────────┐  ┌───────────────────────┐  │
│  │  FILES CHANGED                        │  │  SUCCESS CRITERIA     │  │
│  │  + internal/auth/middleware.go        │  │  ☑ JWT validation     │  │
│  │  + internal/auth/middleware_test.go   │  │  ☑ Token refresh      │  │
│  │  ~ internal/server/routes.go          │  │  ☐ Rate limiting      │  │
│  └───────────────────────────────────────┘  └───────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 4. Dependency Graph Visualization

Interactive task dependency graph (v2 parallel execution ile entegre).

```
┌─────────────────────────────────────────────────────────────────────────┐
│  DEPENDENCY GRAPH                                    Zoom: 100%  ⟳     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│                          ┌─────┐                                        │
│                          │T001 │ ✓                                      │
│                          │ DB  │                                        │
│                          └──┬──┘                                        │
│                    ┌────────┼────────┐                                  │
│                    ▼        ▼        ▼                                  │
│                ┌─────┐  ┌─────┐  ┌─────┐                                │
│                │T002 │  │T003 │  │T004 │                                │
│                │CRUD │✓ │ API │🔄│ Auth│ 🔄                             │
│                └──┬──┘  └──┬──┘  └──┬──┘                                │
│                   │        │        │                                   │
│                   └────────┼────────┘                                   │
│                            ▼                                            │
│                        ┌─────┐                                          │
│                        │T005 │                                          │
│                        │ UI  │ ○                                        │
│                        └─────┘                                          │
│                                                                         │
│  Legend:  ✓ Completed   🔄 In Progress   ○ Not Started   ⚠️ Blocked    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 5. Configuration Editor

Visual configuration management.

```
┌─────────────────────────────────────────────────────────────────────────┐
│  CONFIGURATION                                          Save  Reset    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────┐                                                    │
│  │ AI Settings     │  Provider         [Claude      ▼]                 │
│  │ Task Mode       │  Timeout          [300    ] seconds               │
│  │ Parallel        │  Max Retries      [10     ]                       │
│  │ Paths           │  Stream Output    [✓]                             │
│  │ Advanced        │                                                    │
│  └─────────────────┘  PRD Timeout      [1200   ] seconds               │
│                                                                         │
│                       ─────────────────────────────                     │
│                                                                         │
│                       Auto Branch       [✓]                             │
│                       Auto Commit       [✓]                             │
│                       Autonomous        [✓]                             │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 6. Log Viewer

Advanced log viewing with filtering and search.

```
┌─────────────────────────────────────────────────────────────────────────┐
│  LOGS                    Level: [All ▼]  Task: [All ▼]  Search: [    ] │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  12:45:23  INFO   T005  Starting task execution                        │
│  12:45:24  DEBUG  T005  Provider: Claude, Timeout: 300s                │
│  12:45:25  INFO   T005  AI response received (1523 tokens)             │
│  12:46:01  INFO   T005  File created: internal/auth/middleware.go      │
│  12:46:02  DEBUG  T005  Running success criteria check                 │
│  12:46:03  WARN   T005  Test coverage below threshold (75%)            │
│  12:46:15  INFO   T005  Loop 2 started                                 │
│  12:46:45  INFO   T005  All tests passing                              │
│  12:46:46  INFO   T005  Task completed successfully                    │
│  12:46:47  INFO   GIT   Committing: feat(auth): add middleware         │
│  12:46:48  INFO   T006  Starting task execution                        │
│                                                                         │
│  ──────────────────────────────────────────────────────────────────────│
│  [Auto-scroll: ON]                         Showing 1-50 of 1,234 lines │
└─────────────────────────────────────────────────────────────────────────┘
```

### 7. PRD Editor

Visual PRD creation and editing.

```
┌─────────────────────────────────────────────────────────────────────────┐
│  PRD EDITOR - E-Commerce Platform                      Parse  Preview  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  # E-Commerce Platform PRD                                      │   │
│  │                                                                  │   │
│  │  ## Overview                                                     │   │
│  │  A modern e-commerce platform with user authentication,         │   │
│  │  product catalog, shopping cart, and payment processing.        │   │
│  │                                                                  │   │
│  │  ## Features                                                     │   │
│  │                                                                  │   │
│  │  ### Feature 1: User Authentication                             │   │
│  │  - User registration with email verification                    │   │
│  │  - Login with JWT tokens                                        │   │
│  │  - Password reset functionality                                 │   │
│  │  - OAuth integration (Google, GitHub)                           │   │
│  │                                                                  │   │
│  │  ### Feature 2: Product Catalog                                 │   │
│  │  - Product listing with pagination                              │   │
│  │  - Category filtering                                           │   │
│  │  - Search functionality                                         │   │
│  │  █                                                               │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  Word count: 234  │  Features detected: 2  │  Last saved: 12:30        │
└─────────────────────────────────────────────────────────────────────────┘
```

## Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              BROWSER                                    │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     React Frontend (SPA)                         │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │   │
│  │  │Dashboard│ │TaskBoard│ │Execution│ │  Logs   │ │ Config  │   │   │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘   │   │
│  │                           │                                      │   │
│  │                    ┌──────┴──────┐                               │   │
│  │                    │  WebSocket  │                               │   │
│  │                    └──────┬──────┘                               │   │
│  └───────────────────────────┼─────────────────────────────────────┘   │
└──────────────────────────────┼──────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           HERMES SERVER                                 │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      HTTP/WebSocket Server                       │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │   │
│  │  │  REST API    │  │  WebSocket   │  │Static Files  │          │   │
│  │  │  /api/...    │  │  /ws         │  │  /           │          │   │
│  │  └──────┬───────┘  └──────┬───────┘  └──────────────┘          │   │
│  └─────────┼─────────────────┼──────────────────────────────────────┘   │
│            │                 │                                          │
│            ▼                 ▼                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      HERMES CORE                                 │   │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐        │   │
│  │  │  Task  │ │   AI   │ │  Git   │ │Circuit │ │ Config │        │   │
│  │  │ Reader │ │Executor│ │  Ops   │ │Breaker │ │Manager │        │   │
│  │  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

### New Packages

```
internal/
  web/
    server.go           # HTTP server setup
    routes.go           # API route definitions
    handlers/
      dashboard.go      # Dashboard API handlers
      tasks.go          # Task CRUD handlers
      execution.go      # Execution control handlers
      config.go         # Configuration handlers
      logs.go           # Log streaming handlers
      websocket.go      # WebSocket connection handler
    middleware/
      auth.go           # Authentication middleware
      cors.go           # CORS middleware
      logging.go        # Request logging
    static/
      embed.go          # Embedded frontend files
  
  events/
    broker.go           # Event broker for real-time updates
    types.go            # Event type definitions
    subscriber.go       # WebSocket subscriber management

web/                    # Frontend (React)
  src/
    components/
      Dashboard/
      TaskBoard/
      ExecutionView/
      DependencyGraph/
      LogViewer/
      ConfigEditor/
      PRDEditor/
    hooks/
      useWebSocket.ts
      useTasks.ts
      useExecution.ts
    services/
      api.ts
      websocket.ts
    store/
      index.ts
      taskSlice.ts
      executionSlice.ts
    App.tsx
    index.tsx
  package.json
  vite.config.ts
```

### REST API Endpoints

```
GET    /api/dashboard              # Dashboard statistics
GET    /api/tasks                  # List all tasks
GET    /api/tasks/:id              # Get task details
PUT    /api/tasks/:id/status       # Update task status
GET    /api/features               # List all features
GET    /api/features/:id           # Get feature details

POST   /api/execution/start        # Start task execution
POST   /api/execution/stop         # Stop execution
GET    /api/execution/status       # Current execution status

GET    /api/config                 # Get configuration
PUT    /api/config                 # Update configuration

GET    /api/logs                   # Get logs (paginated)
GET    /api/logs/stream            # SSE log stream

POST   /api/prd/parse              # Parse PRD content
GET    /api/prd                    # Get current PRD

GET    /api/graph                  # Get dependency graph data

WS     /ws                         # WebSocket for real-time updates
```

### WebSocket Events

```typescript
// Client -> Server
interface ClientMessage {
  type: 'subscribe' | 'unsubscribe' | 'command';
  channel?: string;      // 'execution', 'logs', 'tasks'
  command?: string;      // 'start', 'stop', 'pause'
  payload?: any;
}

// Server -> Client
interface ServerMessage {
  type: 'event' | 'error' | 'ack';
  channel: string;
  event: string;
  data: any;
  timestamp: string;
}

// Event Types
type ExecutionEvent = 
  | { event: 'started', taskId: string }
  | { event: 'output', text: string }
  | { event: 'progress', percent: number }
  | { event: 'completed', taskId: string, success: boolean }
  | { event: 'error', message: string };

type TaskEvent =
  | { event: 'created', task: Task }
  | { event: 'updated', task: Task }
  | { event: 'statusChanged', taskId: string, status: string };

type LogEvent =
  | { event: 'entry', level: string, message: string, timestamp: string };
```

### Event Broker

```go
// internal/events/broker.go
type EventBroker struct {
    subscribers map[string]map[*Subscriber]bool
    broadcast   chan Event
    subscribe   chan *Subscriber
    unsubscribe chan *Subscriber
    mu          sync.RWMutex
}

type Subscriber struct {
    ID       string
    Channels []string
    Send     chan Event
    Done     chan struct{}
}

type Event struct {
    Channel   string      `json:"channel"`
    Type      string      `json:"type"`
    Data      interface{} `json:"data"`
    Timestamp time.Time   `json:"timestamp"`
}

func (b *EventBroker) Publish(channel string, eventType string, data interface{})
func (b *EventBroker) Subscribe(channels []string) *Subscriber
func (b *EventBroker) Unsubscribe(sub *Subscriber)
```

## Frontend Technology Stack

| Technology | Purpose |
|------------|---------|
| React 18 | UI Framework |
| TypeScript | Type Safety |
| Vite | Build Tool |
| TanStack Query | Data Fetching |
| Zustand | State Management |
| React Router | Routing |
| Tailwind CSS | Styling |
| shadcn/ui | Component Library |
| Recharts | Charts |
| React Flow | Dependency Graph |
| Monaco Editor | Code/PRD Editor |
| Lucide Icons | Icons |

## Authentication & Security

### Authentication Options

```go
// internal/web/middleware/auth.go
type AuthConfig struct {
    Mode     string   // "none", "token", "basic", "oauth"
    Token    string   // For token mode
    Users    []User   // For basic mode
    OAuth    OAuthConfig
}

type OAuthConfig struct {
    Provider     string // "github", "google"
    ClientID     string
    ClientSecret string
    RedirectURL  string
}
```

### Security Features

| Feature | Description |
|---------|-------------|
| CORS | Configurable allowed origins |
| Rate Limiting | Prevent API abuse |
| Token Auth | API token authentication |
| HTTPS | TLS support |
| Input Validation | Request validation |

## Configuration

```json
{
  "web": {
    "enabled": true,
    "port": 8080,
    "host": "0.0.0.0",
    "auth": {
      "mode": "token",
      "token": "your-secret-token"
    },
    "cors": {
      "allowedOrigins": ["http://localhost:3000"],
      "allowCredentials": true
    },
    "tls": {
      "enabled": false,
      "certFile": "",
      "keyFile": ""
    }
  }
}
```

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | true | Enable web interface |
| `port` | 8080 | HTTP server port |
| `host` | 0.0.0.0 | Bind address |
| `auth.mode` | none | Authentication mode |
| `cors.allowedOrigins` | ["*"] | Allowed CORS origins |
| `tls.enabled` | false | Enable HTTPS |

## CLI Changes

```bash
# Start web server (default)
hermes serve
hermes serve --port 8080
hermes serve --host 127.0.0.1

# Start with TUI (legacy mode)
hermes tui

# Open web interface in browser
hermes web

# API token management
hermes token generate
hermes token revoke <token>
hermes token list
```

## Embedded vs External Frontend

### Option A: Embedded (Recommended)

Frontend built and embedded into Go binary.

```go
// internal/web/static/embed.go
//go:embed dist/*
var staticFiles embed.FS

func StaticHandler() http.Handler {
    sub, _ := fs.Sub(staticFiles, "dist")
    return http.FileServer(http.FS(sub))
}
```

**Pros:**

- Single binary distribution
- No external dependencies
- Easy deployment

**Cons:**

- Larger binary size (~5-10MB)
- Rebuild required for frontend changes

### Option B: External

Frontend served separately or from CDN.

```go
// Serve from local directory
func StaticHandler(dir string) http.Handler {
    return http.FileServer(http.Dir(dir))
}
```

**Pros:**

- Smaller binary
- Independent frontend updates
- Development flexibility

**Cons:**

- More complex deployment
- Additional configuration

## Mobile Responsiveness

```
Desktop (1200px+)        Tablet (768px-1199px)      Mobile (<768px)
┌─────────────────┐      ┌─────────────────┐       ┌───────────┐
│ Sidebar │ Main  │      │    Full Width   │       │  Mobile   │
│         │       │      │    Content      │       │   View    │
│ Nav     │Content│      │                 │       │           │
│         │       │      │  Bottom Nav     │       │Bottom Nav │
└─────────────────┘      └─────────────────┘       └───────────┘
```

## Real-time Updates Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         REAL-TIME UPDATE FLOW                           │
└─────────────────────────────────────────────────────────────────────────┘

  AI Executor                   Event Broker                  WebSocket
      │                              │                            │
      │  ExecuteTask()               │                            │
      │──────────────────────────────>                            │
      │                              │                            │
      │  Publish("execution",        │                            │
      │    "started", task)          │                            │
      │                              │──────────────────────────────>
      │                              │   {type: "started",        │
      │                              │    taskId: "T005"}         │
      │                              │                            │
      │  AI Output Stream            │                            │
      │  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─>       │                            │
      │                              │──────────────────────────────>
      │  Publish("execution",        │   {type: "output",         │
      │    "output", text)           │    text: "Creating..."}    │
      │                              │                            │
      │  Task Complete               │                            │
      │──────────────────────────────>                            │
      │                              │──────────────────────────────>
      │  Publish("execution",        │   {type: "completed",      │
      │    "completed", result)      │    success: true}          │
      │                              │                            │
      │  Publish("tasks",            │                            │
      │    "statusChanged", task)    │──────────────────────────────>
      │                              │   {event: "statusChanged", │
      │                              │    taskId: "T005",         │
      │                              │    status: "COMPLETED"}    │
```

## Implementation Phases

### Phase 1: Foundation (v3.0.0-alpha)

- [ ] HTTP server setup
- [ ] REST API endpoints (tasks, config)
- [ ] Basic React frontend
- [ ] Static file embedding

### Phase 2: Real-time (v3.0.0-beta)

- [ ] WebSocket implementation
- [ ] Event broker
- [ ] Live execution view
- [ ] Log streaming

### Phase 3: Features (v3.0.0-rc)

- [ ] Dashboard with charts
- [ ] Kanban task board
- [ ] Dependency graph visualization
- [ ] Configuration editor
- [ ] PRD editor

### Phase 4: Polish (v3.0.0)

- [ ] Authentication
- [ ] Mobile responsiveness
- [ ] Dark mode
- [ ] Documentation
- [ ] Performance optimization

## Performance Considerations

| Concern | Solution |
|---------|----------|
| Large log files | Pagination, virtual scrolling |
| Many tasks | Virtualized list rendering |
| Real-time updates | Debounced UI updates |
| Bundle size | Code splitting, lazy loading |
| API calls | Request caching, optimistic updates |

## Browser Support

| Browser | Minimum Version |
|---------|-----------------|
| Chrome | 90+ |
| Firefox | 88+ |
| Safari | 14+ |
| Edge | 90+ |

## Deployment Options

### 1. Local Development

```bash
hermes serve --port 8080
# Open http://localhost:8080
```

### 2. Remote Server

```bash
hermes serve --host 0.0.0.0 --port 8080 --auth token
# Access from any device on network
```

### 3. Docker

```dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN make build-web

FROM alpine:latest
COPY --from=builder /app/bin/hermes /usr/local/bin/
EXPOSE 8080
CMD ["hermes", "serve"]
```

### 4. Reverse Proxy (Production)

```nginx
server {
    listen 443 ssl;
    server_name hermes.example.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Comparison: TUI vs Web

| Feature | TUI (v1-v2) | Web (v3) |
|---------|-------------|----------|
| Access | Local terminal | Any browser |
| Real-time | Yes | Yes (WebSocket) |
| Multi-user | No | Yes |
| Mobile | No | Yes |
| Visualizations | Text-based | Charts, graphs |
| Remote access | SSH required | Direct |
| Offline | Yes | Partial |
| Resource usage | Minimal | Moderate |

## Open Questions

1. **Keep TUI?** - Maintain TUI for terminal-only environments?
2. **Multi-project?** - Support managing multiple projects from one interface?
3. **User accounts?** - Full user management or simple token auth?
4. **Notifications?** - Browser notifications for task completion?
5. **Themes?** - Support for custom themes beyond dark/light?
6. **Plugins?** - Allow custom dashboard widgets?

## References

- [Go embed](https://pkg.go.dev/embed)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [React 18](https://react.dev/)
- [Vite](https://vitejs.dev/)
- [TanStack Query](https://tanstack.com/query)
- [shadcn/ui](https://ui.shadcn.com/)
- [React Flow](https://reactflow.dev/)
