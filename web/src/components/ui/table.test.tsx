import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import {
  Table,
  TableHeader,
  TableBody,
  TableFooter,
  TableHead,
  TableRow,
  TableCell,
  TableCaption,
} from "./table"
import { AllProviders } from "@/test/utils"

describe("Table components", () => {
  it("should render Table", () => {
    const { container } = render(
      <AllProviders>
        <Table>
          <TableBody>
            <TableRow>
              <TableCell>Cell</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </AllProviders>
    )

    expect(screen.getByText("Cell")).toBeInTheDocument()
    expect(container.querySelector("table")).toBeInTheDocument()
  })

  it("should render TableHeader", () => {
    const { container } = render(
      <AllProviders>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Header</TableHead>
            </TableRow>
          </TableHeader>
        </Table>
      </AllProviders>
    )

    expect(screen.getByText("Header")).toBeInTheDocument()
    expect(container.querySelector("thead")).toBeInTheDocument()
  })

  it("should render TableBody", () => {
    const { container } = render(
      <AllProviders>
        <Table>
          <TableBody>
            <TableRow>
              <TableCell>Body cell</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </AllProviders>
    )

    expect(screen.getByText("Body cell")).toBeInTheDocument()
    expect(container.querySelector("tbody")).toBeInTheDocument()
  })

  it("should render TableFooter", () => {
    const { container } = render(
      <AllProviders>
        <Table>
          <TableBody>
            <TableRow>
              <TableCell>Data</TableCell>
            </TableRow>
          </TableBody>
          <TableFooter>
            <TableRow>
              <TableCell>Footer</TableCell>
            </TableRow>
          </TableFooter>
        </Table>
      </AllProviders>
    )

    expect(screen.getByText("Footer")).toBeInTheDocument()
    expect(container.querySelector("tfoot")).toBeInTheDocument()
  })

  it("should render TableRow", () => {
    const { container } = render(
      <AllProviders>
        <Table>
          <TableBody>
            <TableRow>
              <TableCell>Row cell</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </AllProviders>
    )

    expect(container.querySelector("tr")).toBeInTheDocument()
  })

  it("should render TableHead", () => {
    const { container } = render(
      <AllProviders>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Head cell</TableHead>
            </TableRow>
          </TableHeader>
        </Table>
      </AllProviders>
    )

    expect(container.querySelector("th")).toBeInTheDocument()
  })

  it("should render TableCell", () => {
    const { container } = render(
      <AllProviders>
        <Table>
          <TableBody>
            <TableRow>
              <TableCell>Data cell</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </AllProviders>
    )

    expect(container.querySelector("td")).toBeInTheDocument()
  })

  it("should render TableCaption", () => {
    const { container } = render(
      <AllProviders>
        <Table>
          <TableCaption>Table caption</TableCaption>
          <TableBody>
            <TableRow>
              <TableCell>Cell</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </AllProviders>
    )

    expect(screen.getByText("Table caption")).toBeInTheDocument()
    expect(container.querySelector("caption")).toBeInTheDocument()
  })

  it("should render complete table structure", () => {
    const { container } = render(
      <AllProviders>
        <Table>
          <TableCaption>Employee Data</TableCaption>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Role</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow>
              <TableCell>John</TableCell>
              <TableCell>Developer</TableCell>
            </TableRow>
          </TableBody>
          <TableFooter>
            <TableRow>
              <TableCell colSpan={2}>Total: 1 employee</TableCell>
            </TableRow>
          </TableFooter>
        </Table>
      </AllProviders>
    )

    expect(screen.getByText("Employee Data")).toBeInTheDocument()
    expect(screen.getByText("Name")).toBeInTheDocument()
    expect(screen.getByText("Role")).toBeInTheDocument()
    expect(screen.getByText("John")).toBeInTheDocument()
    expect(screen.getByText("Developer")).toBeInTheDocument()
    expect(screen.getByText("Total: 1 employee")).toBeInTheDocument()

    // Verify structure
    expect(container.querySelector("table")).toBeInTheDocument()
    expect(container.querySelector("thead")).toBeInTheDocument()
    expect(container.querySelector("tbody")).toBeInTheDocument()
    expect(container.querySelector("tfoot")).toBeInTheDocument()
    expect(container.querySelector("caption")).toBeInTheDocument()
  })

  it("should render TableFooter with custom className", () => {
    const { container } = render(
      <AllProviders>
        <Table>
          <TableFooter className="custom-footer">
            <TableRow>
              <TableCell>Footer</TableCell>
            </TableRow>
          </TableFooter>
        </Table>
      </AllProviders>
    )

    expect(container.querySelector(".custom-footer")).toBeInTheDocument()
  })
})
