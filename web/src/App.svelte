<script lang="ts">
  type Node = {
    node_id: string
    content: string
    source_type: string
    source_tool?: string | null
    tokens_in: number
    tokens_out: number
    access_count: number
    importance_score: number
    retention_confidence: number
  }

  type GraphData = { nodes: Node[]; edges: any[] }

  let loading = true
  let error: string | null = null
  let nodes: Node[] = []

  async function loadGraph() {
    loading = true
    error = null
    try {
      const res = await fetch('/api/graph')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as GraphData
      nodes = data.nodes ?? []
    } catch (e: any) {
      error = e?.message ?? 'Failed to load graph'
    } finally {
      loading = false
    }
  }

  function connectWS() {
    const ws = new WebSocket(`ws://${location.host}/ws`)
    ws.onmessage = () => {
      // event arrived -> reload snapshot (simple + reliable for MVP)
      loadGraph()
    }
    ws.onclose = () => setTimeout(connectWS, 1500)
  }

  loadGraph().then(connectWS)
</script>

<main style="font-family: ui-sans-serif, system-ui; padding: 18px; background: #0b1220; min-height: 100vh; color: #e5e7eb">
  <h1 style="margin: 0 0 8px 0">Water</h1>
  <div style="color:#94a3b8; margin-bottom:14px">
    Live nodes from <code style="background:#111827; padding:2px 6px; border-radius:6px">/api/graph</code>
    via <code style="background:#111827; padding:2px 6px; border-radius:6px">/ws</code>.
  </div>

  {#if loading}
    <div style="color:#94a3b8">Loading…</div>
  {:else if error}
    <div style="color:#fca5a5">Error: {error}</div>
    <div style="margin-top:10px">
      Is <code style="background:#111827; padding:2px 6px; border-radius:6px">water serve</code> running on <code style="background:#111827; padding:2px 6px; border-radius:6px">3141</code>?
    </div>
  {:else}
    <div style="display:flex; gap:12px; flex-wrap:wrap">
      {#each nodes as n (n.node_id)}
        <section style="background:#0f172a; border:1px solid #1f2937; border-radius:12px; padding:12px; width: 320px">
          <div style="font-weight:700; margin-bottom:6px">{n.node_id}</div>
          <div style="color:#94a3b8; font-size:12px; margin-bottom:8px">
            {n.source_type}{#if n.source_tool} · {n.source_tool}{/if}
          </div>
          <div style="white-space:pre-wrap; font-size:13px; line-height:1.35">{n.content}</div>
        </section>
      {/each}
    </div>
  {/if}
</main>

