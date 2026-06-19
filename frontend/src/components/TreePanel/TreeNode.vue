<template>
  <div class="tree-node">
    <div
      class="node-row"
      :style="{ paddingLeft: depth * 16 + 8 + 'px' }"
      :class="{ selected: isSelected, highlighted: isHighlighted }"
      @click="onNodeClick"
      @contextmenu.prevent="onContextMenu"
    >
      <el-icon
        v-if="node.children && node.children.length > 0"
        class="expand-icon"
        :class="{ expanded: expanded }"
        @click.stop="expanded = !expanded"
      >
        <ArrowRight />
      </el-icon>
      <span v-else class="expand-placeholder" />

      <el-checkbox
        v-model="visible"
        size="small"
        @change="onVisibilityChange"
        @click.stop
      />

      <span class="node-icon" :style="{ color: typeColor }">●</span>
      <span class="node-name" :title="node.name">{{ node.name || node.ifcGuid }}</span>
      <span class="node-type">{{ formatType(node.type) }}</span>
    </div>

    <div v-if="expanded && node.children" class="node-children">
      <TreeNode
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :depth="depth + 1"
        @visibility-change="onChildVisibilityChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useModelStore } from '../../stores/model'
import { useViewerStore } from '../../stores/viewer'

const props = defineProps({
  node: { type: Object, required: true },
  depth: { type: Number, default: 0 }
})

const emit = defineEmits(['visibility-change'])

const modelStore = useModelStore()
const viewerStore = useViewerStore()

const expanded = ref(props.depth < 2)
const visible = ref(true)

const isSelected = computed(() => {
  return modelStore.selectedElementIds.has(props.node.id)
})

const isHighlighted = computed(() => {
  return modelStore.highlightedElementId === props.node.id
})

const typeColor = computed(() => {
  const colors = {
    IfcProject: '#ff6b6b',
    IfcSite: '#ffd93d',
    IfcBuilding: '#6bcb77',
    IfcBuildingStorey: '#4d96ff',
    IfcSpace: '#9b59b6'
  }
  return colors[props.node.type] || '#8899aa'
})

function formatType(type) {
  if (!type) return ''
  return type.replace('Ifc', '')
}

function collectDescendantElementIds(node, ids = []) {
  if (node.type === 'IfcBuildingStorey') {
    const floorElements = modelStore.elements.filter(e => e.floorName === node.name)
    for (const e of floorElements) {
      ids.push(e.id)
    }
  } else if (!node.children || node.children.length === 0) {
    const direct = modelStore.elements.find(e => e.id === node.id)
    if (direct) ids.push(direct.id)
  }
  if (node.children) {
    for (const child of node.children) {
      collectDescendantElementIds(child, ids)
    }
  }
  return ids
}

function onNodeClick() {
  modelStore.selectElement(props.node.id)
}

function onContextMenu(e) {
  viewerStore.showContextMenu(e.clientX, e.clientY, props.node.id)
}

function onVisibilityChange(val) {
  const affectedIds = collectDescendantElementIds(props.node)
  emit('visibility-change', { ids: affectedIds, visible: val })
}

function onChildVisibilityChange(payload) {
  emit('visibility-change', payload)
}
</script>

<style scoped>
.tree-node {
  user-select: none;
}

.node-row {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  cursor: pointer;
  border-radius: 3px;
  transition: background 0.15s;
  font-size: 12px;
}

.node-row:hover {
  background: rgba(64, 128, 255, 0.15);
}

.node-row.selected {
  background: rgba(64, 128, 255, 0.25);
}

.node-row.highlighted {
  background: rgba(0, 170, 255, 0.3);
}

.expand-icon {
  font-size: 12px;
  color: #8899aa;
  transition: transform 0.2s;
  cursor: pointer;
}

.expand-icon.expanded {
  transform: rotate(90deg);
}

.expand-placeholder {
  width: 12px;
  display: inline-block;
}

.node-icon {
  font-size: 8px;
}

.node-name {
  color: #ccddee;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-type {
  color: #556677;
  font-size: 10px;
}

.node-children {
  /* children rendered by recursion */
}
</style>
