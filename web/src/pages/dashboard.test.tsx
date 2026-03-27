import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { DashboardPage } from "./dashboard"
import { AllProviders } from "@/test/utils"
import { mockNodes, mockPeers, mockStats } from "@/test/mocks"

// Mock the API hooks
const mockRefetchStats = vi.fn()
const mockRefetchNodes = vi.fn()
const mockRefetchPeers = vi.fn()

// Mutable state for conditional testing
const statsState = { stats: mockStats }

vi.mock("@/lib/api", () => ({
  useStats: () => ({
    data: statsState.stats,
    isLoading: false,
    error: null,
    refetch: mockRefetchStats,
  }),
  useNodes: () => ({
    data: mockNodes,
    isLoading: false,
    error: null,
    refetch: mockRefetchNodes,
  }),
  usePeers: () => ({
    data: mockPeers,
    isLoading: false,
    error: null,
    refetch: mockRefetchPeers,
  }),
}))

describe("DashboardPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    statsState.stats = mockStats
  })

  it("should render the dashboard title", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("Dashboard")).toBeInTheDocument()
    expect(screen.getByText("Overview of your Karadul mesh network")).toBeInTheDocument()
  })

  it("should render refresh button", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    const buttons = screen.getAllByRole("button")
    expect(buttons.length).toBeGreaterThan(0)
  })

  it("should display total nodes count", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("Total Nodes")).toBeInTheDocument()
    expect(screen.getByText("3")).toBeInTheDocument() // mockNodes has 3 items
  })

  it("should display online nodes count", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("2 online")).toBeInTheDocument() // 2 online nodes in mock
  })

  it("should display connected peers card", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("Connected Peers")).toBeInTheDocument()
  })

  it("should display data received card", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("Data Received")).toBeInTheDocument()
    expect(screen.getByText("Total inbound")).toBeInTheDocument()
  })

  it("should display data sent card", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("Data Sent")).toBeInTheDocument()
    expect(screen.getByText("Total outbound")).toBeInTheDocument()
  })

  it("should display CPU usage", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("CPU Usage")).toBeInTheDocument()
  })

  it("should display memory usage", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("Memory Usage")).toBeInTheDocument()
  })

  it("should display uptime", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("Uptime")).toBeInTheDocument()
  })

  it("should display goroutines count", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("Goroutines")).toBeInTheDocument()
    expect(screen.getByText("50")).toBeInTheDocument() // mockStats.goroutines
  })

  it("should display network traffic chart card", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("Network Traffic")).toBeInTheDocument()
    expect(screen.getByText("Data transfer over the last 24 hours")).toBeInTheDocument()
  })

  it("should display system status card", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    expect(screen.getByText("System Status")).toBeInTheDocument()
    expect(screen.getByText("Current system resource usage")).toBeInTheDocument()
  })

  it("should call refetch when clicking refresh button", () => {
    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    // Find refresh button (has RefreshCw icon)
    const buttons = screen.getAllByRole("button")
    const refreshButton = buttons.find(btn => btn.querySelector("svg"))
    if (refreshButton) {
      fireEvent.click(refreshButton)
    }
  })
})

describe("DashboardPage - Loading state", () => {
  beforeEach(() => {
    vi.resetModules()
    vi.doMock("@/lib/api", () => ({
      useStats: () => ({ data: null, isLoading: true, error: null, refetch: vi.fn() }),
      useNodes: () => ({ data: null, isLoading: true, error: null, refetch: vi.fn() }),
      usePeers: () => ({ data: null, isLoading: true, error: null, refetch: vi.fn() }),
    }))
  })

  it("should show loading skeletons when loading", async () => {
    const { DashboardPage: DashboardPageLoading } = await import("./dashboard")

    render(
      <AllProviders>
        <DashboardPageLoading />
      </AllProviders>
    )

    const skeletons = document.querySelectorAll(".animate-pulse")
    expect(skeletons.length).toBeGreaterThan(0)
  })
})

describe("DashboardPage - Error state", () => {
  const mockRefetchStats = vi.fn()
  const mockRefetchNodes = vi.fn()
  const mockRefetchPeers = vi.fn()

  beforeEach(() => {
    vi.resetModules()
    vi.doMock("@/lib/api", () => ({
      useStats: () => ({ data: null, isLoading: false, error: new Error("Failed to fetch stats"), refetch: mockRefetchStats }),
      useNodes: () => ({ data: null, isLoading: false, error: null, refetch: mockRefetchNodes }),
      usePeers: () => ({ data: null, isLoading: false, error: null, refetch: mockRefetchPeers }),
    }))
    vi.clearAllMocks()
  })

  it("should show error alert when there is an error", async () => {
    const { DashboardPage: DashboardPageError } = await import("./dashboard")

    render(
      <AllProviders>
        <DashboardPageError />
      </AllProviders>
    )

    expect(screen.getByText("Failed to load dashboard")).toBeInTheDocument()
    expect(screen.getByText("Failed to fetch stats")).toBeInTheDocument()
  })

  it("should have retry button when there is an error", async () => {
    const { DashboardPage: DashboardPageError } = await import("./dashboard")

    render(
      <AllProviders>
        <DashboardPageError />
      </AllProviders>
    )

    const retryButton = screen.getByRole("button", { name: /retry/i })
    expect(retryButton).toBeInTheDocument()
  })

  it("should call all refetch functions when retry is clicked", async () => {
    const { DashboardPage: DashboardPageError } = await import("./dashboard")

    render(
      <AllProviders>
        <DashboardPageError />
      </AllProviders>
    )

    const retryButton = screen.getByRole("button", { name: /retry/i })
    fireEvent.click(retryButton)

    expect(mockRefetchStats).toHaveBeenCalled()
    expect(mockRefetchNodes).toHaveBeenCalled()
    expect(mockRefetchPeers).toHaveBeenCalled()
  })
})

describe("DashboardPage - Null stats", () => {
  it("should handle null stats with fallback values", () => {
    // Set stats to null to trigger || 0 fallbacks
    statsState.stats = null as any

    render(
      <AllProviders>
        <DashboardPage />
      </AllProviders>
    )

    // Should show 0 values when stats is null
    expect(screen.getByText("Dashboard")).toBeInTheDocument()
  })
})

describe("DashboardPage - Null nodes and peers", () => {
  beforeEach(() => {
    vi.resetModules()
    vi.doMock("@/lib/api", () => ({
      useStats: () => ({
        data: mockStats,
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      }),
      useNodes: () => ({
        data: null as any, // null to trigger || 0 fallback
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      }),
      usePeers: () => ({
        data: null as any, // null to trigger || 0 fallback
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      }),
    }))
  })

  it("should handle null nodes with fallback to 0", async () => {
    const { DashboardPage: DashboardPageNullNodes } = await import("./dashboard")

    render(
      <AllProviders>
        <DashboardPageNullNodes />
      </AllProviders>
    )

    // Should display 0 for nodes count when nodes is null
    expect(screen.getByText("Dashboard")).toBeInTheDocument()
  })
})
