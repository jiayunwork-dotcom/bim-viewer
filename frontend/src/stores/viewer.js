import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useViewerStore = defineStore('viewer', () => {
  const activePanel = ref('tree')
  const filterCategory = ref(null)
  const isolatedElementIds = ref(new Set())
  const hiddenElementIds = ref(new Set())
  const transparentElementIds = ref(new Set())
  const contextMenuVisible = ref(false)
  const contextMenuPosition = ref({ x: 0, y: 0 })
  const contextMenuTarget = ref(null)
  const cameraPosition = ref(null)
  const cameraTarget = ref(null)
  const currentLOD = ref(0)

  const categories = [
    { id: 'Wall', name: '墙体', color: '#b0b0b0' },
    { id: 'Slab', name: '楼板', color: '#c8c8c8' },
    { id: 'Column', name: '柱', color: '#a0a0a0' },
    { id: 'Beam', name: '梁', color: '#909090' },
    { id: 'Pipe', name: '管道', color: '#4080ff' },
    { id: 'Duct', name: '风管', color: '#80c0ff' },
    { id: 'Equipment', name: '设备', color: '#808080' },
    { id: 'Door', name: '门', color: '#d0a060' },
    { id: 'Window', name: '窗', color: '#80d0ff' }
  ]

  function setActivePanel(panel) {
    activePanel.value = activePanel.value === panel ? null : panel
  }

  function setFilterCategory(category) {
    filterCategory.value = filterCategory.value === category ? null : category
  }

  function isolateElements(elementIds) {
    isolatedElementIds.value = new Set(elementIds)
    hiddenElementIds.value = new Set()
    transparentElementIds.value = new Set()
  }

  function hideElements(elementIds) {
    const newSet = new Set(hiddenElementIds.value)
    for (const id of elementIds) {
      newSet.add(id)
    }
    hiddenElementIds.value = newSet
  }

  function makeTransparent(elementIds) {
    const newSet = new Set(transparentElementIds.value)
    for (const id of elementIds) {
      newSet.add(id)
    }
    transparentElementIds.value = newSet
  }

  function showAllElements() {
    isolatedElementIds.value = new Set()
    hiddenElementIds.value = new Set()
    transparentElementIds.value = new Set()
  }

  function showContextMenu(x, y, target) {
    contextMenuVisible.value = true
    contextMenuPosition.value = { x, y }
    contextMenuTarget.value = target
  }

  function hideContextMenu() {
    contextMenuVisible.value = false
    contextMenuTarget.value = null
  }

  function setCamera(position, target) {
    cameraPosition.value = position
    cameraTarget.value = target
  }

  function setLOD(lod) {
    currentLOD.value = lod
  }

  return {
    activePanel, filterCategory, isolatedElementIds, hiddenElementIds,
    transparentElementIds, contextMenuVisible, contextMenuPosition,
    contextMenuTarget, cameraPosition, cameraTarget, currentLOD,
    categories,
    setActivePanel, setFilterCategory, isolateElements, hideElements,
    makeTransparent, showAllElements, showContextMenu, hideContextMenu,
    setCamera, setLOD
  }
})
