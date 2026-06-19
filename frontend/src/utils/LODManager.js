import * as THREE from 'three'

export class LODManager {
  constructor(renderer) {
    this.renderer = renderer
    this.lodLevels = [
      { distance: 50, lod: 0 },
      { distance: 150, lod: 1 },
      { distance: Infinity, lod: 2 }
    ]
    this.meshLODs = new Map()
    this.currentLODCache = new Map()
  }

  registerElement(elementId, meshes) {
    this.meshLODs.set(elementId, meshes)
  }

  update(camera) {
    const cameraPosition = camera.position

    for (const [elementId, meshes] of this.meshLODs) {
      const lod = this._determineLOD(cameraPosition, meshes)
      const currentLOD = this.currentLODCache.get(elementId)

      if (lod !== currentLOD) {
        this._switchLOD(elementId, lod)
        this.currentLODCache.set(elementId, lod)
      }
    }
  }

  _determineLOD(cameraPosition, meshes) {
    let minDistance = Infinity

    for (const [lodLevel, mesh] of meshes) {
      if (mesh) {
        const pos = new THREE.Vector3()
        mesh.getWorldPosition(pos)
        const dist = cameraPosition.distanceTo(pos)
        if (dist < minDistance) {
          minDistance = dist
        }
      }
    }

    for (const level of this.lodLevels) {
      if (minDistance <= level.distance) {
        return level.lod
      }
    }

    return 2
  }

  _switchLOD(elementId, targetLOD) {
    const meshes = this.meshLODs.get(elementId)
    if (!meshes) return

    for (const [lodLevel, mesh] of meshes) {
      if (mesh) {
        mesh.visible = lodLevel === targetLOD
      }
    }
  }

  forceLOD(lod) {
    for (const [elementId, meshes] of this.meshLODs) {
      this._switchLOD(elementId, lod)
      this.currentLODCache.set(elementId, lod)
    }
  }

  dispose() {
    this.meshLODs.clear()
    this.currentLODCache.clear()
  }
}
