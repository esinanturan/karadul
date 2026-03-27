import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { Avatar, AvatarFallback, AvatarImage } from "./avatar"
import { AllProviders } from "@/test/utils"

describe("Avatar Components", () => {
  it("should render Avatar component", () => {
    render(
      <AllProviders>
        <Avatar>
          <AvatarFallback>JD</AvatarFallback>
        </Avatar>
      </AllProviders>
    )

    expect(screen.getByText("JD")).toBeInTheDocument()
  })

  it("should render Avatar with image", () => {
    render(
      <AllProviders>
        <Avatar>
          <AvatarImage src="https://example.com/avatar.jpg" alt="User" />
          <AvatarFallback>JD</AvatarFallback>
        </Avatar>
      </AllProviders>
    )

    expect(screen.getByText("JD")).toBeInTheDocument()
  })

  it("should render AvatarFallback with custom className", () => {
    render(
      <AllProviders>
        <Avatar>
          <AvatarFallback className="custom-class">AB</AvatarFallback>
        </Avatar>
      </AllProviders>
    )

    const fallback = screen.getByText("AB")
    expect(fallback).toHaveClass("custom-class")
  })
})
