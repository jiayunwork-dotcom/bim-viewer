<template>
  <div class="collision-panel">
    <div class="panel-header">
      <span>碰撞检测</span>
    </div>

    <div class="collision-config">
      <div class="group-section">
        <div class="group-label">A组构件 ({{ collisionStore.groupA.length }})</div>
        <div class="group-items">
          <el-tag
            v-for="id in collisionStore.groupA"
            :key="id"
            size="small"
            closable
            @close="collisionStore.removeFromGroupA(id)"
            class="group-tag"
          >
            {{ getElementName(id) }}
          </el-tag>
        </div>
        <el-select
          placeholder="按类型添加"
          size="small"
          style="width: 100%"
          @change="(cat) => collisionStore.addCategoryToGroup(modelStore, cat, 'A')"
          clearable
        >
          <el-option v-for="cat in viewerStore.categories" :key="cat.id" :label="cat.name" :value="cat.id" />
        </el-select>
      </div>

      <div class="group-section">
        <div class="group-label">B组构件 ({{ collisionStore.groupB.length }})</div>
        <div class="group-items">
          <el-tag
            v-for="id in collisionStore.groupB"
            :key="id"
            size="small"
            closable
            type="warning"
            @close="collisionStore.removeFromGroupB(id)"
            class="group-tag"
          >
            {{ getElementName(id) }}
          </el-tag>
        </div>
        <el-select
          placeholder="按类型添加"
          size="small"
          style="width: 100%"
          @change="(cat) => collisionStore.addCategoryToGroup(modelStore, cat, 'B')"
          clearable
        >
          <el-option v-for="cat in viewerStore.categories" :key="cat.id" :label="cat.name" :value="cat.id" />
        </el-select>
      </div>

      <div class="threshold-section">
        <span>软碰撞阈值 (mm)</span>
        <el-input-number
          v-model="collisionStore.threshold"
          :min="1"
          :max="500"
          size="small"
          style="width: 120px"
        />
      </div>

      <div class="action-buttons">
        <el-button
          type="primary"
          size="small"
          :loading="collisionStore.detecting"
          :disabled="!canDetect"
          @click="runDetection"
        >
          开始检测
        </el-button>
        <el-button size="small" @click="collisionStore.clearGroups">清空分组</el-button>
        <el-button
          size="small"
          :disabled="!collisionStore.currentTaskId"
          @click="exportReport"
        >
          导出CSV
        </el-button>
      </div>
    </div>

    <div v-if="collisionStore.results.length > 0" class="results-section">
      <div class="results-header">
        <span>检测结果 ({{ collisionStore.results.length }} 条)</span>
        <div class="filter-chips">
          <el-tag
            :type="severityFilter === 'all' ? 'primary' : 'info'"
            size="small"
            @click="severityFilter = 'all'"
            class="filter-chip"
          >
            全部
          </el-tag>
          <el-tag
            :type="severityFilter === 'hard' ? 'danger' : 'info'"
            size="small"
            @click="severityFilter = 'hard'"
            class="filter-chip"
          >
            硬碰撞
          </el-tag>
          <el-tag
            :type="severityFilter === 'soft' ? 'warning' : 'info'"
            size="small"
            @click="severityFilter = 'soft'"
            class="filter-chip"
          >
            软碰撞
          </el-tag>
        </div>
      </div>

      <el-table
        :data="filteredResults"
        size="small"
        max-height="200"
        @row-click="onResultClick"
        style="width: 100%"
        :row-class-name="getRowClassName"
      >
        <el-table-column prop="elementAName" label="构件A" width="100" />
        <el-table-column prop="elementAType" label="类型" width="80" />
        <el-table-column prop="elementBName" label="构件B" width="100" />
        <el-table-column prop="elementBType" label="类型" width="80" />
        <el-table-column prop="collisionType" label="类型" width="60">
          <template #default="{ row }">
            <el-tag :type="row.collisionType === 'hard' ? 'danger' : 'warning'" size="small">
              {{ row.collisionType === 'hard' ? '硬' : '软' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="penetration" label="深度/距离" width="80">
          <template #default="{ row }">
            {{ row.penetration.toFixed(1) }}mm
          </template>
        </el-table-column>
        <el-table-column prop="severity" label="严重度" width="60">
          <template #default="{ row }">
            <el-tag
              :type="row.severity === 'high' ? 'danger' : row.severity === 'medium' ? 'warning' : 'success'"
              size="small"
            >
              {{ row.severity === 'high' ? '高' : row.severity === 'medium' ? '中' : '低' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useModelStore } from '../../stores/model'
import { useCollisionStore } from '../../stores/collision'
import { useViewerStore } from '../../stores/viewer'
import { ElMessage } from 'element-plus'

const props = defineProps({
  renderer: Object
})

const modelStore = useModelStore()
const collisionStore = useCollisionStore()
const viewerStore = useViewerStore()
const severityFilter = ref('all')

const canDetect = computed(() => {
  return collisionStore.groupA.length > 0 && collisionStore.groupB.length > 0 && modelStore.currentModel
})

const filteredResults = computed(() => {
  if (severityFilter.value === 'all') return collisionStore.results
  if (severityFilter.value === 'hard') return collisionStore.results.filter(r => r.collisionType === 'hard')
  return collisionStore.results.filter(r => r.collisionType === 'soft')
})

function getElementName(id) {
  const el = modelStore.elements.find(e => e.id === id)
  return el ? (el.name || el.ifcGuid || id.slice(-8)) : id.slice(-8)
}

async function runDetection() {
  try {
    const result = await collisionStore.detectCollisions(modelStore.currentModel.id)
    ElMessage.success(`检测完成，发现 ${result.count} 处碰撞`)
  } catch (err) {
    ElMessage.error('碰撞检测失败: ' + err.message)
  }
}

function onResultClick(row) {
  if (props.renderer) {
    const point = row.collisionPoint
    props.renderer.flyTo(
      { x: point[0], y: point[1] + 10, z: point[2] + 10 },
      { x: point[0], y: point[1], z: point[2] }
    )

    props.renderer.highlightElement(row.elementAId, 0xff0000)
    props.renderer.addCollisionMarker({
      x: point[0], y: point[1], z: point[2]
    })
  }
}

function getRowClassName({ row }) {
  return `severity-${row.severity}`
}

function exportReport() {
  if (collisionStore.currentTaskId) {
    collisionStore.exportCSV(collisionStore.currentTaskId)
    ElMessage.success('碰撞报告已导出')
  }
}
</script>

<style scoped>
.collision-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.panel-header {
  padding: 8px 12px;
  border-bottom: 1px solid #2a2a4a;
}

.panel-header span {
  color: #ccddee;
  font-size: 13px;
  font-weight: 500;
}

.collision-config {
  padding: 8px 12px;
  border-bottom: 1px solid #2a2a4a;
}

.group-section {
  margin-bottom: 10px;
}

.group-label {
  color: #8899aa;
  font-size: 12px;
  margin-bottom: 4px;
}

.group-items {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 6px;
  max-height: 60px;
  overflow-y: auto;
}

.group-tag {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.threshold-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #8899aa;
  font-size: 12px;
  margin-bottom: 10px;
}

.action-buttons {
  display: flex;
  gap: 6px;
}

.results-section {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.results-header {
  padding: 6px 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: #8899aa;
  font-size: 12px;
  border-bottom: 1px solid #2a2a4a;
}

.filter-chips {
  display: flex;
  gap: 4px;
}

.filter-chip {
  cursor: pointer;
}

:deep(.severity-high) {
  background: rgba(255, 0, 0, 0.1);
}

:deep(.severity-medium) {
  background: rgba(255, 200, 0, 0.05);
}
</style>
