<template>
  <div class="tree-panel">
    <div class="panel-header">
      <span>模型结构</span>
      <el-input
        v-model="searchText"
        placeholder="搜索构件..."
        size="small"
        clearable
        prefix-icon="Search"
        class="search-input"
      />
    </div>
    <div class="tree-content">
      <div v-for="node in filteredTree" :key="node.id">
        <TreeNode :node="node" :depth="0" @visibility-change="onVisibilityChange" />
      </div>
      <div v-if="!filteredTree.length" class="empty-text">暂无模型结构数据</div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useModelStore } from '../../stores/model'
import TreeNode from './TreeNode.vue'

const emit = defineEmits(['visibility-change'])

const modelStore = useModelStore()
const searchText = ref('')

const filteredTree = computed(() => {
  if (!searchText.value) return modelStore.spatialTree
  return filterTree(modelStore.spatialTree, searchText.value.toLowerCase())
})

function filterTree(nodes, query) {
  const result = []
  for (const node of nodes) {
    const nameMatch = node.name?.toLowerCase().includes(query)
    const typeMatch = node.type?.toLowerCase().includes(query)
    const filteredChildren = node.children ? filterTree(node.children, query) : []

    if (nameMatch || typeMatch || filteredChildren.length > 0) {
      result.push({
        ...node,
        children: filteredChildren.length > 0 ? filteredChildren : node.children
      })
    }
  }
  return result
}

function onVisibilityChange(payload) {
  emit('visibility-change', payload)
}
</script>

<style scoped>
.tree-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.panel-header {
  padding: 8px;
  border-bottom: 1px solid #2a2a4a;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.panel-header span {
  color: #ccddee;
  font-size: 13px;
  font-weight: 500;
}

.search-input {
  width: 100%;
}

.tree-content {
  flex: 1;
  overflow-y: auto;
  padding: 4px;
}

.empty-text {
  color: #556677;
  font-size: 12px;
  text-align: center;
  padding: 20px;
}
</style>
