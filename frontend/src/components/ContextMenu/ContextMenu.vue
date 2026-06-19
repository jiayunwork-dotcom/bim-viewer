<template>
  <div
    v-if="viewerStore.contextMenuVisible"
    class="context-menu"
    :style="{ left: viewerStore.contextMenuPosition.x + 'px', top: viewerStore.contextMenuPosition.y + 'px' }"
  >
    <div class="menu-item" @click="isolateElement">
      <el-icon><View /></el-icon>
      <span>隔离显示</span>
    </div>
    <div class="menu-item" @click="hideElement">
      <el-icon><Hide /></el-icon>
      <span>隐藏</span>
    </div>
    <div class="menu-item" @click="makeTransparent">
      <el-icon><SemiSelect /></el-icon>
      <span>透明化</span>
    </div>
    <div class="menu-item" @click="viewProperties">
      <el-icon><InfoFilled /></el-icon>
      <span>查看属性</span>
    </div>
    <div class="menu-divider" />
    <div class="menu-item" @click="addToCollisionGroup('A')">
      <el-icon><Plus /></el-icon>
      <span>添加到碰撞A组</span>
    </div>
    <div class="menu-item" @click="addToCollisionGroup('B')">
      <el-icon><Plus /></el-icon>
      <span>添加到碰撞B组</span>
    </div>
  </div>
</template>

<script setup>
import { useModelStore } from '../../stores/model'
import { useViewerStore } from '../../stores/viewer'
import { useCollisionStore } from '../../stores/collision'
import { ElMessage } from 'element-plus'

const props = defineProps({
  renderer: Object
})

const modelStore = useModelStore()
const viewerStore = useViewerStore()
const collisionStore = useCollisionStore()

function getElementId() {
  return viewerStore.contextMenuTarget
}

function isolateElement() {
  const id = getElementId()
  if (id && props.renderer) {
    props.renderer.isolateElements([id])
    viewerStore.isolateElements([id])
  }
  viewerStore.hideContextMenu()
}

function hideElement() {
  const id = getElementId()
  if (id && props.renderer) {
    props.renderer.setElementVisibility(id, false)
    viewerStore.hideElements([id])
  }
  viewerStore.hideContextMenu()
}

function makeTransparent() {
  const id = getElementId()
  if (id && props.renderer) {
    props.renderer.setElementOpacity(id, 0.2)
    viewerStore.makeTransparent([id])
  }
  viewerStore.hideContextMenu()
}

function viewProperties() {
  const id = getElementId()
  if (id) {
    modelStore.selectElement(id)
  }
  viewerStore.setActivePanel('property')
  viewerStore.hideContextMenu()
}

function addToCollisionGroup(group) {
  const id = getElementId()
  if (id) {
    if (group === 'A') {
      collisionStore.addToGroupA(id)
      ElMessage.success('已添加到A组')
    } else {
      collisionStore.addToGroupB(id)
      ElMessage.success('已添加到B组')
    }
  }
  viewerStore.hideContextMenu()
}

document.addEventListener('click', () => {
  if (viewerStore.contextMenuVisible) {
    viewerStore.hideContextMenu()
  }
})
</script>

<style scoped>
.context-menu {
  position: fixed;
  background: #1e2a4a;
  border: 1px solid #2a2a5a;
  border-radius: 8px;
  padding: 4px 0;
  min-width: 180px;
  z-index: 1000;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  cursor: pointer;
  color: #ccddee;
  font-size: 13px;
  transition: background 0.15s;
}

.menu-item:hover {
  background: rgba(64, 128, 255, 0.2);
}

.menu-item .el-icon {
  font-size: 14px;
  color: #8899aa;
}

.menu-divider {
  height: 1px;
  background: #2a2a5a;
  margin: 4px 0;
}
</style>
