import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useClipStore = defineStore('clip', () => {
  const planes = ref([])
  const maxPlanes = 3

  const presetNormals = {
    XY: { x: 0, y: 0, z: 1 },
    XZ: { x: 0, y: 1, z: 0 },
    YZ: { x: 1, y: 0, z: 0 }
  }

  function addPlane(normal = 'XZ', position = 0) {
    if (planes.value.length >= maxPlanes) return false
    const normalVec = presetNormals[normal] || { x: 0, y: 1, z: 0 }
    planes.value.push({
      id: `clip_${Date.now()}`,
      normal: { ...normalVec },
      position,
      constant: -position,
      enabled: true,
      preset: normal
    })
    return true
  }

  function removePlane(planeId) {
    planes.value = planes.value.filter(p => p.id !== planeId)
  }

  function updatePlanePosition(planeId, position) {
    const plane = planes.value.find(p => p.id === planeId)
    if (plane) {
      plane.position = position
      plane.constant = -position
    }
  }

  function togglePlane(planeId) {
    const plane = planes.value.find(p => p.id === planeId)
    if (plane) {
      plane.enabled = !plane.enabled
    }
  }

  function setFloorClip(floorHeight) {
    clearAll()
    addPlane('XZ', floorHeight)
  }

  function clearAll() {
    planes.value = []
  }

  return {
    planes, maxPlanes, presetNormals,
    addPlane, removePlane, updatePlanePosition, togglePlane,
    setFloorClip, clearAll
  }
})
