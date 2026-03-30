# Skill: Svelte + Cytoscape.js Graph Visualization

Panduan integrasi Cytoscape.js ke dalam Svelte untuk knowledge graph Water.

---

## Setup

```bash
cd web
npm install cytoscape cytoscape-elk
npm install chart.js
npm install axios date-fns
npm install tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

---

## TypeScript Types

```typescript
// src/types/index.ts

export interface KnowledgeNode {
  node_id: string
  content: string
  source_type: 'mcp_output' | 'context' | 'memory'
  source_tool: string | null
  tokens_in: number
  tokens_out: number
  access_count: number
  importance_score: number
  retention_confidence: number
  community_id?: number
  tags: string[]
}

export interface GraphEdge {
  edge_id: string
  from_node_id: string
  to_node_id: string
  edge_type: 'semantic' | 'causal' | 'retrieval'
  weight: number
  salience: number
}

export interface GraphData {
  nodes: KnowledgeNode[]
  edges: GraphEdge[]
}

export interface Stats {
  total_nodes: number
  total_edges: number
  total_tokens: number
  avg_retention: number
  communities: number
}
```

---

## API Client

```typescript
// src/api.ts
import axios from 'axios'
import type { GraphData, KnowledgeNode, Stats } from './types'

const BASE = 'http://localhost:3141/api'

export const api = {
  async getGraph(): Promise<GraphData> {
    const { data } = await axios.get(`${BASE}/graph`)
    return data
  },
  
  async getNodes(limit = 100): Promise<KnowledgeNode[]> {
    const { data } = await axios.get(`${BASE}/nodes`, { params: { limit } })
    return data.nodes
  },
  
  async getStats(): Promise<Stats> {
    const { data } = await axios.get(`${BASE}/stats`)
    return data
  },
  
  async postEvent(event: object): Promise<void> {
    await axios.post(`${BASE}/events`, event)
  }
}
```

---

## Svelte Stores

```typescript
// src/stores/graph.ts
import { writable, derived } from 'svelte/store'
import type { GraphData, KnowledgeNode } from '../types'

export const graphData = writable<GraphData>({ nodes: [], edges: [] })
export const selectedNode = writable<KnowledgeNode | null>(null)
export const loading = writable(false)
export const error = writable<string | null>(null)

// Derived: nodes grouped by community
export const nodesByCommunity = derived(graphData, ($g) => {
  const groups: Record<number, KnowledgeNode[]> = {}
  for (const node of $g.nodes) {
    const cid = node.community_id ?? 0
    if (!groups[cid]) groups[cid] = []
    groups[cid].push(node)
  }
  return groups
})
```

---

## Graph Component (Cytoscape.js)

```svelte
<!-- src/components/Graph.svelte -->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import cytoscape from 'cytoscape'
  import { graphData, selectedNode } from '../stores/graph'
  import type { GraphData } from '../types'

  let container: HTMLDivElement
  let cy: cytoscape.Core

  // Community colors — up to 8 clusters
  const COMMUNITY_COLORS = [
    '#6366f1', '#f59e0b', '#10b981', '#ef4444',
    '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16'
  ]

  // Convert store data → Cytoscape elements
  function toElements(data: GraphData): cytoscape.ElementDefinition[] {
    const nodes = data.nodes.map(n => ({
      data: {
        id: n.node_id,
        label: n.content.substring(0, 40) + '...',
        tokens: n.tokens_in + n.tokens_out,
        importance: n.importance_score,
        retention: n.retention_confidence,
        community: n.community_id ?? 0,
        color: COMMUNITY_COLORS[(n.community_id ?? 0) % COMMUNITY_COLORS.length],
        // Node size proportional to access count (min 20, max 60)
        size: Math.max(20, Math.min(60, 20 + n.access_count * 4)),
        raw: n
      }
    }))
    
    const edges = data.edges.map(e => ({
      data: {
        id: e.edge_id,
        source: e.from_node_id,
        target: e.to_node_id,
        weight: e.weight,
        // Salience → opacity (fade old edges)
        opacity: Math.max(0.1, e.salience),
        edgeType: e.edge_type
      }
    }))
    
    return [...nodes, ...edges]
  }

  function initCytoscape(data: GraphData) {
    if (cy) cy.destroy()
    
    cy = cytoscape({
      container,
      elements: toElements(data),
      
      style: [
        {
          selector: 'node',
          style: {
            'background-color': 'data(color)',
            'label': 'data(label)',
            'width': 'data(size)',
            'height': 'data(size)',
            'font-size': '10px',
            'text-valign': 'bottom',
            'text-halign': 'center',
            'color': '#e5e7eb',
            'text-outline-color': '#1f2937',
            'text-outline-width': 1,
            'border-width': 2,
            'border-color': '#374151'
          }
        },
        {
          selector: 'node:selected',
          style: {
            'border-color': '#f59e0b',
            'border-width': 3,
          }
        },
        {
          selector: 'edge',
          style: {
            'width': 'mapData(weight, 0, 1, 1, 4)',
            'opacity': 'data(opacity)',
            'line-color': (ele) => {
              const type = ele.data('edgeType')
              return type === 'semantic' ? '#6366f1'
                   : type === 'causal'   ? '#f59e0b'
                   :                       '#10b981'
            },
            'curve-style': 'bezier',
            'target-arrow-shape': 'triangle',
            'target-arrow-color': '#6b7280',
            'arrow-scale': 0.8,
          }
        }
      ],
      
      layout: {
        name: 'cose',        // Force-directed, works well for knowledge graphs
        animate: false,
        nodeRepulsion: 400000,
        idealEdgeLength: 100,
        gravity: 80,
      }
    })

    // Click node → select
    cy.on('tap', 'node', (evt) => {
      const node = evt.target.data('raw')
      selectedNode.set(node)
    })
    
    // Click background → deselect
    cy.on('tap', (evt) => {
      if (evt.target === cy) selectedNode.set(null)
    })
  }

  // Re-render when data changes
  $: if (container && $graphData.nodes.length > 0) {
    initCytoscape($graphData)
  }

  onMount(() => {
    // Initial render with current store value
    if ($graphData.nodes.length > 0) {
      initCytoscape($graphData)
    }
  })

  onDestroy(() => {
    if (cy) cy.destroy()
  })
</script>

<div class="graph-container">
  <div bind:this={container} class="w-full h-full" />
  
  {#if $graphData.nodes.length === 0}
    <div class="empty-state">
      <p class="text-gray-400">No knowledge nodes yet.</p>
      <p class="text-gray-500 text-sm">Run <code>water serve</code> and connect an agent.</p>
    </div>
  {/if}
</div>

<style>
  .graph-container {
    position: relative;
    width: 100%;
    height: 100%;
    background: #111827;
    border-radius: 8px;
    overflow: hidden;
  }
  .empty-state {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }
</style>
```

---

## WebSocket Live Updates

```typescript
// src/stores/websocket.ts
import { graphData } from './graph'

let ws: WebSocket | null = null

export function connectWebSocket() {
  ws = new WebSocket('ws://localhost:3141/ws')
  
  ws.onopen = () => console.log('Water WebSocket connected')
  
  ws.onmessage = (evt) => {
    try {
      const event = JSON.parse(evt.data)
      // Append new node/edge to graph store
      graphData.update(g => ({
        nodes: [...g.nodes, event.node].filter(Boolean),
        edges: [...g.edges, ...(event.edges ?? [])]
      }))
    } catch (e) {
      console.error('WS parse error', e)
    }
  }
  
  ws.onclose = () => {
    // Reconnect after 3s
    setTimeout(connectWebSocket, 3000)
  }
}

export function disconnectWebSocket() {
  ws?.close()
}
```

---

## App Layout

```svelte
<!-- src/App.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import Graph from './components/Graph.svelte'
  import Sidebar from './components/Sidebar.svelte'
  import Metrics from './components/Metrics.svelte'
  import { graphData, loading, error } from './stores/graph'
  import { connectWebSocket } from './stores/websocket'
  import { api } from './api'

  onMount(async () => {
    loading.set(true)
    try {
      const data = await api.getGraph()
      graphData.set(data)
      connectWebSocket()
    } catch (e) {
      error.set('Failed to connect to Water server. Is `water serve` running?')
    } finally {
      loading.set(false)
    }
  })
</script>

<div class="flex h-screen bg-gray-950 text-gray-100">
  <!-- Left: Graph -->
  <div class="flex-1 p-4">
    <Graph />
  </div>
  
  <!-- Right: Sidebar + Metrics -->
  <div class="w-80 flex flex-col border-l border-gray-800">
    <Sidebar />
    <Metrics />
  </div>
</div>
```

---

## Tips

- Gunakan `cose` layout untuk knowledge graph (force-directed, lebih natural dari `breadthfirst`)
- Update Cytoscape secara incremental dengan `cy.add(elements)` daripada re-render penuh
- Edge opacity = salience value (0.1–1.0) — ini yang buat edges "fade" seiring waktu
- Node size berbasis `access_count` — node yang sering diakses tampak lebih besar