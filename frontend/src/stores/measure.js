import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useMeasureStore = defineStore('measure', () => {
  const activeTool = ref(null)
  const measurements = ref([])
  const currentPoints = ref([])

  const tools = [
    { id: 'distance', name: '距离测量', icon: 'Ruler', minPoints: 2 },
    { id: 'angle', name: '角度测量', icon: 'Connection', minPoints: 3 },
    { id: 'area', name: '面积计算', icon: 'Grid', minPoints: 3 },
    { id: 'volume', name: '体积估算', icon: 'Box', minPoints: 4 }
  ]

  function setActiveTool(toolId) {
    if (activeTool.value === toolId) {
      activeTool.value = null
      currentPoints.value = []
      return
    }
    activeTool.value = toolId
    currentPoints.value = []
  }

  function addPoint(point) {
    if (!activeTool.value) return
    currentPoints.value.push(point)

    const tool = tools.find(t => t.id === activeTool.value)
    if (tool && currentPoints.value.length >= tool.minPoints) {
      completeMeasurement()
    }
  }

  function completeMeasurement() {
    const tool = tools.find(t => t.id === activeTool.value)
    if (!tool || currentPoints.value.length < tool.minPoints) return

    let result = {}
    const pts = currentPoints.value

    switch (activeTool.value) {
      case 'distance': {
        const dx = pts[1].x - pts[0].x
        const dy = pts[1].y - pts[0].y
        const dz = pts[1].z - pts[0].z
        const dist = Math.sqrt(dx * dx + dy * dy + dz * dz)
        result = { value: dist, unit: 'mm', label: `${dist.toFixed(1)} mm` }
        break
      }
      case 'angle': {
        const v1 = {
          x: pts[0].x - pts[1].x,
          y: pts[0].y - pts[1].y,
          z: pts[0].z - pts[1].z
        }
        const v2 = {
          x: pts[2].x - pts[1].x,
          y: pts[2].y - pts[1].y,
          z: pts[2].z - pts[1].z
        }
        const dot = v1.x * v2.x + v1.y * v2.y + v1.z * v2.z
        const mag1 = Math.sqrt(v1.x * v1.x + v1.y * v1.y + v1.z * v1.z)
        const mag2 = Math.sqrt(v2.x * v2.x + v2.y * v2.y + v2.z * v2.z)
        const cosAngle = Math.max(-1, Math.min(1, dot / (mag1 * mag2)))
        const angle = Math.acos(cosAngle) * (180 / Math.PI)
        result = { value: angle, unit: '°', label: `${angle.toFixed(1)}°` }
        break
      }
      case 'area': {
        let area = 0
        const n = pts.length
        for (let i = 0; i < n; i++) {
          const j = (i + 1) % n
          area += pts[i].x * pts[j].z
          area -= pts[j].x * pts[i].z
        }
        area = Math.abs(area) / 2
        result = { value: area, unit: 'mm²', label: `${area.toFixed(2)} mm²` }
        break
      }
      case 'volume': {
        const minPt = { x: Infinity, y: Infinity, z: Infinity }
        const maxPt = { x: -Infinity, y: -Infinity, z: -Infinity }
        for (const p of pts) {
          minPt.x = Math.min(minPt.x, p.x)
          minPt.y = Math.min(minPt.y, p.y)
          minPt.z = Math.min(minPt.z, p.z)
          maxPt.x = Math.max(maxPt.x, p.x)
          maxPt.y = Math.max(maxPt.y, p.y)
          maxPt.z = Math.max(maxPt.z, p.z)
        }
        const vol = (maxPt.x - minPt.x) * (maxPt.y - minPt.y) * (maxPt.z - minPt.z)
        result = { value: vol, unit: 'mm³', label: `${vol.toFixed(2)} mm³` }
        break
      }
    }

    measurements.value.push({
      id: `meas_${Date.now()}`,
      type: activeTool.value,
      points: [...currentPoints.value],
      result,
      visible: true
    })

    currentPoints.value = []
  }

  function removeMeasurement(id) {
    measurements.value = measurements.value.filter(m => m.id !== id)
  }

  function toggleMeasurementVisibility(id) {
    const m = measurements.value.find(m => m.id === id)
    if (m) m.visible = !m.visible
  }

  function clearMeasurements() {
    measurements.value = []
    currentPoints.value = []
  }

  return {
    activeTool, measurements, currentPoints, tools,
    setActiveTool, addPoint, completeMeasurement,
    removeMeasurement, toggleMeasurementVisibility, clearMeasurements
  }
})
