<template>
  <div class="version-compare-panel">
    <div class="panel-header">
      <span>版本对比</span>
      <el-button
        v-if="versionStore.compareMode"
        size="small"
        type="danger"
        text
        @click="handleExitCompare"
      >
        退出对比
      </el-button>
    </div>

    <div v-if="!versionStore.compareMode" class="version-config">
      <div class="config-section">
        <div class="section-title">创建新版本</div>
        <el-input
          v-model="newVersionDesc"
          placeholder="输入版本备注说明..."
          size="small"
          type="textarea"
          :rows="2"
          maxlength="200"
          show-word-limit
        />
        <el-button
          type="primary"
          size="small"
          style="width: 100%; margin-top: 8px"
          :loading="versionStore.creating"
          :disabled="!newVersionDesc.trim()"
          @click="handleCreateVersion"
        >
          创建当前版本快照
        </el-button>
      </div>

      <el-divider />

      <div class="config-section">
        <div class="section-title">版本对比</div>
        
        <div class="version-select-group">
          <div class="select-label">基准版本</div>
          <el-select
            v-model="baseVersionId"
            placeholder="选择基准版本"
            size="small"
            style="width: 100%"
            :loading="versionStore.loading"
          >
            <el-option
              v-for="v in versionStore.versions"
              :key="v.id"
              :label="`${v.versionNumber} - ${v.description || '无备注'}`"
              :value="v.id"
            >
              <div class="version-option">
                <span class="version-tag">{{ v.versionNumber }}</span>
                <span class="version-desc">{{ v.description || '无备注' }}</span>
                <span class="version-date">{{ formatDate(v.createdAt) }}</span>
              </div>
            </el-option>
          </el-select>
        </div>

        <div class="version-select-group">
          <div class="select-label">对比版本</div>
          <el-select
            v-model="compareVersionId"
            placeholder="选择对比版本"
            size="small"
            style="width: 100%"
            :loading="versionStore.loading"
          >
            <el-option
              v-for="v in versionStore.versions"
              :key="v.id"
              :label="`${v.versionNumber} - ${v.description || '无备注'}`"
              :value="v.id"
            >
              <div class="version-option">
                <span class="version-tag">{{ v.versionNumber }}</span>
                <span class="version-desc">{{ v.description || '无备注' }}</span>
                <span class="version-date">{{ formatDate(v.createdAt) }}</span>
              </div>
            </el-option>
          </el-select>
        </div>

        <el-button
          type="success"
          size="small"
          style="width: 100%; margin-top: 12px"
          :loading="versionStore.loading"
          :disabled="!canCompare"
          @click="handleCompare"
        >
          开始对比
        </el-button>
      </div>

      <el-divider v-if="versionStore.versions.length > 0" />

      <div v-if="versionStore.versions.length > 0" class="version-list">
        <div class="section-title">历史版本</div>
        <div class="version-list-content">
          <div
            v-for="v in versionStore.versions"
            :key="v.id"
            class="version-item"
          >
            <div class="version-item-header">
              <el-tag size="small" type="primary">{{ v.versionNumber }}</el-tag>
              <span class="version-item-date">{{ formatDate(v.createdAt) }}</span>
            </div>
            <div class="version-item-desc">{{ v.description || '无备注' }}</div>
            <div class="version-item-stats">
              <el-tag size="small" type="info">
                {{ Object.keys(v.elementSnapshot || {}).length }} 个构件
              </el-tag>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="compare-results">
      <div class="compare-versions-info">
        <div class="version-info-item base">
          <div class="version-label">基准版本</div>
          <div class="version-number">{{ versionStore.currentBaseVersion?.versionNumber }}</div>
          <div class="version-desc">{{ versionStore.currentBaseVersion?.description || '无备注' }}</div>
        </div>
        <div class="compare-arrow">
          <el-icon><ArrowRight /></el-icon>
        </div>
        <div class="version-info-item compare">
          <div class="version-label">对比版本</div>
          <div class="version-number">{{ versionStore.currentCompareVersion?.versionNumber }}</div>
          <div class="version-desc">{{ versionStore.currentCompareVersion?.description || '无备注' }}</div>
        </div>
      </div>

      <div class="diff-stats-bar">
        <div
          class="stat-item added"
          :class="{ active: versionStore.filterType === 'added' }"
          @click="handleFilterClick('added')"
        >
          <span class="stat-color" :style="{ background: versionStore.DIFF_COLORS_CSS.added }" />
          <span class="stat-label">新增</span>
          <span class="stat-count">{{ versionStore.stats.added }}</span>
        </div>
        <div
          class="stat-item removed"
          :class="{ active: versionStore.filterType === 'removed' }"
          @click="handleFilterClick('removed')"
        >
          <span class="stat-color" :style="{ background: versionStore.DIFF_COLORS_CSS.removed }" />
          <span class="stat-label">删除</span>
          <span class="stat-count">{{ versionStore.stats.removed }}</span>
        </div>
        <div
          class="stat-item modified"
          :class="{ active: versionStore.filterType === 'modified' }"
          @click="handleFilterClick('modified')"
        >
          <span class="stat-color" :style="{ background: versionStore.DIFF_COLORS_CSS.modified }" />
          <span class="stat-label">修改</span>
          <span class="stat-count">{{ versionStore.stats.modified }}</span>
        </div>
        <div
          class="stat-item unchanged"
          :class="{ active: versionStore.filterType === 'unchanged' }"
          @click="handleFilterClick('unchanged')"
        >
          <span class="stat-color" :style="{ background: versionStore.DIFF_COLORS_CSS.unchanged }" />
          <span class="stat-label">未变</span>
          <span class="stat-count">{{ versionStore.stats.unchanged }}</span>
        </div>
      </div>

      <div class="diff-summary">
        总共 <strong>{{ versionStore.stats.total }}</strong> 个构件，
        其中变更 <strong>{{ versionStore.stats.added + versionStore.stats.removed + versionStore.stats.modified }}</strong> 个
      </div>

      <el-button
        size="small"
        style="width: 100%; margin: 8px 0"
        @click="handleFilterClick('all')"
      >
        显示全部类型
      </el-button>

      <div class="diff-elements-list">
        <div class="list-header">差异构件列表</div>
        <div class="list-content" v-loading="versionStore.loading">
          <div
            v-for="elementId in versionStore.filteredElementIds"
            :key="elementId"
            class="diff-element-item"
            :class="{ selected: versionStore.selectedElementId === elementId }"
            @click="handleElementClick(elementId)"
          >
            <span
              class="diff-type-dot"
              :style="{ background: versionStore.DIFF_COLORS_CSS[versionStore.getElementDiffType(elementId)] }"
            />
            <span class="element-name">{{ getElementName(elementId) }}</span>
            <span v-if="versionStore.annotatedElementIds.has(elementId)" class="annotation-badge" title="有批注">
              <el-icon><ChatDotRound /></el-icon>
            </span>
            <el-tag
              size="small"
              :type="getDiffTagType(versionStore.getElementDiffType(elementId))"
            >
              {{ getDiffLabel(versionStore.getElementDiffType(elementId)) }}
            </el-tag>
          </div>
          <div v-if="versionStore.filteredElementIds.length === 0" class="empty-list">
            暂无该类型的差异构件
          </div>
        </div>
      </div>

      <div class="annotations-section">
        <el-collapse v-model="annotationCollapse">
          <el-collapse-item name="annotations">
            <template #title>
              <div class="annotation-collapse-title">
                <span>批注列表</span>
                <el-tag size="small" type="warning">{{ versionStore.annotations.length }}</el-tag>
              </div>
            </template>
            <div class="annotation-list" v-loading="versionStore.annotationsLoading">
              <div
                v-for="ann in versionStore.annotations"
                :key="ann.id"
                class="annotation-item"
              >
                <div class="annotation-item-header">
                  <span
                    class="diff-type-dot"
                    :style="{ background: versionStore.DIFF_COLORS_CSS[versionStore.getElementDiffType(ann.elementId)] }"
                  />
                  <span class="annotation-element-name" @click="handleAnnotationJump(ann.elementId)">
                    {{ getElementName(ann.elementId) }}
                  </span>
                  <el-button
                    v-if="canDeleteAnnotation(ann)"
                    class="annotation-delete-btn"
                    size="small"
                    type="danger"
                    text
                    @click.stop="handleDeleteAnnotation(ann.id)"
                  >
                    删除
                  </el-button>
                </div>
                <div class="annotation-content">{{ ann.content }}</div>
                <div class="annotation-meta">
                  <span class="annotation-author">{{ ann.author }}</span>
                  <span class="annotation-time">{{ formatDate(ann.createdAt) }}</span>
                </div>
              </div>
              <div v-if="versionStore.annotations.length === 0" class="empty-list">
                暂无批注
              </div>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>

      <div class="export-section">
        <el-button
          type="primary"
          size="small"
          style="width: 100%"
          :loading="exportingReport"
          @click="handleExportReport"
        >
          <el-icon style="margin-right: 4px"><Download /></el-icon>
          导出对比报告
        </el-button>
      </div>
    </div>

    <el-drawer
      v-model="propertyCompareVisible"
      title="属性对比"
      direction="rtl"
      size="500px"
    >
      <div v-if="selectedElementForCompare" class="property-compare">
        <div class="element-header">
          <div class="element-title">{{ getElementName(selectedElementForCompare) }}</div>
          <el-tag
            size="small"
            :type="getDiffTagType(versionStore.getElementDiffType(selectedElementForCompare))"
          >
            {{ getDiffLabel(versionStore.getElementDiffType(selectedElementForCompare)) }}
          </el-tag>
        </div>

        <div class="compare-headers">
          <div class="compare-header base">
            <div class="header-version">{{ versionStore.currentBaseVersion?.versionNumber }}</div>
            <div class="header-desc">{{ versionStore.currentBaseVersion?.description || '无备注' }}</div>
          </div>
          <div class="compare-header compare">
            <div class="header-version">{{ versionStore.currentCompareVersion?.versionNumber }}</div>
            <div class="header-desc">{{ versionStore.currentCompareVersion?.description || '无备注' }}</div>
          </div>
        </div>

        <div v-if="loadingProperties" class="loading-properties">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>加载属性中...</span>
        </div>

        <div v-else class="properties-table">
          <div
            v-for="(row, index) in propertyRows"
            :key="index"
            class="property-row"
            :class="{ modified: row.modified, 'only-base': row.onlyBase, 'only-compare': row.onlyCompare }"
          >
            <div class="property-name">{{ row.name }}</div>
            <div class="property-value base">
              <span v-if="row.onlyCompare" class="missing-value">-</span>
              <span v-else>{{ formatValue(row.baseValue) }}</span>
            </div>
            <div class="property-value compare">
              <span v-if="row.onlyBase" class="missing-value">-</span>
              <span v-else>{{ formatValue(row.compareValue) }}</span>
            </div>
          </div>
        </div>

        <div class="annotation-section">
          <div class="annotation-section-title">
            <el-icon><ChatDotRound /></el-icon>
            <span>变更批注</span>
          </div>

          <div v-if="elementAnnotations.length > 0" class="existing-annotations">
            <div
              v-for="ann in elementAnnotations"
              :key="ann.id"
              class="existing-annotation-item"
            >
              <div class="existing-annotation-header">
                <span class="existing-annotation-author">{{ ann.author }}</span>
                <span class="existing-annotation-time">{{ formatDate(ann.createdAt) }}</span>
                <el-button
                  v-if="canDeleteAnnotation(ann)"
                  size="small"
                  type="danger"
                  text
                  @click="handleDeleteAnnotation(ann.id)"
                >
                  删除
                </el-button>
              </div>
              <div class="existing-annotation-content">{{ ann.content }}</div>
            </div>
          </div>

          <div class="annotation-input-area">
            <el-input
              v-model="newAnnotationContent"
              type="textarea"
              :rows="3"
              placeholder="输入批注内容（最大500字符）..."
              maxlength="500"
              show-word-limit
              resize="none"
            />
            <el-button
              type="primary"
              size="small"
              style="margin-top: 8px"
              :loading="submittingAnnotation"
              :disabled="!newAnnotationContent.trim() || newAnnotationContent.trim().length > 500"
              @click="handleSubmitAnnotation"
            >
              提交批注
            </el-button>
          </div>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useVersionStore } from '../../stores/version'
import { useModelStore } from '../../stores/model'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowRight, Loading, ChatDotRound, Download } from '@element-plus/icons-vue'
import { getCurrentUsername } from '../../utils/api'

const props = defineProps({
  renderer: Object,
  modelId: String
})

const versionStore = useVersionStore()
const modelStore = useModelStore()

const newVersionDesc = ref('')
const baseVersionId = ref('')
const compareVersionId = ref('')
const propertyCompareVisible = ref(false)
const selectedElementForCompare = ref(null)
const loadingProperties = ref(false)
const baseElementProperties = ref(null)
const compareElementProperties = ref(null)
const newAnnotationContent = ref('')
const submittingAnnotation = ref(false)
const exportingReport = ref(false)
const annotationCollapse = ref([])

const elementAnnotations = computed(() => {
  if (!selectedElementForCompare.value) return []
  return versionStore.annotationsByElement.value[selectedElementForCompare.value] || []
})

const canCompare = computed(() => {
  return baseVersionId.value && compareVersionId.value && baseVersionId.value !== compareVersionId.value
})

const propertyRows = computed(() => {
  if (!baseElementProperties.value && !compareElementProperties.value) return []

  const baseProps = baseElementProperties.value?.properties || {}
  const compareProps = compareElementProperties.value?.properties || {}
  const baseBasic = getBasicProperties(baseElementProperties.value)
  const compareBasic = getBasicProperties(compareElementProperties.value)

  const allBase = { ...baseBasic, ...baseProps }
  const allCompare = { ...compareBasic, ...compareProps }

  const allKeys = new Set([...Object.keys(allBase), ...Object.keys(allCompare)])
  const rows = []

  for (const key of allKeys) {
    const baseVal = allBase[key]
    const compareVal = allCompare[key]
    const baseStr = String(baseVal ?? '')
    const compareStr = String(compareVal ?? '')
    const modified = baseStr !== compareStr

    rows.push({
      name: key,
      baseValue: baseVal,
      compareValue: compareVal,
      modified,
      onlyBase: key in allBase && !(key in allCompare),
      onlyCompare: key in allCompare && !(key in allBase)
    })
  }

  rows.sort((a, b) => {
    if (a.modified && !b.modified) return -1
    if (!a.modified && b.modified) return 1
    return a.name.localeCompare(b.name)
  })

  return rows
})

function getBasicProperties(element) {
  if (!element) return {}
  return {
    '名称': element.name,
    '类型': element.type,
    '分类': element.category,
    'IFC GUID': element.ifcGuid,
    '楼层': element.floorName,
    '几何哈希': element.geometryHash
  }
}

onMounted(() => {
  if (props.modelId) {
    loadVersions()
  }
})

watch(() => props.modelId, (newModelId) => {
  if (newModelId) {
    loadVersions()
  }
})

async function loadVersions() {
  try {
    await versionStore.fetchVersions(props.modelId)
  } catch (err) {
    ElMessage.error('加载版本列表失败')
  }
}

async function handleCreateVersion() {
  if (!newVersionDesc.value.trim()) {
    ElMessage.warning('请输入版本备注')
    return
  }
  try {
    const version = await versionStore.createVersion(props.modelId, newVersionDesc.value.trim())
    ElMessage.success(`版本 ${version.versionNumber} 创建成功`)
    newVersionDesc.value = ''
  } catch (err) {
    ElMessage.error('创建版本失败: ' + err.message)
  }
}

async function handleCompare() {
  if (!canCompare.value) {
    ElMessage.warning('请选择两个不同的版本进行对比')
    return
  }
  try {
    const result = await versionStore.compareVersions(props.modelId, baseVersionId.value, compareVersionId.value)
    
    if (props.renderer) {
      props.renderer.setCompareMode(true, versionStore.elementDiffMap)
    }
    
    const diffCount = result.diff.added.length + result.diff.removed.length + result.diff.modified.length
    ElMessage.success(`对比完成，发现 ${diffCount} 处变更`)

    try {
      await versionStore.fetchAnnotations()
      if (props.renderer) {
        props.renderer.updateVersionAnnotationPins(Array.from(versionStore.annotatedElementIds))
      }
    } catch (e) {
      console.warn('加载批注失败:', e)
    }
  } catch (err) {
    ElMessage.error('对比失败: ' + err.message)
  }
}

function handleFilterClick(type) {
  versionStore.setFilterType(type)
  if (props.renderer) {
    props.renderer.setDiffFilter(type)
  }
}

function handleExitCompare() {
  ElMessageBox.confirm(
    '确定要退出对比模式吗？',
    '退出对比',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(() => {
    versionStore.exitCompareMode()
    if (props.renderer) {
      props.renderer.exitCompareMode()
    }
    baseVersionId.value = ''
    compareVersionId.value = ''
  }).catch(() => {})
}

function getElementName(elementId) {
  const el = modelStore.elements.find(e => e.id === elementId)
  if (el) {
    return el.name || el.type || el.ifcGuid || elementId.slice(-8)
  }
  return elementId.slice(-8)
}

function getDiffLabel(type) {
  const labels = {
    added: '新增',
    removed: '删除',
    modified: '修改',
    unchanged: '未变'
  }
  return labels[type] || type
}

function getDiffTagType(type) {
  const types = {
    added: 'success',
    removed: 'danger',
    modified: 'warning',
    unchanged: 'info'
  }
  return types[type] || 'info'
}

async function handleElementClick(elementId) {
  versionStore.selectElement(elementId)
  selectedElementForCompare.value = elementId
  propertyCompareVisible.value = true
  newAnnotationContent.value = ''
  
  if (props.renderer) {
    const color = versionStore.getElementDiffColor(elementId)
    if (color) {
      props.renderer.highlightElement(elementId, color)
    }
  }

  await loadElementProperties(elementId)
}

async function loadElementProperties(elementId) {
  loadingProperties.value = true
  baseElementProperties.value = null
  compareElementProperties.value = null
  
  try {
    const [baseEl, compareEl] = await Promise.all([
      versionStore.getVersionElement(versionStore.currentBaseVersion?.id, elementId),
      versionStore.getVersionElement(versionStore.currentCompareVersion?.id, elementId)
    ])
    baseElementProperties.value = baseEl
    compareElementProperties.value = compareEl
  } catch (err) {
    console.error('Failed to load element properties:', err)
  } finally {
    loadingProperties.value = false
  }
}

function formatValue(val) {
  if (val === null || val === undefined) return '-'
  if (typeof val === 'object') return JSON.stringify(val)
  return String(val)
}

function padZero(n) {
  return n < 10 ? '0' + n : '' + n
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  let date
  if (dateStr instanceof Date) {
    date = dateStr
  } else {
    if (typeof dateStr === 'string' && !dateStr.includes('T')) {
      dateStr = dateStr.replace(' ', 'T')
    }
    date = new Date(dateStr)
  }
  if (isNaN(date.getTime())) {
    return String(dateStr)
  }
  return `${date.getFullYear()}-${padZero(date.getMonth() + 1)}-${padZero(date.getDate())} ${padZero(date.getHours())}:${padZero(date.getMinutes())}`
}

function canDeleteAnnotation(annotation) {
  const currentUser = getCurrentUsername()
  if (!currentUser) return false
  return annotation.author === currentUser || annotation.author === 'anonymous'
}

async function handleSubmitAnnotation() {
  const content = newAnnotationContent.value.trim()
  if (!content) {
    ElMessage.warning('批注内容不能为空')
    return
  }
  if (content.length > 500) {
    ElMessage.warning('批注内容不能超过500字符')
    return
  }
  if (!selectedElementForCompare.value) {
    ElMessage.warning('请先选择构件')
    return
  }

  submittingAnnotation.value = true
  try {
    await versionStore.createAnnotation(selectedElementForCompare.value, content)
    ElMessage.success('批注添加成功')
    newAnnotationContent.value = ''
    if (props.renderer) {
      props.renderer.updateVersionAnnotationPins(Array.from(versionStore.annotatedElementIds))
    }
  } catch (err) {
    ElMessage.error('添加批注失败: ' + (err.response?.data?.error || err.message))
  } finally {
    submittingAnnotation.value = false
  }
}

async function handleDeleteAnnotation(annotationId) {
  try {
    await ElMessageBox.confirm('确定要删除这条批注吗？', '删除批注', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }

  try {
    await versionStore.deleteAnnotation(annotationId)
    ElMessage.success('批注已删除')
    if (props.renderer) {
      props.renderer.updateVersionAnnotationPins(Array.from(versionStore.annotatedElementIds))
    }
  } catch (err) {
    ElMessage.error('删除批注失败: ' + (err.response?.data?.error || err.message))
  }
}

function handleAnnotationJump(elementId) {
  versionStore.selectElement(elementId)
  if (props.renderer) {
    props.renderer.focusOnElement(elementId)
    const color = versionStore.getElementDiffColor(elementId)
    if (color) {
      props.renderer.highlightElement(elementId, color)
    }
  }
}

async function handleExportReport() {
  if (!versionStore.currentBaseVersion || !versionStore.currentCompareVersion) {
    ElMessage.warning('请先进行版本对比')
    return
  }

  exportingReport.value = true
  try {
    const report = await versionStore.generateCompareReport(props.modelId)

    const modelName = report.metaInfo?.modelName || 'model'
    const baseVer = report.metaInfo?.baseVersion || 'v1'
    const compareVer = report.metaInfo?.compareVersion || 'v2'
    const fileName = `${modelName}_${baseVer}-vs-${compareVer}_对比报告.json`

    const formattedJson = JSON.stringify(report, null, 2)
    const blob = new Blob([formattedJson], { type: 'application/json;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = fileName
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)

    ElMessage.success('报告导出成功')
  } catch (err) {
    ElMessage.error('导出报告失败: ' + (err.response?.data?.error || err.message))
  } finally {
    exportingReport.value = false
  }
}
</script>

<style scoped>
.version-compare-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.panel-header {
  padding: 8px 12px;
  border-bottom: 1px solid #2a2a4a;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.panel-header span {
  color: #ccddee;
  font-size: 13px;
  font-weight: 500;
}

.version-config {
  padding: 12px;
  overflow-y: auto;
  flex: 1;
}

.config-section {
  margin-bottom: 8px;
}

.section-title {
  color: #8899aa;
  font-size: 12px;
  margin-bottom: 8px;
  font-weight: 500;
}

.version-select-group {
  margin-bottom: 12px;
}

.select-label {
  color: #8899aa;
  font-size: 12px;
  margin-bottom: 4px;
}

.version-option {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.version-tag {
  font-weight: 600;
  color: #409eff;
}

.version-desc {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #606266;
}

.version-date {
  color: #909399;
  font-size: 11px;
}

.version-list-content {
  max-height: 200px;
  overflow-y: auto;
}

.version-item {
  padding: 8px;
  border: 1px solid #2a2a4a;
  border-radius: 4px;
  margin-bottom: 8px;
  background: rgba(0, 0, 0, 0.1);
}

.version-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.version-item-date {
  color: #909399;
  font-size: 11px;
}

.version-item-desc {
  color: #aabbcc;
  font-size: 12px;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.version-item-stats {
  display: flex;
  gap: 4px;
}

.compare-results {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.compare-versions-info {
  display: flex;
  align-items: center;
  padding: 12px;
  background: rgba(0, 0, 0, 0.2);
  border-bottom: 1px solid #2a2a4a;
  gap: 8px;
}

.version-info-item {
  flex: 1;
  text-align: center;
  padding: 8px;
  border-radius: 4px;
}

.version-info-item.base {
  background: rgba(245, 108, 108, 0.1);
  border: 1px solid rgba(245, 108, 108, 0.3);
}

.version-info-item.compare {
  background: rgba(103, 194, 58, 0.1);
  border: 1px solid rgba(103, 194, 58, 0.3);
}

.version-label {
  color: #8899aa;
  font-size: 11px;
  margin-bottom: 2px;
}

.version-number {
  color: #ffffff;
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 2px;
}

.version-desc {
  color: #aabbcc;
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compare-arrow {
  color: #409eff;
  font-size: 20px;
}

.diff-stats-bar {
  display: flex;
  padding: 8px 4px;
  gap: 4px;
  border-bottom: 1px solid #2a2a4a;
}

.stat-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 4px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
}

.stat-item:hover {
  background: rgba(255, 255, 255, 0.05);
}

.stat-item.active {
  background: rgba(64, 158, 255, 0.15);
  border-color: #409eff;
}

.stat-color {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  margin-bottom: 4px;
}

.stat-label {
  color: #8899aa;
  font-size: 11px;
  margin-bottom: 2px;
}

.stat-count {
  color: #ffffff;
  font-size: 16px;
  font-weight: 600;
}

.diff-summary {
  padding: 8px 12px;
  color: #aabbcc;
  font-size: 12px;
  text-align: center;
  border-bottom: 1px solid #2a2a4a;
}

.diff-summary strong {
  color: #409eff;
}

.diff-elements-list {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding: 0 12px 12px;
}

.list-header {
  color: #8899aa;
  font-size: 12px;
  padding: 8px 0;
  font-weight: 500;
}

.list-content {
  flex: 1;
  overflow-y: auto;
}

.diff-element-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
  margin-bottom: 4px;
  border: 1px solid transparent;
}

.diff-element-item:hover {
  background: rgba(255, 255, 255, 0.05);
}

.diff-element-item.selected {
  background: rgba(64, 158, 255, 0.1);
  border-color: #409eff;
}

.diff-type-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.element-name {
  flex: 1;
  color: #ccddee;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-list {
  color: #666666;
  font-size: 12px;
  text-align: center;
  padding: 20px;
}

.property-compare {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.element-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid #e4e7ed;
}

.element-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.compare-headers {
  display: flex;
  border-bottom: 1px solid #e4e7ed;
}

.compare-header {
  flex: 1;
  padding: 12px;
  text-align: center;
}

.compare-header.base {
  background: rgba(245, 108, 108, 0.05);
  border-right: 1px solid #e4e7ed;
}

.compare-header.compare {
  background: rgba(103, 194, 58, 0.05);
}

.header-version {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.header-desc {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}

.loading-properties {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: #909399;
  gap: 8px;
}

.properties-table {
  flex: 1;
  overflow-y: auto;
}

.property-row {
  display: flex;
  border-bottom: 1px solid #f0f0f0;
  font-size: 13px;
}

.property-row.modified {
  background: rgba(255, 200, 0, 0.1);
}

.property-row.only-base .property-value.compare {
  background: rgba(245, 108, 108, 0.1);
}

.property-row.only-compare .property-value.base {
  background: rgba(103, 194, 58, 0.1);
}

.property-name {
  width: 30%;
  padding: 8px 12px;
  color: #606266;
  background: #fafafa;
  border-right: 1px solid #f0f0f0;
}

.property-value {
  width: 35%;
  padding: 8px 12px;
  color: #303133;
  word-break: break-all;
}

.property-value.base {
  border-right: 1px solid #f0f0f0;
}

.missing-value {
  color: #c0c4cc;
}

:deep(.el-drawer__header) {
  margin-bottom: 0;
  border-bottom: 1px solid #e4e7ed;
}

.annotation-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #ff9800;
  font-size: 14px;
}

.annotations-section {
  padding: 0 12px 8px;
  border-top: 1px solid #2a2a4a;
}

.annotation-collapse-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #8899aa;
  font-size: 12px;
  font-weight: 500;
}

.annotation-list {
  max-height: 300px;
  overflow-y: auto;
}

.annotation-item {
  padding: 10px;
  border: 1px solid #2a2a4a;
  border-radius: 4px;
  margin-bottom: 8px;
  background: rgba(0, 0, 0, 0.15);
}

.annotation-item-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
}

.annotation-element-name {
  flex: 1;
  color: #ccddee;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.annotation-element-name:hover {
  color: #409eff;
}

.annotation-delete-btn {
  margin-left: auto;
  padding: 2px 6px;
  font-size: 11px;
}

.annotation-content {
  color: #aabbcc;
  font-size: 12px;
  line-height: 1.5;
  word-break: break-all;
  margin-bottom: 4px;
}

.annotation-meta {
  display: flex;
  gap: 10px;
  color: #8899aa;
  font-size: 11px;
}

.export-section {
  padding: 12px;
  border-top: 1px solid #2a2a4a;
}

:deep(.el-collapse) {
  border: none;
  background: transparent;
}

:deep(.el-collapse-item__header) {
  background: transparent;
  border-bottom: none;
  color: #8899aa;
  font-size: 12px;
  padding-left: 0;
  padding-right: 0;
  height: 36px;
  line-height: 36px;
}

:deep(.el-collapse-item__wrap) {
  background: transparent;
  border-bottom: none;
}

:deep(.el-collapse-item__content) {
  padding: 0;
}

.annotation-section {
  padding: 16px;
  border-top: 1px solid #e4e7ed;
  background: #fafafa;
}

.annotation-section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12px;
}

.annotation-section-title .el-icon {
  color: #ff9800;
}

.existing-annotations {
  margin-bottom: 16px;
}

.existing-annotation-item {
  padding: 12px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  margin-bottom: 8px;
}

.existing-annotation-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}

.existing-annotation-author {
  font-weight: 600;
  font-size: 13px;
  color: #303133;
}

.existing-annotation-time {
  font-size: 12px;
  color: #909399;
  flex: 1;
}

.existing-annotation-content {
  font-size: 13px;
  color: #606266;
  line-height: 1.6;
  word-break: break-all;
}

.annotation-input-area {
  background: #fff;
  padding: 12px;
  border: 1px solid #ebeef5;
  border-radius: 4px;
}
</style>
