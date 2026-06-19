<template>
  <div class="property-panel">
    <div class="panel-header">
      <span>构件属性</span>
    </div>
    <div v-if="!selectedElements.length" class="empty-text">
      请选择一个构件查看属性
    </div>
    <div v-else class="property-content">
      <div v-for="element in selectedElements" :key="element.id" class="element-props">
        <div class="prop-section">
          <div class="section-title">基本信息</div>
          <div class="prop-row">
            <span class="prop-label">名称</span>
            <span class="prop-value">{{ element.name || '-' }}</span>
          </div>
          <div class="prop-row">
            <span class="prop-label">类型</span>
            <span class="prop-value">{{ element.type }}</span>
          </div>
          <div class="prop-row">
            <span class="prop-label">分类</span>
            <span class="prop-value">{{ getCategoryName(element.category) }}</span>
          </div>
          <div class="prop-row">
            <span class="prop-label">楼层</span>
            <span class="prop-value">{{ element.floorName || '-' }}</span>
          </div>
          <div class="prop-row">
            <span class="prop-label">GUID</span>
            <span class="prop-value guid">{{ element.ifcGuid }}</span>
          </div>
        </div>

        <div class="prop-section">
          <div class="section-title">空间范围</div>
          <div class="prop-row">
            <span class="prop-label">最小点</span>
            <span class="prop-value">{{ formatVec3(element.aabbMin) }}</span>
          </div>
          <div class="prop-row">
            <span class="prop-label">最大点</span>
            <span class="prop-value">{{ formatVec3(element.aabbMax) }}</span>
          </div>
          <div class="prop-row">
            <span class="prop-label">尺寸</span>
            <span class="prop-value">{{ formatSize(element) }}</span>
          </div>
        </div>

        <div v-if="element.properties && Object.keys(element.properties).length > 0" class="prop-section">
          <div class="section-title">扩展属性</div>
          <div v-for="(value, key) in element.properties" :key="key" class="prop-row">
            <span class="prop-label">{{ key.replace('prop_', 'P') }}</span>
            <span class="prop-value">{{ value || '-' }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useModelStore } from '../../stores/model'
import { useViewerStore } from '../../stores/viewer'

const modelStore = useModelStore()
const viewerStore = useViewerStore()

const selectedElements = computed(() => modelStore.selectedElements)

function getCategoryName(category) {
  const cat = viewerStore.categories.find(c => c.id === category)
  return cat ? cat.name : category
}

function formatVec3(v) {
  if (!v) return '-'
  return `(${v[0].toFixed(1)}, ${v[1].toFixed(1)}, ${v[2].toFixed(1)})`
}

function formatSize(element) {
  if (!element.aabbMin || !element.aabbMax) return '-'
  const dx = (element.aabbMax[0] - element.aabbMin[0]).toFixed(0)
  const dy = (element.aabbMax[1] - element.aabbMin[1]).toFixed(0)
  const dz = (element.aabbMax[2] - element.aabbMin[2]).toFixed(0)
  return `${dx} × ${dy} × ${dz} mm`
}
</script>

<style scoped>
.property-panel {
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

.empty-text {
  color: #556677;
  font-size: 12px;
  text-align: center;
  padding: 40px 12px;
}

.property-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.element-props {
  margin-bottom: 12px;
}

.prop-section {
  background: rgba(42, 42, 74, 0.5);
  border-radius: 6px;
  padding: 8px;
  margin-bottom: 8px;
}

.section-title {
  color: #6699cc;
  font-size: 12px;
  font-weight: 500;
  margin-bottom: 6px;
  padding-bottom: 4px;
  border-bottom: 1px solid #2a2a4a;
}

.prop-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 3px 0;
  font-size: 12px;
}

.prop-label {
  color: #8899aa;
  flex-shrink: 0;
  margin-right: 8px;
}

.prop-value {
  color: #ccddee;
  text-align: right;
  word-break: break-all;
}

.prop-value.guid {
  font-family: monospace;
  font-size: 10px;
  color: #778899;
}
</style>
