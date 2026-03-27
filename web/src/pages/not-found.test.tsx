import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NotFoundPage } from "./not-found"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

// Mock react-router-dom Link component
vi.mock("react-router-dom", () => ({
  Link: ({ children, to, className }: { children: React.ReactNode; to: string; className?: string }) => (
    <a href={to} className={className}>
      {children}
    </a>
  ),
}))

// Mock window.history.back
const mockBack = vi.fn()
vi.stubGlobal("history", {
  back: mockBack,
  pushState: vi.fn(),
  replaceState: vi.fn(),
  forward: vi.fn(),
  go: vi.fn(),
})

function renderWithProviders() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <NotFoundPage />
    </QueryClientProvider>
  )
}

describe("NotFoundPage", () => {
  beforeEach(() => {
    mockBack.mockReset()
  })

  it("should render 404 message", () => {
    renderWithProviders()

    expect(screen.getByText("404")).toBeInTheDocument()
    expect(screen.getByText("Page not found")).toBeInTheDocument()
    expect(screen.getByText(/The page you are looking for doesn't exist/)).toBeInTheDocument()
  })

  it("should have Go Home link", () => {
    renderWithProviders()

    const goHomeLink = screen.getByRole("link", { name: /go home/i })
    expect(goHomeLink).toBeInTheDocument()
    expect(goHomeLink).toHaveAttribute("href", "/")
  })

  it("should have Go Back button", () => {
    renderWithProviders()

    const goBackButton = screen.getByRole("button", { name: /go back/i })
    expect(goBackButton).toBeInTheDocument()
  })

  it("should go back when Go Back is clicked", () => {
    renderWithProviders()

    const goBackButton = screen.getByRole("button", { name: /go back/i })
    fireEvent.click(goBackButton)

    expect(mockBack).toHaveBeenCalled()
  })

  it("should render the alert triangle icon", () => {
    renderWithProviders()

    // Check for the SVG icon
    expect(document.querySelector("svg")).toBeInTheDocument()
  })

  it("should have proper card structure", () => {
    const { container } = renderWithProviders()

    // Check for Card component - look for any rounded element
    const roundedElement = container.querySelector("[class*='rounded']")
    expect(roundedElement).toBeInTheDocument()
  })

  it("should display the warning icon container", () => {
    const { container } = renderWithProviders()

    // Check for the circular icon container
    expect(container.querySelector(".rounded-full")).toBeInTheDocument()
  })

  it("should have flex column layout for buttons on mobile", () => {
    const { container } = renderWithProviders()

    // Check for flex layout on buttons
    const buttonContainer = container.querySelector(".flex-col.gap-2")
    expect(buttonContainer).toBeInTheDocument()
  })
})
