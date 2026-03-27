import { describe, it, expect, beforeEach } from "vitest"
import { act } from "@testing-library/react"
import { useKaradulStore } from "./store"
import { mockNodes, mockPeers, mockTopology, mockStats } from "@/test/mocks"

describe("useKaradulStore", () => {
  beforeEach(() => {
    // Reset store to initial state
    act(() => {
      useKaradulStore.setState({
        nodes: [],
        peers: [],
        topology: { nodes: [], connections: [] },
        stats: null,
        selectedNode: null,
        darkMode: false,
        isConnected: false,
        isLoading: true,
      })
    })
  })

  describe("nodes", () => {
    it("should have initial empty nodes array", () => {
      expect(useKaradulStore.getState().nodes).toEqual([])
    })

    it("should set nodes", () => {
      act(() => {
        useKaradulStore.getState().setNodes(mockNodes)
      })
      expect(useKaradulStore.getState().nodes).toEqual(mockNodes)
    })

    it("should replace nodes on subsequent calls", () => {
      act(() => {
        useKaradulStore.getState().setNodes(mockNodes)
        useKaradulStore.getState().setNodes([mockNodes[0]])
      })
      expect(useKaradulStore.getState().nodes).toHaveLength(1)
    })
  })

  describe("peers", () => {
    it("should have initial empty peers array", () => {
      expect(useKaradulStore.getState().peers).toEqual([])
    })

    it("should set peers", () => {
      act(() => {
        useKaradulStore.getState().setPeers(mockPeers)
      })
      expect(useKaradulStore.getState().peers).toEqual(mockPeers)
    })
  })

  describe("topology", () => {
    it("should have initial empty topology", () => {
      const topology = useKaradulStore.getState().topology
      expect(topology.nodes).toEqual([])
      expect(topology.connections).toEqual([])
    })

    it("should set topology", () => {
      act(() => {
        useKaradulStore.getState().setTopology(mockTopology)
      })
      const topology = useKaradulStore.getState().topology
      expect(topology.nodes).toHaveLength(2)
      expect(topology.connections).toHaveLength(1)
    })
  })

  describe("stats", () => {
    it("should have initial null stats", () => {
      expect(useKaradulStore.getState().stats).toBeNull()
    })

    it("should set stats", () => {
      act(() => {
        useKaradulStore.getState().setStats(mockStats)
      })
      expect(useKaradulStore.getState().stats).toEqual(mockStats)
    })
  })

  describe("selectedNode", () => {
    it("should have initial null selectedNode", () => {
      expect(useKaradulStore.getState().selectedNode).toBeNull()
    })

    it("should set selectedNode", () => {
      act(() => {
        useKaradulStore.getState().setSelectedNode(mockNodes[0])
      })
      expect(useKaradulStore.getState().selectedNode).toEqual(mockNodes[0])
    })

    it("should clear selectedNode with null", () => {
      act(() => {
        useKaradulStore.getState().setSelectedNode(mockNodes[0])
        useKaradulStore.getState().setSelectedNode(null)
      })
      expect(useKaradulStore.getState().selectedNode).toBeNull()
    })
  })

  describe("darkMode", () => {
    it("should have initial false darkMode", () => {
      expect(useKaradulStore.getState().darkMode).toBe(false)
    })

    it("should toggle darkMode", () => {
      act(() => {
        useKaradulStore.getState().toggleDarkMode()
      })
      expect(useKaradulStore.getState().darkMode).toBe(true)

      act(() => {
        useKaradulStore.getState().toggleDarkMode()
      })
      expect(useKaradulStore.getState().darkMode).toBe(false)
    })
  })

  describe("isConnected", () => {
    it("should have initial false isConnected", () => {
      expect(useKaradulStore.getState().isConnected).toBe(false)
    })

    it("should set isConnected", () => {
      act(() => {
        useKaradulStore.getState().setIsConnected(true)
      })
      expect(useKaradulStore.getState().isConnected).toBe(true)
    })
  })

  describe("isLoading", () => {
    it("should have initial true isLoading", () => {
      expect(useKaradulStore.getState().isLoading).toBe(true)
    })

    it("should set isLoading", () => {
      act(() => {
        useKaradulStore.getState().setIsLoading(false)
      })
      expect(useKaradulStore.getState().isLoading).toBe(false)
    })
  })
})
