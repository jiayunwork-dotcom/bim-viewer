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
      <div class="stats-bar">
        <div
          class="stat-item"
          :class="{ active: collisionStore.statusFilter === 'all' }"
          @click="collisionStore.setStatusFilter('all')"
        >
          <span class="stat-label">全部</span>
          <span class="stat-count">{{ collisionStore.stats.total }}</span>
        </div>
        <div
          class="stat-item pending"
          :class="{ active: collisionStore.statusFilter === 'pending' }"
          @click="collisionStore.setStatusFilter('pending')"
        >
          <span class="stat-label">待处理</span>
          <span class="stat-count">{{ collisionStore.stats.pending }}</span>
        </div>
        <div
          class="stat-item confirmed"
          :class="{ active: collisionStore.statusFilter === 'confirmed' }"
          @click="collisionStore.setStatusFilter('confirmed')"
        >
          <span class="stat-label">已确认</span>
          <span class="stat-count">{{ collisionStore.stats.confirmed }}</span>
        </div>
        <div
          class="stat-item false"
          :class="{ active: collisionStore.statusFilter === 'false_positive' }"
          @click="collisionStore.setStatusFilter('false_positive')"
        >
          <span class="stat-label">误报</span>
          <span class="stat-count">{{ collisionStore.stats.false }}</span>
        </div>
        <div
          class="stat-item resolved"
          :class="{ active: collisionStore.statusFilter === 'resolved' }"
          @click="collisionStore.setStatusFilter('resolved')"
        >
          <span class="stat-label">已解决</span>
          <span class="stat-count">{{ collisionStore.stats.resolved }}</span>
        </div>
      </div>

      <div class="results-header">
        <div class="filter-chips">
          <el-tag
            :type="collisionStore.severityFilter === 'all' ? 'primary' : 'info'"
            size="small"
            @click="collisionStore.setSeverityFilter('all')"
            class="filter-chip"
          >
            全部类型
          </el-tag>
          <el-tag
            :type="collisionStore.severityFilter === 'hard' ? 'danger' : 'info'"
            size="small"
            @click="collisionStore.setSeverityFilter('hard')"
            class="filter-chip"
          >
            硬碰撞
          </el-tag>
          <el-tag
            :type="collisionStore.severityFilter === 'soft' ? 'warning' : 'info'"
            size="small"
            @click="collisionStore.setSeverityFilter('soft')"
            class="filter-chip"
          >
            软碰撞
          </el-tag>
        </div>
        <div class="batch-actions" v-if="collisionStore.hasSelection">
          <span class="selected-count">已选 {{ collisionStore.selectedIds.length }} 条</span>
          <el-dropdown trigger="click" @command="handleBatchStatusChange">
            <el-button size="small" type="primary">
              批量标记
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item :command="collisionStore.COLLISION_STATUS.PENDING">
                  <el-icon><Clock /></el-icon> 待处理
                </el-dropdown-item>
                <el-dropdown-item :command="collisionStore.COLLISION_STATUS.CONFIRMED">
                  <el-icon><WarningFilled /></el-icon> 已确认
                </el-dropdown-item>
                <el-dropdown-item :command="collisionStore.COLLISION_STATUS.FALSE_POSITIVE">
                  <el-icon><CircleCheckFilled /></el-icon> 误报
                </el-dropdown-item>
                <el-dropdown-item :command="collisionStore.COLLISION_STATUS.RESOLVED">
                  <el-icon><Check /></el-icon> 已解决
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button size="small" @click="collisionStore.clearSelection">取消选择</el-button>
        </div>
      </div>

      <el-table
        :data="collisionStore.filteredResults"
        size="small"
        max-height="300"
        @row-click="onRowClick"
        style="width: 100%"
        :row-class-name="getRowClassName"
        ref="resultTable"
      >
        <el-table-column width="40" align="center">
          <template #header>
            <el-checkbox
              :model-value="collisionStore.allSelected"
              :indeterminate="collisionStore.hasSelection && !collisionStore.allSelected"
              @change="collisionStore.toggleAllSelection"
              @click.stop
            />
          </template>
          <template #default="{ row }">
            <el-checkbox
              :model-value="collisionStore.selectedIds.includes(row.id)"
              @change="collisionStore.toggleSelection(row.id)"
              @click.stop
            />
          </template>
        </el-table-column>
        <el-table-column prop="elementAName" label="构件A" width="90" />
        <el-table-column prop="elementAType" label="类型" width="60" />
        <el-table-column prop="elementAFloor" label="A楼层" width="70" />
        <el-table-column prop="elementBName" label="构件B" width="90" />
        <el-table-column prop="elementBType" label="类型" width="60" />
        <el-table-column prop="elementBFloor" label="B楼层" width="70" />
        <el-table-column prop="collisionType" label="碰撞" width="50">
          <template #default="{ row }">
            <el-tag :type="row.collisionType === 'hard' ? 'danger' : 'warning'" size="small">
              {{ row.collisionType === 'hard' ? '硬' : '软' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="penetration" label="深度/距离" width="70">
          <template #default="{ row }">
            {{ row.penetration.toFixed(1) }}mm
          </template>
        </el-table-column>
        <el-table-column prop="severity" label="严重度" width="50">
          <template #default="{ row }">
            <el-tag
              :type="row.severity === 'high' ? 'danger' : row.severity === 'medium' ? 'warning' : 'success'"
              size="small"
            >
              {{ row.severity === 'high' ? '高' : row.severity === 'medium' ? '中' : '低' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag
              :type="collisionStore.STATUS_TYPES[row.status] || 'info'"
              size="small"
            >
              {{ collisionStore.STATUS_LABELS[row.status] || row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button-group>
              <el-dropdown trigger="click" @command="(cmd) => handleStatusChange(row, cmd)">
                <el-button size="small" type="primary">
                  标记
                  <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item :command="collisionStore.COLLISION_STATUS.PENDING">
                      <el-icon><Clock /></el-icon> 待处理
                    </el-dropdown-item>
                    <el-dropdown-item :command="collisionStore.COLLISION_STATUS.CONFIRMED">
                      <el-icon><WarningFilled /></el-icon> 已确认
                    </el-dropdown-item>
                    <el-dropdown-item :command="collisionStore.COLLISION_STATUS.FALSE_POSITIVE">
                      <el-icon><CircleCheckFilled /></el-icon> 误报
                    </el-dropdown-item>
                    <el-dropdown-item :command="collisionStore.COLLISION_STATUS.RESOLVED">
                      <el-icon><Check /></el-icon> 已解决
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              <el-button size="small" @click.stop="showHistory(row)">
                <el-icon><Histogram /></el-icon>
              </el-button>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog
      v-model="statusDialogVisible"
      :title="statusDialogTitle"
      width="400px"
      @close="statusForm.remark = ''"
    >
      <el-form :model="statusForm" label-width="80px">
        <el-form-item label="新状态">
          <el-tag :type="collisionStore.STATUS_TYPES[statusForm.newStatus]" size="large">
            {{ collisionStore.STATUS_LABELS[statusForm.newStatus] }}
          </el-tag>
        </el-form-item>
        <el-form-item label="备注" required>
          <el-input
            v-model="statusForm.remark"
            type="textarea"
            :rows="4"
            placeholder="请填写状态变更的原因说明..."
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="statusDialogVisible = false">取消</el-button>
        <el-button type="primary" :disabled="!statusForm.remark.trim()" @click="confirmStatusChange">
          确认
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="historyDialogVisible"
      title="历史记录"
      width="500px"
    >
      <div v-if="currentHistoryResult" class="history-header">
        <div class="history-item-info">
          <span class="label">构件A:</span>
          <span>{{ currentHistoryResult.elementAName }}</span>
        </div>
        <div class="history-item-info">
          <span class="label">构件B:</span>
          <span>{{ currentHistoryResult.elementBName }}</span>
        </div>
        <div class="history-item-info">
          <span class="label">当前状态:</span>
          <el-tag :type="collisionStore.STATUS_TYPES[currentHistoryResult.status]">
            {{ collisionStore.STATUS_LABELS[currentHistoryResult.status] }}
          </el-tag>
        </div>
      </div>
      <div v-loading="collisionStore.historyLoading" class="timeline-container">
        <el-timeline>
          <el-timeline-item
            v-for="(item, index) in currentHistory"
            :key="item.id"
            :timestamp="formatDateTime(item.createdAt)"
            :type="getTimelineType(item.newStatus)"
            :hollow="index === 0"
          >
            <div class="history-content">
              <div class="history-status">
                <span class="status-transition">
                  <el-tag :type="collisionStore.STATUS_TYPES[item.oldStatus]" size="small">
                    {{ collisionStore.STATUS_LABELS[item.oldStatus] || '创建' }}
                  </el-tag>
                  <el-icon class="arrow-icon"><ArrowRight /></el-icon>
                  <el-tag :type="collisionStore.STATUS_TYPES[item.newStatus]" size="small">
                    {{ collisionStore.STATUS_LABELS[item.newStatus] }}
                  </el-tag>
                </span>
              </div>
              <div class="history-remark">{{ item.remark }}</div>
              <div class="history-meta">
                <span>操作人: {{ item.operator }}</span>
              </div>
            </div>
          </el-timeline-item>
          <el-timeline-item
            v-if="currentHistoryResult && currentHistory.length === 0"
            type="primary"
          >
            <div class="history-content">
              <div class="history-status">
                <el-tag type="info" size="small">创建</el-tag>
                <el-icon class="arrow-icon"><ArrowRight /></el-icon>
                <el-tag
                  :type="collisionStore.STATUS_TYPES[currentHistoryResult.status]"
                  size="small"
                >
                  {{ collisionStore.STATUS_LABELS[currentHistoryResult.status] }}
                </el-tag>
              </div>
              <div class="history-remark">碰撞检测创建</div>
              <div class="history-meta">
                <span>创建时间: {{ formatDateTime(currentHistoryResult.createdAt) }}</span>
              </div>
            </div>
          </el-timeline-item>
        </el-timeline>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useModelStore } from '../../stores/model'
import { useCollisionStore } from '../../stores/collision'
import { useViewerStore } from '../../stores/viewer'
import { ElMessage } from 'element-plus'
import {
  ArrowDown, Clock, WarningFilled, CircleCheckFilled, Check,
  Histogram, ArrowRight
} from '@element-plus/icons-vue'

const props = defineProps({
  renderer: Object,
  modelId: String
})

const modelStore = useModelStore()
const collisionStore = useCollisionStore()
const viewerStore = useViewerStore()

const statusDialogVisible = ref(false)
const statusDialogTitle = ref('')
const statusForm = ref({
  newStatus: '',
  remark: '',
  resultIds: [],
  isBatch: false
})

const historyDialogVisible = ref(false)
const currentHistoryResult = ref(null)
const currentHistory = computed(() => {
  if (!currentHistoryResult.value) return []
  return collisionStore.historyMap[currentHistoryResult.value.id] || []
})

const canDetect = computed(() => {
  return collisionStore.groupA.length > 0 && collisionStore.groupB.length > 0 && props.modelId
})

watch(() => props.modelId, (newModelId) => {
  if (newModelId) {
    loadExistingResults(newModelId)
  }
}, { immediate: true })

async function loadExistingResults(modelId) {
  try {
    await collisionStore.fetchResultsByModel(modelId)
  } catch (err) {
    console.error('Failed to load existing results:', err)
  }
}

function getElementName(id) {
  const el = modelStore.elements.find(e => e.id === id)
  return el ? (el.name || el.ifcGuid || id.slice(-8)) : id.slice(-8)
}

async function runDetection() {
  try {
    const result = await collisionStore.detectCollisions(props.modelId)
    ElMessage.success(`检测完成，发现 ${result.count} 处碰撞`)
  } catch (err) {
    ElMessage.error('碰撞检测失败: ' + err.message)
  }
}

function onRowClick(row) {
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
  let className = `severity-${row.severity}`
  if (row.status === 'resolved') {
    className += ' status-resolved'
  } else if (row.status === 'false_positive') {
    className += ' status-false'
  }
  return className
}

function handleStatusChange(row, newStatus) {
  statusForm.value = {
    newStatus,
    remark: '',
    resultIds: [row.id],
    isBatch: false
  }
  statusDialogTitle.value = `标记为"${collisionStore.STATUS_LABELS[newStatus]}"`
  statusDialogVisible.value = true
}

function handleBatchStatusChange(newStatus) {
  statusForm.value = {
    newStatus,
    remark: '',
    resultIds: [...collisionStore.selectedIds],
    isBatch: true
  }
  statusDialogTitle.value = `批量标记 ${collisionStore.selectedIds.length} 条记录为"${collisionStore.STATUS_LABELS[newStatus]}"`
  statusDialogVisible.value = true
}

async function confirmStatusChange() {
  try {
    if (statusForm.value.isBatch) {
      await collisionStore.batchUpdateStatus(
        statusForm.value.newStatus,
        statusForm.value.remark.trim()
      )
      ElMessage.success('批量状态更新成功')
    } else {
      await collisionStore.updateStatus(
        statusForm.value.resultIds[0],
        statusForm.value.newStatus,
        statusForm.value.remark.trim()
      )
      ElMessage.success('状态更新成功')
    }
    statusDialogVisible.value = false
    statusForm.value.remark = ''
  } catch (err) {
    ElMessage.error('状态更新失败: ' + err.message)
  }
}

async function showHistory(row) {
  currentHistoryResult.value = row
  historyDialogVisible.value = true
  try {
    await collisionStore.fetchHistory(row.id)
  } catch (err) {
    ElMessage.error('加载历史记录失败: ' + err.message)
  }
}

function getTimelineType(status) {
  const typeMap = {
    pending: 'info',
    confirmed: 'danger',
    false_positive: 'success',
    resolved: 'primary'
  }
  return typeMap[status] || 'info'
}

function formatDateTime(dateStr) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

async function exportReport() {
  if (collisionStore.currentTaskId) {
    try {
      await collisionStore.exportCSV(collisionStore.currentTaskId)
      ElMessage.success('碰撞报告已导出')
    } catch (err) {
      ElMessage.error('导出失败: ' + err.message)
    }
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

.stats-bar {
  display: flex;
  padding: 8px 12px;
  gap: 8px;
  border-bottom: 1px solid #2a2a4a;
  background: rgba(0, 0, 0, 0.2);
}

.stat-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 6px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
}

.stat-item:hover {
  background: rgba(255, 255, 255, 0.05);
}

.stat-item.active {
  background: rgba(64, 158, 255, 0.1);
  border-color: #409eff;
}

.stat-item.pending.active {
  background: rgba(144, 147, 153, 0.1);
  border-color: #909399;
}

.stat-item.confirmed.active {
  background: rgba(245, 108, 108, 0.1);
  border-color: #f56c6c;
}

.stat-item.false.active {
  background: rgba(103, 194, 58, 0.1);
  border-color: #67c23a;
}

.stat-item.resolved.active {
  background: rgba(64, 158, 255, 0.1);
  border-color: #409eff;
}

.stat-label {
  color: #8899aa;
  font-size: 11px;
  margin-bottom: 2px;
}

.stat-count {
  color: #ccddee;
  font-size: 16px;
  font-weight: 600;
}

.stat-item.pending .stat-count {
  color: #909399;
}

.stat-item.confirmed .stat-count {
  color: #f56c6c;
}

.stat-item.false .stat-count {
  color: #67c23a;
}

.stat-item.resolved .stat-count {
  color: #409eff;
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

.batch-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.selected-count {
  color: #409eff;
  font-weight: 500;
}

:deep(.severity-high) {
  background: rgba(255, 0, 0, 0.1);
}

:deep(.severity-medium) {
  background: rgba(255, 200, 0, 0.05);
}

:deep(.status-resolved) {
  opacity: 0.6;
}

:deep(.status-resolved td) {
  text-decoration: line-through;
}

:deep(.status-false) {
  opacity: 0.7;
  background: rgba(103, 194, 58, 0.05) !important;
}

.history-header {
  padding: 12px;
  background: rgba(0, 0, 0, 0.05);
  border-radius: 4px;
  margin-bottom: 16px;
}

.history-item-info {
  display: flex;
  align-items: center;
  margin-bottom: 6px;
  font-size: 13px;
}

.history-item-info:last-child {
  margin-bottom: 0;
}

.history-item-info .label {
  width: 80px;
  color: #909399;
}

.timeline-container {
  max-height: 400px;
  overflow-y: auto;
}

.history-content {
  padding: 8px 0;
}

.history-status {
  display: flex;
  align-items: center;
  margin-bottom: 6px;
}

.status-transition {
  display: flex;
  align-items: center;
  gap: 6px;
}

.arrow-icon {
  color: #909399;
  font-size: 16px;
}

.history-remark {
  color: #606266;
  font-size: 13px;
  margin-bottom: 4px;
  line-height: 1.5;
}

.history-meta {
  color: #909399;
  font-size: 12px;
}
</style>
