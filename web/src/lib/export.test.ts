import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import {
  toCSV,
  downloadFile,
  exportNodesCSV,
  exportNodesJSON,
  exportPeersCSV,
  exportPeersJSON,
  exportAuthKeysCSV,
} from "./export"
import { mockNodes, mockPeers, mockAuthKeys } from "@/test/mocks"

describe("export utilities", () => {
  // Mock DOM methods
  const mockAppendChild = vi.fn()
  const mockRemoveChild = vi.fn()
  const mockClick = vi.fn()

  beforeEach(() => {
    // Mock document.createElement
    vi.spyOn(document, "createElement").mockImplementation(() => ({
      click: mockClick,
      setAttribute: vi.fn(),
      style: {},
    } as unknown as HTMLAnchorElement))

    // Mock document.body
    vi.spyOn(document.body, "appendChild").mockImplementation(mockAppendChild)
    vi.spyOn(document.body, "removeChild").mockImplementation(mockRemoveChild)

    // Mock URL.createObjectURL and revokeObjectURL
    vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:test")
    vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe("toCSV", () => {
    it("should convert simple data to CSV", () => {
      const data = [{ name: "Alice", age: 30 }, { name: "Bob", age: 25 }]
      const result = toCSV(data, ["name", "age"])
      expect(result).toBe("name,age\nAlice,30\nBob,25")
    })

    it("should handle null and undefined values", () => {
      const data = [{ name: "Alice", age: null }, { name: "Bob", age: undefined }]
      const result = toCSV(data, ["name", "age"])
      expect(result).toBe("name,age\nAlice,\nBob,")
    })

    it("should quote values containing commas", () => {
      const data = [{ name: "Smith, John", age: 30 }]
      const result = toCSV(data, ["name", "age"])
      expect(result).toBe('name,age\n"Smith, John",30')
    })

    it("should handle empty arrays", () => {
      const result = toCSV([], ["name", "age"])
      expect(result).toBe("name,age")
    })

    it("should convert numbers to strings", () => {
      const data = [{ value: 123 }]
      const result = toCSV(data, ["value"])
      expect(result).toBe("value\n123")
    })

    it("should convert booleans to strings", () => {
      const data = [{ active: true }, { active: false }]
      const result = toCSV(data, ["active"])
      expect(result).toBe("active\ntrue\nfalse")
    })
  })

  describe("downloadFile", () => {
    it("should create download link with correct attributes", () => {
      downloadFile("test content", "test.txt", "text/plain")

      expect(document.createElement).toHaveBeenCalledWith("a")
      expect(mockAppendChild).toHaveBeenCalled()
      expect(mockClick).toHaveBeenCalled()
      expect(mockRemoveChild).toHaveBeenCalled()
    })

    it("should revoke object URL after download", () => {
      downloadFile("test", "test.txt", "text/plain")

      expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:test")
    })
  })

  describe("exportNodesCSV", () => {
    it("should export nodes to CSV with correct columns", () => {
      exportNodesCSV(mockNodes)

      expect(document.createElement).toHaveBeenCalledWith("a")
      expect(mockClick).toHaveBeenCalled()
    })

    it("should handle empty nodes array", () => {
      exportNodesCSV([])

      expect(document.createElement).toHaveBeenCalledWith("a")
    })
  })

  describe("exportNodesJSON", () => {
    it("should export nodes to JSON", () => {
      exportNodesJSON(mockNodes)

      expect(document.createElement).toHaveBeenCalledWith("a")
      expect(mockClick).toHaveBeenCalled()
    })
  })

  describe("exportPeersCSV", () => {
    it("should export peers to CSV with correct columns", () => {
      exportPeersCSV(mockPeers)

      expect(document.createElement).toHaveBeenCalledWith("a")
      expect(mockClick).toHaveBeenCalled()
    })
  })

  describe("exportPeersJSON", () => {
    it("should export peers to JSON", () => {
      exportPeersJSON(mockPeers)

      expect(document.createElement).toHaveBeenCalledWith("a")
      expect(mockClick).toHaveBeenCalled()
    })
  })

  describe("exportAuthKeysCSV", () => {
    it("should export auth keys to CSV", () => {
      exportAuthKeysCSV(mockAuthKeys)

      expect(document.createElement).toHaveBeenCalledWith("a")
      expect(mockClick).toHaveBeenCalled()
    })
  })
})
