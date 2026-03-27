import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { ScrollArea, ScrollBar } from "./scroll-area"
import { AllProviders } from "@/test/utils"

describe("ScrollArea", () => {
  it("should render scroll area with children", () => {
    render(
      <AllProviders>
        <ScrollArea>
          <div>Content inside scroll area</div>
        </ScrollArea>
      </AllProviders>
    )

    expect(screen.getByText("Content inside scroll area")).toBeInTheDocument()
  })

  it("should accept custom className", () => {
    render(
      <AllProviders>
        <ScrollArea className="custom-scroll">
          <div>Content</div>
        </ScrollArea>
      </AllProviders>
    )

    const scrollArea = document.querySelector(".custom-scroll")
    expect(scrollArea).toBeInTheDocument()
  })

  it("should render with viewport props", () => {
    render(
      <AllProviders>
        <ScrollArea>
          <div>Test content</div>
        </ScrollArea>
      </AllProviders>
    )

    expect(screen.getByText("Test content")).toBeInTheDocument()
  })
})

describe("ScrollBar", () => {
  it("should render scroll bar", () => {
    render(
      <AllProviders>
        <ScrollArea>
          <div>Content</div>
          <ScrollBar orientation="vertical" />
        </ScrollArea>
      </AllProviders>
    )

    expect(screen.getByText("Content")).toBeInTheDocument()
  })

  it("should support horizontal orientation", () => {
    render(
      <AllProviders>
        <ScrollArea>
          <div>Wide content</div>
          <ScrollBar orientation="horizontal" />
        </ScrollArea>
      </AllProviders>
    )

    expect(screen.getByText("Wide content")).toBeInTheDocument()
  })
})
