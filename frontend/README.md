# Workflow SA Frontend

This is a standalone Single Page Application (SPA) built with Vite, React, TypeScript, and Tailwind CSS. It provides a BPMN Modeler and dashboards for monitoring processes and workflow instances.

## Features

- **Dashboard**: View key statistics about workflow instances (Total, Active, Completed, Failed).
- **Modeler**: A custom BPMN 2.0 modeler built with `React Flow` (`@xyflow/react`), allowing you to create and edit workflow diagrams.
- **Processes**: View deployed process definitions (currently a placeholder).
- **Instances**: Monitor running process instances (currently a placeholder).

## Tech Stack

- **Framework**: React + Vite
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **UI Components**: Shadcn/ui (Radix UI + Tailwind)
- **Icons**: Lucide React
- **Charts**: Recharts
- **BPMN**: Custom Parser + React Flow (@xyflow/react)
- **Routing**: React Router DOM

## Getting Started

### Prerequisites

- Node.js (v18+ recommended)
- npm
- **Backend Services**: The frontend requires the backend services to be running. See the [root README](../README.md) for instructions on starting the full stack using Docker Compose.

### Installation

1. Navigate to the frontend directory:
   ```bash
   cd frontend
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

### Running the Development Server

To start the local development server:

```bash
npm run dev
```

The application will be available at `http://localhost:5173` (or the next available port).

### Building for Production

To build the application for production deployment:

```bash
npm run build
```

The build artifacts will be stored in the `dist/` directory.

## Project Structure

- `src/layouts`: Application layouts (e.g., DashboardLayout with Sidebar).
- `src/pages`: Main page components (Dashboard, Modeler, Processes, Instances).
- `src/components`: Reusable UI components.
- `src/lib`: Utility functions.
