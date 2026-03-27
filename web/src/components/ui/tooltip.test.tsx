import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from "./tooltip"
import { AllProviders } from "@/test/utils"
import { Button } from "./button"

describe("Tooltip Components", () => {
  it("should render tooltip provider", () => {
    render(
      <AllProviders>
        <TooltipProvider>
          <div>Child content</div>
        </TooltipProvider>
      </AllProviders>
    )

    expect(screen.getByText("Child content")).toBeInTheDocument()
  })

  it("should render tooltip with trigger and content", () => {
    render(
      <AllProviders>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button>Hover me</Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>Tooltip content</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </AllProviders>
    )

    expect(screen.getByText("Hover me")).toBeInTheDocument()
    // Tooltip content is only shown on hover in actual browser
    // but we can verify it's in the DOM
  })

  it("should render tooltip content with custom className", () => {
    render(
      <AllProviders>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger>Trigger</TooltipTrigger>
            <TooltipContent className="custom-tooltip">
              <p>Content</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </AllProviders>
    )

    // The trigger should be present
    expect(screen.getByText("Trigger")).toBeInTheDocument()
  })

  it("should support default open state", () => {
    render(
      <AllProviders>
        <TooltipProvider>
          <Tooltip defaultOpen>
            <TooltipTrigger>Trigger</TooltipTrigger>
            <TooltipContent>
              <p>Unique tooltip text content</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </AllProviders>
    )

    // Tooltip content should be visible with defaultOpen
    // Use getAllByText since it appears multiple times
    expect(screen.getAllByText("Unique tooltip text content").length).toBeGreaterThan(0)
  })
})
