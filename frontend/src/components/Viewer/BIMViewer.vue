<template>
  <div class="bim-viewer" ref="viewerContainer">
    <div class="toolbar">
      <el-button-group>
        <el-tooltip content="适应视图" placement="bottom">
          <el-button size="small" @click="fitView">
            <el-icon><FullScreen /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="显示全部" placement="bottom">
          <el-button size="small" @click="showAll">
            <el-icon><View /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="清空选择" placement="bottom">
          <el-button size="small" @click="clearSelection">
            <el-icon><Close /></el-icon>
          </el-button>
        </el-tooltip>
      </el-button-group>

      <el-divider direction="vertical" />

      <el-button-group>
        <el-tooltip content="距离" placement="bottom">
          <el-button
            size="small"
            :type="measureStore.activeTool === 'distance' ? 'primary' : 'default'"
            @click="measureStore.setActiveTool('distance')"
          >
            <el-icon><Ruler /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="角度" placement="bottom">
          <el-button
            size="small"
            :type="measureStore.activeTool === 'angle' ? 'primary' : 'default'"
            @click="measureStore.setActiveTool('angle')"
          >
            <el-icon><Connection /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="面积" placement="bottom">
          <el-button
            size="small"
            :type="measureStore.activeTool === 'area' ? 'primary' : 'default'"
            @click="measureStore.setActiveTool('area')"
          >
            <el-icon><Grid /></el-icon>
          </el-button>
        </el-tooltip>
      </el-button-group>

      <el-divider direction="vertical" />

      <el-button-group>
        <el-tooltip content="XY剖切" placement="bottom">
          <el-button size="small" @click="addClipPlane('XY')">XY</el-button>
        </el-tooltip>
        <el-tooltip content="XZ剖切" placement="bottom">
          <el-button size="small" @click="addClipPlane('XZ')">XZ</el-button>
        </el-tooltip>
        <el-tooltip content="YZ剖切" placement="bottom">
          <el-button size="small" @click="addClipPlane('YZ')">YZ</el-button>
        </el-tooltip>
      </el-button-group>

      <el-divider direction="vertical" />

      <el-select v-model="floorClip" placeholder="楼层快切" size="small" style="width: 120px" clearable @change="onFloorClip">
        <el-option v-for="floor in floors" :key="floor.name" :label="floor.name" :value="floor.height" />
      </el-select>

      <div class="spacer" />

      <div class="fps-counter">FPS: {{ fps }}</div>
    </div>

    <div class="viewer-content">
      <div class="side-panel" v-show="viewerStore.activePanel === 'tree'" :class="{ collapsed: !viewerStore.activePanel }">
        <TreePanel />
      </div>

      <div class="canvas-area" ref="canvasContainer" />

      <div class="right-panel" v-show="viewerStore.activePanel === 'property'">
        <PropertyPanel />
      </div>
    </div>

    <div class="bottom-panel" v-show="viewerStore.activePanel === 'collision'">
      <CollisionPanel :renderer="renderer" />
    </div>

    <div class="side-buttons">
      <el-tooltip content="模型树" placement="left">
        <el-button
          :type="viewerStore.activePanel === 'tree' ? 'primary' : 'default'"
          size="small"
          circle
          @click="viewerStore.setActivePanel('tree')"
        >
          <el-icon><Document /></el-icon>
        </el-button>
      </el-tooltip>
      <el-tooltip content="属性" placement="left">
        <el-button
          :type="viewerStore.activePanel === 'property' ? 'primary' : 'default'"
          size="small"
          circle
          @click="viewerStore.setActivePanel('property')"
        >
          <el-icon><InfoFilled /></el-icon>
        </el-button>
      </el-tooltip>
      <el-tooltip content="碰撞检测" placement="left">
        <el-button
          :type="viewerStore.activePanel === 'collision' ? 'primary' : 'default'"
          size="small"
          circle
          @click="viewerStore.setActivePanel('collision')"
        >
          <el-icon><Warning /></el-icon>
        </el-button>
      </el-tooltip>
    </div>

    <ContextMenu :renderer="renderer" />

    <div class="category-filter" v-if="modelStore.currentModel">
      <div class="filter-title">构件筛选</div>
      <div class="filter-items">
        <div
          v-for="cat in viewerStore.categories"
          :key="cat.id"
          class="filter-item"
          :class="{ active: viewerStore.filterCategory === cat.id }"
          @click="toggleCategoryFilter(cat.id)"
        >
          <span class="color-dot" :style="{ background: cat.color }" />
          <span>{{ cat.name }}</span>
        </div>
      </div>
    </div>

    <div class="clip-panel" v-if="clipStore.planes.length > 0">
      <div class="clip-title">剖切面</div>
      <div v-for="plane in clipStore.planes" :key="plane.id" class="clip-item">
        <el-switch v-model="plane.enabled" size="small" @change="updateClipPlane(plane)" />
        <span>{{ plane.preset }}: {{ plane.position.toFixed(0) }}</span>
        <el-slider
          v-model="plane.position"
          :min="-50000"
          :max="50000"
          :step="100"
          size="small"
          style="flex: 1; margin: 0 8px"
          @input="updateClipPlane(plane)"
        />
        <el-button size="small" circle @click="clipStore.removePlane(plane.id); updateAllClipPlanes()">
          <el-icon><Close /></el-icon>
        </el-button>
      </div>
    </div>

    <div class="measure-results" v-if="measureStore.measurements.length > 0">
      <div class="measure-title">测量结果</div>
      <div v-for="m in measureStore.measurements" :key="m.id" class="measure-item">
        <span>{{ getToolName(m.type) }}: {{ m.result.label }}</span>
        <el-button size="small" circle @click="measureStore.removeMeasurement(m.id)">
          <el-icon><Close /></el-icon>
        </el-button>
      </div>
      <el-button size="small" @click="takeScreenshot" style="margin-top: 8px">截图导出</el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useModelStore } from '../../stores/model'
import { useViewerStore } from '../../stores/viewer'
import { useClipStore } from '../../stores/clip'
import { useMeasureStore } from '../../stores/measure'
import { useCollisionStore } from '../../stores/collision'
import { BIMRenderer } from '../../utils/BIMRenderer'
import TreePanel from '../TreePanel/TreePanel.vue'
import PropertyPanel from '../PropertyPanel/PropertyPanel.vue'
import CollisionPanel from '../CollisionPanel/CollisionPanel.vue'
import ContextMenu from '../ContextMenu/ContextMenu.vue'
import * as THREE from 'three'
import { ElMessage } from 'element-plus'

const route = useRoute()
const modelStore = useModelStore()
const viewerStore = useViewerStore()
const clipStore = useClipStore()
const measureStore = useMeasureStore()

const viewerContainer = ref(null)
const canvasContainer = ref(null)
const renderer = ref(null)
const lodManager = ref(null)
const fps = ref(0)
const floorClip = ref(null)

const floors = computed(() => {
  const floorMap = modelStore.elementsByFloor
  const result = []
  let height = 0
  for (const [name] of Object.entries(floorMap)) {
    result.push({ name, height })
    height += 3.5
  }
  return result
})

const categoryColors = {
  Wall: 0xb0b0b0,
  Slab: 0xc8c8c8,
  Column: 0xa0a0a0,
  Beam: 0x909090,
  Pipe: 0x4080ff,
  Duct: 0x80c0ff,
  Equipment: 0x808080,
  Door: 0xd0a060,
  Window: 0x80d0ff
}

onMounted(async () => {
  const modelId = route.params.modelId
  if (!modelId) return

  await modelStore.fetchModel(modelId)
  await modelStore.fetchSpatialTree(modelId)
  await modelStore.fetchElements(modelId)

  if (canvasContainer.value) {
    renderer.value = new BIMRenderer(canvasContainer.value)
    renderer.value.onElementClick = handleElementClick
    renderer.value.onBoxSelect = handleBoxSelect
    renderer.value.onMeasureClick = handleMeasurePoint

    await loadModelMeshes()

    renderer.value.fitToView()
  }

  const fpsInterval = setInterval(() => {
    if (renderer.value) {
      fps.value = renderer.value.getFPS()
    }
  }, 1000)

  onUnmounted(() => {
    clearInterval(fpsInterval)
    if (renderer.value) {
      renderer.value.dispose()
    }
  })
})

watch(() => measureStore.activeTool, (tool) => {
  if (renderer.value) {
    renderer.value.setMeasureMode(!!tool)
  }
})

function handleMeasurePoint(point) {
  const p = { x: point.x, y: point.y, z: point.z }
  measureStore.addPoint(p)

  if (measureStore.currentPoints.length === 0) {
    const latest = measureStore.measurements[measureStore.measurements.length - 1]
    if (latest) {
      renderer.value.addMeasurementLine(latest.points)
      const midIdx = Math.floor(latest.points.length / 2)
      const midPt = latest.points[midIdx]
      renderer.value.addMeasurementLabel(latest.result.label, midPt)
      ElMessage.success(`测量完成: ${latest.result.label}`)
    }
  } else {
    const tempPoints = [...measureStore.currentPoints]
    if (tempPoints.length >= 2) {
      renderer.value.addMeasurementLine(tempPoints, 0x00ff00)
    }
  }
}

async function loadModelMeshes() {
  if (!renderer.value || !modelStore.currentModel) return

  const modelId = modelStore.currentModel.id
  const elements = modelStore.elements

  const geometryGroups = new Map()
  for (const e of elements) {
    if (!geometryGroups.has(e.geometryHash)) {
      geometryGroups.set(e.geometryHash, [])
    }
    geometryGroups.get(e.geometryHash).push(e)
  }

  for (const [hash, group] of geometryGroups) {
    const template = group[0]
    const geometry = createElementGeometry(template)
    const color = categoryColors[template.category] || 0x808080
    const material = new THREE.MeshPhongMaterial({
      color,
      side: THREE.DoubleSide,
      clippingPlanes: renderer.value.clippingPlanes
    })

    if (group.length > 1) {
      const instances = group.map(e => ({
        elementId: e.id,
        position: {
          x: (e.aabbMin[0] + e.aabbMax[0]) / 2,
          y: (e.aabbMin[1] + e.aabbMax[1]) / 2,
          z: (e.aabbMin[2] + e.aabbMax[2]) / 2
        },
        scale: {
          x: (e.aabbMax[0] - e.aabbMin[0]) / (template.aabbMax[0] - template.aabbMin[0]) || 1,
          y: (e.aabbMax[1] - e.aabbMin[1]) / (template.aabbMax[1] - template.aabbMin[1]) || 1,
          z: (e.aabbMax[2] - e.aabbMin[2]) / (template.aabbMax[2] - template.aabbMin[2]) || 1
        }
      }))

      renderer.value.addInstancedMesh(hash, geometry, material, instances)
    } else {
      const e = group[0]
      renderer.value.addElementMesh(e.id, geometry, material, {
        x: (e.aabbMin[0] + e.aabbMax[0]) / 2,
        y: (e.aabbMin[1] + e.aabbMax[1]) / 2,
        z: (e.aabbMin[2] + e.aabbMax[2]) / 2
      })
    }
  }
}

function createElementGeometry(element) {
  const sx = element.aabbMax[0] - element.aabbMin[0] || 1
  const sy = element.aabbMax[1] - element.aabbMin[1] || 1
  const sz = element.aabbMax[2] - element.aabbMin[2] || 1

  switch (element.category) {
    case 'Pipe':
    case 'Duct':
      return new THREE.CylinderGeometry(
        Math.min(sx, sy) / 2,
        Math.min(sx, sy) / 2,
        sz,
        16
      )
    default:
      return new THREE.BoxGeometry(sx, sy, sz)
  }
}

function handleElementClick(elementId, shiftKey, contextMenuPos) {
  if (contextMenuPos) {
    viewerStore.showContextMenu(contextMenuPos.x, contextMenuPos.y, elementId)
    return
  }

  if (shiftKey) {
    modelStore.addToSelection(elementId)
  } else {
    modelStore.selectElement(elementId)
  }

  renderer.value.highlightElement(elementId)
  viewerStore.setActivePanel('property')
}

function handleBoxSelect(elementIds) {
  modelStore.selectElements(elementIds)
}

function fitView() {
  if (renderer.value) renderer.value.fitToView()
}

function showAll() {
  if (renderer.value) renderer.value.showAllElements()
  viewerStore.showAllElements()
  modelStore.clearSelection()
  renderer.value.clearHighlight()
}

function clearSelection() {
  modelStore.clearSelection()
  renderer.value.clearHighlight()
}

function toggleCategoryFilter(category) {
  viewerStore.setFilterCategory(category)

  if (viewerStore.filterCategory) {
    renderer.value.setCategoryOpacity(category, 0.15, modelStore)
  } else {
    for (const e of modelStore.elements) {
      renderer.value.setElementOpacity(e.id, 1.0)
    }
  }
}

function addClipPlane(preset) {
  clipStore.addPlane(preset, 0)
  updateAllClipPlanes()
}

function updateClipPlane(plane) {
  if (renderer.value && plane.enabled) {
    const threePlane = renderer.value.clippingPlanes.find(p =>
      p.normal.x === plane.normal.x &&
      p.normal.y === plane.normal.y &&
      p.normal.z === plane.normal.z
    )
    if (threePlane) {
      renderer.value.updateClippingPlane(threePlane, plane.constant)
    } else {
      updateAllClipPlanes()
    }
  } else {
    updateAllClipPlanes()
  }
}

function updateAllClipPlanes() {
  if (!renderer.value) return
  renderer.value.clearClippingPlanes()
  for (const plane of clipStore.planes) {
    if (plane.enabled) {
      renderer.value.addClippingPlane(plane.normal, plane.constant)
    }
  }
}

function onFloorClip(height) {
  if (height === null || height === undefined) {
    clipStore.clearAll()
    updateAllClipPlanes()
    return
  }
  clipStore.clearAll()
  clipStore.addPlane('XZ', height)
  updateAllClipPlanes()
}

function getToolName(type) {
  const tool = measureStore.tools.find(t => t.id === type)
  return tool ? tool.name : type
}

function takeScreenshot() {
  if (renderer.value) {
    const dataUrl = renderer.value.takeScreenshot()
    const link = document.createElement('a')
    link.download = `bim_screenshot_${Date.now()}.png`
    link.href = dataUrl
    link.click()
  }
}
</script>

<style scoped>
.bim-viewer {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #1a1a2e;
  position: relative;
}

.toolbar {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  background: #16213e;
  border-bottom: 1px solid #2a2a4a;
  gap: 8px;
  z-index: 10;
}

.toolbar .spacer {
  flex: 1;
}

.fps-counter {
  color: #88cc88;
  font-size: 12px;
  font-family: monospace;
}

.viewer-content {
  display: flex;
  flex: 1;
  overflow: hidden;
  position: relative;
}

.side-panel {
  width: 300px;
  background: #16213e;
  border-right: 1px solid #2a2a4a;
  overflow-y: auto;
  z-index: 5;
  transition: width 0.2s;
}

.side-panel.collapsed {
  width: 0;
}

.canvas-area {
  flex: 1;
  position: relative;
}

.right-panel {
  width: 320px;
  background: #16213e;
  border-left: 1px solid #2a2a4a;
  overflow-y: auto;
  z-index: 5;
}

.bottom-panel {
  height: 300px;
  background: #16213e;
  border-top: 1px solid #2a2a4a;
  overflow-y: auto;
  z-index: 5;
}

.side-buttons {
  position: absolute;
  left: 8px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 10;
}

.category-filter {
  position: absolute;
  bottom: 16px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(22, 33, 62, 0.95);
  border: 1px solid #2a2a4a;
  border-radius: 8px;
  padding: 8px 12px;
  z-index: 10;
}

.filter-title {
  color: #aabbcc;
  font-size: 12px;
  margin-bottom: 6px;
}

.filter-items {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  color: #8899aa;
  font-size: 12px;
  padding: 2px 6px;
  border-radius: 4px;
  transition: all 0.2s;
}

.filter-item:hover,
.filter-item.active {
  color: #ffffff;
  background: rgba(64, 128, 255, 0.2);
}

.color-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}

.clip-panel {
  position: absolute;
  top: 60px;
  right: 340px;
  background: rgba(22, 33, 62, 0.95);
  border: 1px solid #2a2a4a;
  border-radius: 8px;
  padding: 8px 12px;
  z-index: 10;
  min-width: 280px;
}

.clip-title,
.measure-title {
  color: #aabbcc;
  font-size: 12px;
  margin-bottom: 6px;
}

.clip-item {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #8899aa;
  font-size: 12px;
  margin-bottom: 4px;
}

.measure-results {
  position: absolute;
  top: 60px;
  left: 316px;
  background: rgba(22, 33, 62, 0.95);
  border: 1px solid #2a2a4a;
  border-radius: 8px;
  padding: 8px 12px;
  z-index: 10;
  min-width: 200px;
}

.measure-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #8899aa;
  font-size: 12px;
  margin-bottom: 4px;
}
</style>
