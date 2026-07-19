[![Docker Image CI](https://github.com/gregyjames/NanoStatus/actions/workflows/docker-image.yml/badge.svg)](https://github.com/gregyjames/NanoStatus/actions/workflows/docker-image.yml)
![GitHub repo size](https://img.shields.io/github/repo-size/gregyjames/NanoStatus)
![Docker Image Size (tag)](https://img.shields.io/docker/image-size/gjames8/nanostatus/latest)
![Docker Pulls](https://img.shields.io/docker/pulls/gjames8/nanostatus)

# NanoStatus

Finally got a cursor subscription so wanted to build something cool. NanoStatus is a lightweight, single-container monitoring dashboard built with Go and React. Monitor your services' uptime, response times, and health in real-time with a beautiful, modern interface. I personally made this for my PiZero, UptimeKuma was a little too heavy so wanted to make something a little lighter. 

# Sample
![Alt text](https://github.com/gregyjames/NanoStatus/blob/add_project/Sample.png?raw=true)

## ✨ Features

- 🚀 **Single Binary Deployment** - Everything embedded in one Go binary
- 📊 **Real-time Monitoring** - Server-Sent Events (SSE) for instant updates
- 🎨 **Modern UI** - Sleek dark theme with glassmorphism effects and smooth animations
- 📈 **Response Time Charts** - Historical data with multiple time frames (1h, 12h, 1w, 1y)
- 🔍 **Service Management** - Add, edit, pause, and delete monitors
- 📱 **Fully Responsive** - Works beautifully on desktop, tablet, and mobile
- 💾 **SQLite Database** - Lightweight persistence with automatic cleanup
- ⚡ **Efficient Updates** - Only sends data when values actually change
- 🎯 **Customizable Intervals** - Set individual check intervals per service
- 🧹 **Auto Cleanup** - Automatically removes check history older than 1 year

## 🚀 Quick Start

### Using Docker (Recommended)

1. **Build the Docker image:**
   ```bash
   docker build -t nanostatus .
   ```

2. **Run the container:**
   ```bash
   docker run -p 8080:8080 -v "$(pwd)/data:/data" nanostatus
   ```

   The application will be available at `http://localhost:8080`

   **Note:** The `-v "$(pwd)/data:/data"` flag persists the database across container restarts.

3. **Access the dashboard:**
   Open `http://localhost:8080` in your browser

### Local Development

#### Prerequisites
- [Bun](https://bun.sh) (for frontend)
- [Go](https://go.dev) 1.24+ (for backend)

#### Running Locally

1. **Build the frontend:**
   ```bash
   cd src
   bun install
   bun run build.ts --outdir=../dist
   cd ..
   ```

2. **Run the Go server:**
   ```bash
   go run .
   ```

   Or build and run:
   ```bash
   go build -o nanostatus .
   ./nanostatus
   ```

3. **Access the application:**
   Open http://localhost:8080 in your browser

#### Using Makefile

```bash
make build        # Build both frontend and backend
make run          # Build and run the application
make dev-frontend # Run frontend dev server (with hot reload)
make dev-backend  # Run Go server (requires built frontend)
make clean        # Clean build artifacts
```

## 📖 How It Works

### Architecture

- **Backend**: Go HTTP server with embedded frontend static files
- **Frontend**: React with TypeScript, Tailwind CSS, and Framer Motion
- **Database**: SQLite with GORM ORM
- **Real-time Updates**: Server-Sent Events (SSE) for efficient streaming
- **Background Jobs**: 
  - Automatic service health checks based on individual intervals
  - Daily cleanup of check history older than 1 year (runs at midnight)

### Monitoring

- Services are checked via actual HTTP requests
- Response times are measured and stored in the database
- Uptime is calculated from the last 24 hours of check history
- Stats are only calculated and broadcast when values change
- Updates are streamed to clients via SSE (no polling needed)

### Data Management

- Check history is stored in SQLite for historical analysis
- Automatic cleanup runs daily at 12:00 AM to remove data older than 1 year
- Database uses WAL mode for better concurrency
- All data persists in `/data` volume when using Docker

## 🔌 API Endpoints

### REST API

- `GET /api/monitors` - List all monitors
- `POST /api/monitors/create` - Create a new monitor
- `GET /api/stats` - Get overall statistics (only unpaused services)
- `GET /api/response-time?id=<id>&range=<range>` - Get response time history
  - `range` options: `1h`, `12h`, `24h`, `1w`, `1y` (default: `24h`)
- `GET /api/monitor?id=<id>` - Get specific monitor details
- `PUT /api/monitor?id=<id>` - Update a monitor or toggle pause state
- `DELETE /api/monitor?id=<id>` - Delete a monitor

### Server-Sent Events (SSE)

- `GET /api/events` - Real-time event stream
  - Event types: `monitor_update`, `monitor_added`, `monitor_deleted`, `stats_update`
  - Automatically reconnects on connection loss
  - Keepalive messages every 30 seconds

## ⚙️ Configuration

### Environment Variables

- `PORT` - Server port (default: `8080`)
- `DB_PATH` - Database file path
  - Default: `./nanostatus.db` (local) or `/data/nanostatus.db` (Docker)

### YAML Configuration

You can pre-populate monitors using a YAML configuration file. Place `monitors.yaml` in the same directory as your database file.

**Example `monitors.yaml` (in the same directory as your database):**

```yaml
monitors:
  - name: "Example.com"
    url: "https://example.com"
    icon: "🌐"
    checkInterval: 60
    isThirdParty: false
    paused: false

  - name: "Google"
    url: "https://google.com"
    icon: "🔍"
    checkInterval: 30
    isThirdParty: true
    paused: false

  - name: "GitHub"
    url: "https://github.com"
    icon: "💻"
    checkInterval: 120
    isThirdParty: true
    paused: false
```

**Configuration Fields:**
- `name` (required) - Display name for the service
- `url` (required) - Full URL to monitor (e.g., `https://example.com`)
- `icon` (optional) - Emoji icon to display
- `checkInterval` (optional) - How often to check in seconds (default: 60)
- `isThirdParty` (optional) - Whether this is a third-party service (default: false)
- `paused` (optional) - Whether monitoring should start paused (default: false)

**Location:**
- The YAML file must be named `monitors.yaml` and placed in the same directory as your database
- For Docker: If `DB_PATH=/data/nanostatus.db`, place the file at `/data/monitors.yaml`
- For local: If database is at `./nanostatus.db`, place the file at `./monitors.yaml`

**How It Works:**
- The YAML configuration is synchronized on every server startup
- Each monitor from YAML gets a hash calculated from its configuration
- The system compares hashes to detect changes:
  - **New monitors** in YAML are created
  - **Changed monitors** (different hash) are updated (preserving runtime data like status and uptime)
  - **Removed monitors** (no longer in YAML) are deleted
  - **Unchanged monitors** are left as-is
- Monitors created via the UI/API are **not** managed by YAML and won't be modified
- If a monitor with the same name/URL exists but was created via UI/API, the YAML version will be skipped to avoid duplicates

### Service Configuration

When creating a monitor via the UI or API, you can configure:
- **Name**: Display name for the service
- **URL**: Full URL to monitor (e.g., `https://example.com`)
- **Icon**: Optional emoji icon
- **Check Interval**: How often to check (10-3600 seconds, default: 60)
- **Third-party Service**: Flag for external services

## 🎯 Features in Detail

### Service Management
- **Add Services**: Create new monitors with custom check intervals
- **Edit Services**: Update name, URL, icon, and check interval
- **Pause/Resume**: Temporarily disable monitoring for specific services
- **Delete Services**: Remove monitors and their history

### Real-time Updates
- **SSE Streaming**: All updates pushed instantly to connected clients
- **Change Detection**: Stats only calculated and sent when values change
- **Debouncing**: Rapid monitor updates are batched for efficiency
- **No Polling**: Frontend receives updates via SSE, eliminating HTTP polling

### Response Time History
- **Multiple Time Frames**: View data for 1 hour, 12 hours, 1 week, or 1 year
- **Interactive Charts**: Beautiful area charts with gradient fills
- **Time-based Formatting**: Labels adapt to the selected time range

### Statistics
- **Overall Uptime**: Average uptime across all unpaused services
- **Service Counts**: Number of services online/offline (unpaused only)
- **Average Response Time**: Calculated from last 24 hours of check history
- **Real-time Updates**: Stats update automatically when monitors change

## 🏗️ Project Structure

```
.
├── main.go              # Main server entry point and routing
├── models.go            # Data models and structures
├── database.go            # Database initialization and seeding
├── config.go             # YAML configuration loader
├── checker.go            # Service health checking logic
├── stats.go              # Statistics calculation
├── sse.go                # Server-Sent Events broadcasting
├── handlers.go           # HTTP API endpoint handlers
├── cleanup.go            # Background cleanup jobs
├── go.mod                # Go dependencies
├── go.sum                # Go dependency checksums
├── Dockerfile            # Standard multi-stage Docker build (distroless)
├── Dockerfile.minimal    # Minimal Docker build with UPX compression (scratch)
├── Makefile              # Build automation
├── monitors.yaml.example # Example YAML configuration file
├── dist/                 # Frontend build output (generated)
├── nanostatus.db         # SQLite database (generated)
├── monitors.yaml         # YAML config (optional, same dir as DB)
├── src/                  # Frontend source code
│   ├── src/
│   │   ├── App.tsx       # Main React component
│   │   ├── components/   # React components
│   │   │   ├── Header.tsx
│   │   │   ├── StatsGrid.tsx
│   │   │   ├── ServicesGrid.tsx
│   │   │   ├── ServiceCard.tsx
│   │   │   ├── MonitorDetails.tsx
│   │   │   ├── AddServiceDialog.tsx
│   │   │   ├── EditServiceDialog.tsx
│   │   │   └── ui/       # shadcn/ui components
│   │   ├── types/        # TypeScript type definitions
│   │   └── ...
│   └── package.json
└── README.md
```

## 🔧 Development

### Building

```bash
# Build frontend
cd src
bun install
bun run build.ts --outdir=../dist

# Build backend
go build -o nanostatus .
```

### Docker Build

The Dockerfile uses a multi-stage build:
1. **Frontend Builder**: Builds React app with Bun
2. **Backend Builder**: Compiles Go binary with embedded frontend
3. **Final Stage**: Minimal distroless image (no shell, no package manager)

Result: Ultra-small, secure container (~16MB)

## 📝 License

See LICENSE file for details.

## 🙏 Acknowledgments

Built with:
- [Go](https://go.dev) - Backend server
- [React](https://react.dev) - Frontend framework
- [Bun](https://bun.sh) - JavaScript runtime and package manager
- [Tailwind CSS](https://tailwindcss.com) - Styling
- [Framer Motion](https://www.framer.com/motion/) - Animations
- [Recharts](https://recharts.org) - Charting
- [shadcn/ui](https://ui.shadcn.com) - UI components
- [GORM](https://gorm.io) - ORM
- [SQLite](https://sqlite.org) - Database

### Icon Attribution

The favicon/logo is from the [small-n-flat](http://paomedia.github.io/small-n-flat/) icon set, licensed under [CC0 1.0 Universal](http://creativecommons.org/publicdomain/zero/1.0/). See [ICON_LICENSE](ICON_LICENSE) for full license text.
