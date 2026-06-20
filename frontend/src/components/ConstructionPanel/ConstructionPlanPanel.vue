<template>
  <div class="construction-panel">
    <div class="panel-header">
      <span class="panel-title">施工4D模拟</span>
      <el-button size="small" type="primary" @click="showCreatePlan = true" v-if="!constructionStore.currentPlan" :disabled="!canCreatePlan">
        新建计划
      </el-button>
      <el-button size="small" @click="backToList" v-if="constructionStore.currentPlan">
        返回列表
      </el-button>
    </div>

    <div class="panel-body" v-if="!constructionStore.currentPlan">
      <div v-if="constructionStore.loading" class="loading-hint">加载中...</div>
      <div v-else-if="constructionStore.plans.length === 0" class="empty-hint">暂无施工计划，请点击新建</div>
      <div
        v-for="plan in constructionStore.plans"
        :key="plan.id"
        class="plan-card"
        @click="selectPlan(plan)"
      >
        <div class="plan-name">{{ plan.name }}</div>
        <div class="plan-dates">{{ plan.startDate }} ~ {{ plan.endDate }}</div>
        <div class="plan-meta">{{ (plan.phases || []).length }} 个阶段</div>
        <el-button
          size="small"
          type="danger"
          circle
          class="plan-delete"
          @click.stop="handleDeletePlan(plan.id)"
        >
          <el-icon><Delete /></el-icon>
        </el-button>
      </div>
    </div>

    <div class="plan-detail" v-if="constructionStore.currentPlan">
      <div class="detail-header">
        <span class="detail-name">{{ constructionStore.currentPlan.name }}</span>
        <el-button size="small" @click="showEditPlan = true">编辑</el-button>
      </div>
      <div class="detail-dates">
        {{ constructionStore.currentPlan.startDate }} ~ {{ constructionStore.currentPlan.endDate }}
      </div>

      <div class="phases-section">
        <div class="section-header">
          <span>施工阶段</span>
          <el-button size="small" type="primary" @click="openAddPhase">添加阶段</el-button>
        </div>

        <div v-if="constructionStore.allPhases.length === 0" class="empty-hint">
          暂无阶段，请点击添加
        </div>

        <div
          v-for="(phase, idx) in constructionStore.allPhases"
          :key="phase.id"
          class="phase-item"
        >
          <div class="phase-color" :style="{ background: phase.color || constructionStore.PHASE_COLORS[idx % constructionStore.PHASE_COLORS.length] }" />
          <div class="phase-info">
            <div class="phase-name">{{ phase.name }}</div>
            <div class="phase-dates">{{ phase.startDate }} ~ {{ phase.endDate }}</div>
            <div class="phase-elements">{{ (phase.elementIds || []).length }} 个构件</div>
          </div>
          <div class="phase-actions">
            <el-button size="small" circle @click="openEditPhase(phase)">
              <el-icon><Edit /></el-icon>
            </el-button>
            <el-button size="small" circle @click="openPickElements(phase)">
              <el-icon><Pointer /></el-icon>
            </el-button>
            <el-button size="small" type="danger" circle @click="handleDeletePhase(phase.id)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </div>
      </div>

      <div class="gantt-section" v-if="constructionStore.allPhases.length > 0">
        <div class="section-header">
          <span>甘特图预览</span>
        </div>
        <div class="gantt-chart" ref="ganttRef">
          <div class="gantt-rows">
            <div v-for="(phase, idx) in constructionStore.allPhases" :key="phase.id" class="gantt-row">
              <div class="gantt-label">{{ phase.name }}</div>
              <div class="gantt-bar-area" @click="onGanttBarClick(phase)">
                <div
                  class="gantt-bar"
                  :style="{
                    left: getBarLeft(phase) + '%',
                    width: getBarWidth(phase) + '%',
                    background: phase.color || constructionStore.PHASE_COLORS[idx % constructionStore.PHASE_COLORS.length]
                  }"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="playback-section">
        <div class="section-header">
          <span>4D播放</span>
          <el-button
            v-if="!constructionStore.playbackActive"
            size="small"
            type="success"
            @click="startPlayback"
            :disabled="constructionStore.allPhases.length === 0"
          >
            开始播放
          </el-button>
          <el-button
            v-else
            size="small"
            type="danger"
            @click="stopPlayback"
          >
            退出播放
          </el-button>
        </div>
      </div>
    </div>

    <el-dialog v-model="showCreatePlan" title="新建施工计划" width="420px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="newPlan.name" placeholder="请输入计划名称" />
        </el-form-item>
        <el-form-item label="开始日期">
          <el-date-picker v-model="newPlan.startDate" type="date" value-format="YYYY-MM-DD" placeholder="选择开始日期" style="width: 100%" />
        </el-form-item>
        <el-form-item label="结束日期">
          <el-date-picker v-model="newPlan.endDate" type="date" value-format="YYYY-MM-DD" placeholder="选择结束日期" style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreatePlan = false">取消</el-button>
        <el-button type="primary" @click="handleCreatePlan" :loading="saving">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showEditPlan" title="编辑施工计划" width="420px" :close-on-click-modal="false">
      <el-form label-width="80px" v-if="constructionStore.currentPlan">
        <el-form-item label="名称">
          <el-input v-model="editPlanData.name" />
        </el-form-item>
        <el-form-item label="开始日期">
          <el-date-picker v-model="editPlanData.startDate" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
        <el-form-item label="结束日期">
          <el-date-picker v-model="editPlanData.endDate" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditPlan = false">取消</el-button>
        <el-button type="primary" @click="handleUpdatePlan" :loading="saving">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showPhaseForm" :title="editingPhase ? '编辑阶段' : '添加阶段'" width="460px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="phaseForm.name" placeholder="请输入阶段名称" />
        </el-form-item>
        <el-form-item label="开始日期">
          <el-date-picker v-model="phaseForm.startDate" type="date" value-format="YYYY-MM-DD" placeholder="选择开始日期" style="width: 100%" />
        </el-form-item>
        <el-form-item label="结束日期">
          <el-date-picker v-model="phaseForm.endDate" type="date" value-format="YYYY-MM-DD" placeholder="选择结束日期" style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPhaseForm = false">取消</el-button>
        <el-button type="primary" @click="handleSavePhase" :loading="saving">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showElementPicker" title="选择构件" width="500px" :close-on-click-modal="false">
      <div class="element-picker">
        <div class="picker-search">
          <el-input v-model="elementFilter" placeholder="搜索构件" size="small" clearable />
        </div>
        <div class="picker-tree">
          <div v-for="cat in filteredCategories" :key="cat.name" class="picker-category">
            <div class="category-header" @click="toggleCategory(cat.name)">
              <el-checkbox
                :model-value="isCategoryAllSelected(cat.name)"
                :indeterminate="isCategoryPartialSelected(cat.name)"
                @change="toggleCategoryAll(cat.name, $event)"
                @click.stop
              />
              <span class="category-name">{{ cat.name }} ({{ cat.elements.length }})</span>
              <el-icon class="category-arrow" :class="{ expanded: expandedCategories[cat.name] }">
                <ArrowRight />
              </el-icon>
            </div>
            <div v-show="expandedCategories[cat.name]" class="category-items">
              <div v-for="elem in cat.elements" :key="elem.id" class="picker-item">
                <el-checkbox
                  :model-value="pickedElementIds.has(elem.id)"
                  @change="toggleElement(elem.id, $event)"
                />
                <span class="elem-name">{{ elem.name || elem.ifcGuid || elem.id }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="picker-footer">
          已选 {{ pickedElementIds.size }} 个构件
        </div>
      </div>
      <template #footer>
        <el-button @click="showElementPicker = false">取消</el-button>
        <el-button type="primary" @click="saveElementPicks" :loading="saving">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useConstructionStore } from '../../stores/construction'
import { useModelStore } from '../../stores/model'
import { Delete, Edit, Pointer, ArrowRight } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const props = defineProps({
  modelId: { type: String, required: false, default: '' }
})

const route = useRoute()
const constructionStore = useConstructionStore()
const modelStore = useModelStore()

const resolvedModelId = computed(() => {
  return props.modelId || route.params.modelId || ''
})

const canCreatePlan = computed(() => !!resolvedModelId.value)

const showCreatePlan = ref(false)
const showEditPlan = ref(false)
const showPhaseForm = ref(false)
const showElementPicker = ref(false)
const saving = ref(false)
const editingPhase = ref(null)

const newPlan = reactive({ name: '', startDate: '', endDate: '' })
const editPlanData = reactive({ name: '', startDate: '', endDate: '' })
const phaseForm = reactive({ name: '', startDate: '', endDate: '' })

const pickedElementIds = ref(new Set())
const pickingPhaseId = ref(null)
const elementFilter = ref('')
const expandedCategories = reactive({})

onMounted(() => {
  if (resolvedModelId.value) {
    constructionStore.fetchPlans(resolvedModelId.value)
  }
})

watch(resolvedModelId, (val) => {
  if (val) constructionStore.fetchPlans(val)
})

function selectPlan(plan) {
  constructionStore.fetchPlan(plan.id)
}

function backToList() {
  constructionStore.stopPlayback()
  constructionStore.setCurrentPlan(null)
  constructionStore.fetchPlans(resolvedModelId.value)
}

async function handleCreatePlan() {
  if (!newPlan.name || !newPlan.startDate || !newPlan.endDate) {
    ElMessage.warning('请填写完整信息')
    return
  }
  if (!resolvedModelId.value) {
    ElMessage.error('模型ID不存在，请刷新页面重试')
    return
  }
  saving.value = true
  try {
    const plan = await constructionStore.createPlan({
      modelId: resolvedModelId.value,
      name: newPlan.name,
      startDate: newPlan.startDate,
      endDate: newPlan.endDate
    })
    showCreatePlan.value = false
    newPlan.name = ''
    newPlan.startDate = ''
    newPlan.endDate = ''
    constructionStore.setCurrentPlan(plan)
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '创建失败')
  } finally {
    saving.value = false
  }
}

function showEditPlanDialog() {
  if (constructionStore.currentPlan) {
    editPlanData.name = constructionStore.currentPlan.name
    editPlanData.startDate = constructionStore.currentPlan.startDate
    editPlanData.endDate = constructionStore.currentPlan.endDate
  }
  showEditPlan.value = true
}

watch(showEditPlan, (val) => {
  if (val) showEditPlanDialog()
})

async function handleUpdatePlan() {
  saving.value = true
  try {
    await constructionStore.updatePlan(constructionStore.currentPlan.id, {
      name: editPlanData.name,
      startDate: editPlanData.startDate,
      endDate: editPlanData.endDate
    })
    showEditPlan.value = false
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '更新失败')
  } finally {
    saving.value = false
  }
}

async function handleDeletePlan(planId) {
  try {
    await ElMessageBox.confirm('确定删除该施工计划？', '确认', { type: 'warning' })
    await constructionStore.deletePlan(planId)
  } catch {}
}

function openAddPhase() {
  editingPhase.value = null
  phaseForm.name = ''
  phaseForm.startDate = constructionStore.currentPlan?.startDate || ''
  phaseForm.endDate = constructionStore.currentPlan?.startDate || ''
  showPhaseForm.value = true
}

function openEditPhase(phase) {
  editingPhase.value = phase
  phaseForm.name = phase.name
  phaseForm.startDate = phase.startDate
  phaseForm.endDate = phase.endDate
  showPhaseForm.value = true
}

async function handleSavePhase() {
  if (!phaseForm.name || !phaseForm.startDate || !phaseForm.endDate) {
    ElMessage.warning('请填写完整阶段信息')
    return
  }
  saving.value = true
  try {
    const planId = constructionStore.currentPlan.id
    if (editingPhase.value) {
      await constructionStore.updatePhase(planId, editingPhase.value.id, {
        name: phaseForm.name,
        startDate: phaseForm.startDate,
        endDate: phaseForm.endDate
      })
    } else {
      await constructionStore.createPhase(planId, {
        name: phaseForm.name,
        startDate: phaseForm.startDate,
        endDate: phaseForm.endDate,
        elementIds: []
      })
    }
    showPhaseForm.value = false
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDeletePhase(phaseId) {
  try {
    await ElMessageBox.confirm('确定删除该阶段？', '确认', { type: 'warning' })
    await constructionStore.deletePhase(constructionStore.currentPlan.id, phaseId)
  } catch {}
}

function openPickElements(phase) {
  pickingPhaseId.value = phase.id
  pickedElementIds.value = new Set(phase.elementIds || [])
  elementFilter.value = ''
  showElementPicker.value = true
}

async function saveElementPicks() {
  saving.value = true
  try {
    await constructionStore.updatePhase(
      constructionStore.currentPlan.id,
      pickingPhaseId.value,
      { elementIds: [...pickedElementIds.value] }
    )
    showElementPicker.value = false
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

function toggleElement(elementId, checked) {
  const newSet = new Set(pickedElementIds.value)
  if (checked) {
    newSet.add(elementId)
  } else {
    newSet.delete(elementId)
  }
  pickedElementIds.value = newSet
}

function toggleCategory(catName) {
  expandedCategories[catName] = !expandedCategories[catName]
}

function isCategoryAllSelected(catName) {
  const cat = filteredCategories.value.find(c => c.name === catName)
  if (!cat || cat.elements.length === 0) return false
  return cat.elements.every(e => pickedElementIds.value.has(e.id))
}

function isCategoryPartialSelected(catName) {
  const cat = filteredCategories.value.find(c => c.name === catName)
  if (!cat || cat.elements.length === 0) return false
  const some = cat.elements.some(e => pickedElementIds.value.has(e.id))
  const all = cat.elements.every(e => pickedElementIds.value.has(e.id))
  return some && !all
}

function toggleCategoryAll(catName, checked) {
  const cat = filteredCategories.value.find(c => c.name === catName)
  if (!cat) return
  const newSet = new Set(pickedElementIds.value)
  for (const e of cat.elements) {
    if (checked) {
      newSet.add(e.id)
    } else {
      newSet.delete(e.id)
    }
  }
  pickedElementIds.value = newSet
}

const filteredCategories = computed(() => {
  const categories = modelStore.elementsByCategory
  const result = []
  for (const [name, elems] of Object.entries(categories)) {
    let filtered = elems
    if (elementFilter.value) {
      const keyword = elementFilter.value.toLowerCase()
      filtered = elems.filter(e =>
        (e.name && e.name.toLowerCase().includes(keyword)) ||
        (e.ifcGuid && e.ifcGuid.toLowerCase().includes(keyword)) ||
        e.id.toLowerCase().includes(keyword)
      )
    }
    if (filtered.length > 0) {
      result.push({ name, elements: filtered })
    }
  }
  return result
})

function getBarLeft(phase) {
  const plan = constructionStore.currentPlan
  if (!plan) return 0
  const start = new Date(plan.startDate).getTime()
  const end = new Date(plan.endDate).getTime()
  const phaseStart = new Date(phase.startDate).getTime()
  return ((phaseStart - start) / (end - start)) * 100
}

function getBarWidth(phase) {
  const plan = constructionStore.currentPlan
  if (!plan) return 0
  const start = new Date(plan.startDate).getTime()
  const end = new Date(plan.endDate).getTime()
  const phaseStart = new Date(phase.startDate).getTime()
  const phaseEnd = new Date(phase.endDate).getTime()
  return ((phaseEnd - phaseStart) / (end - start)) * 100
}

function onGanttBarClick(phase) {
  if (constructionStore.playbackActive) {
    constructionStore.seekToDate(phase.startDate)
  }
}

function startPlayback() {
  if (!constructionStore.currentPlan) return
  constructionStore.setCurrentPlan(constructionStore.currentPlan)
  constructionStore.startPlayback()
}

function stopPlayback() {
  constructionStore.stopPlayback()
}
</script>

<style scoped>
.construction-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  color: #ccddee;
  font-size: 13px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid #2a2a4a;
  background: #1a1a2e;
}

.panel-title {
  font-weight: 600;
  font-size: 14px;
  color: #ffffff;
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.loading-hint,
.empty-hint {
  text-align: center;
  color: #667788;
  padding: 24px 0;
  font-size: 12px;
}

.plan-card {
  padding: 10px 12px;
  margin-bottom: 6px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid #2a2a4a;
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  transition: background 0.2s;
}

.plan-card:hover {
  background: rgba(64, 128, 255, 0.1);
}

.plan-name {
  font-weight: 600;
  color: #ffffff;
  margin-bottom: 4px;
}

.plan-dates {
  font-size: 11px;
  color: #8899aa;
}

.plan-meta {
  font-size: 11px;
  color: #667788;
  margin-top: 2px;
}

.plan-delete {
  position: absolute;
  top: 8px;
  right: 8px;
  opacity: 0;
  transition: opacity 0.2s;
}

.plan-card:hover .plan-delete {
  opacity: 1;
}

.plan-detail {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid #2a2a4a;
}

.detail-name {
  font-weight: 600;
  font-size: 14px;
  color: #ffffff;
}

.detail-dates {
  font-size: 11px;
  color: #8899aa;
  padding: 0 12px 6px;
}

.phases-section,
.gantt-section,
.playback-section {
  padding: 8px 12px;
  border-top: 1px solid #2a2a4a;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 600;
  color: #aabbcc;
}

.phase-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  margin-bottom: 4px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 4px;
}

.phase-color {
  width: 4px;
  height: 32px;
  border-radius: 2px;
  flex-shrink: 0;
}

.phase-info {
  flex: 1;
  min-width: 0;
}

.phase-name {
  font-size: 12px;
  color: #ffffff;
  font-weight: 500;
}

.phase-dates {
  font-size: 10px;
  color: #8899aa;
}

.phase-elements {
  font-size: 10px;
  color: #667788;
}

.phase-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
}

.gantt-chart {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  overflow: hidden;
}

.gantt-row {
  display: flex;
  align-items: center;
  height: 28px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.gantt-label {
  width: 80px;
  font-size: 10px;
  color: #8899aa;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 0 4px;
  flex-shrink: 0;
}

.gantt-bar-area {
  flex: 1;
  position: relative;
  height: 100%;
  cursor: pointer;
}

.gantt-bar {
  position: absolute;
  top: 4px;
  height: 20px;
  border-radius: 3px;
  min-width: 2px;
  opacity: 0.85;
  transition: opacity 0.2s;
}

.gantt-bar-area:hover .gantt-bar {
  opacity: 1;
}

.element-picker {
  display: flex;
  flex-direction: column;
  height: 400px;
}

.picker-search {
  padding-bottom: 8px;
  border-bottom: 1px solid #eee;
}

.picker-tree {
  flex: 1;
  overflow-y: auto;
  margin-top: 8px;
}

.picker-category {
  margin-bottom: 2px;
}

.category-header {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 0;
  cursor: pointer;
  user-select: none;
}

.category-name {
  font-size: 13px;
  font-weight: 500;
}

.category-arrow {
  margin-left: auto;
  transition: transform 0.2s;
}

.category-arrow.expanded {
  transform: rotate(90deg);
}

.category-items {
  padding-left: 24px;
}

.picker-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 2px 0;
}

.elem-name {
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.picker-footer {
  padding-top: 8px;
  border-top: 1px solid #eee;
  font-size: 12px;
  color: #999;
}
</style>
