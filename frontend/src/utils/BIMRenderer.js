import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'

export class BIMRenderer {
  constructor(container) {
    this.container = container
    this.scene = new THREE.Scene()
    this.camera = null
    this.renderer = null
    this.controls = null
    this.raycaster = new THREE.Raycaster()
    this.mouse = new THREE.Vector2()
    this.elementMeshes = new Map()
    this.instanceGroups = new Map()
    this.lodManager = null
    this.clippingPlanes = []
    this.outlinePass = null
    this.highlightMesh = null
    this.boxHelper = null
    this.animationId = null
    this.onElementClick = null
    this.onElementHover = null
    this.onBoxSelect = null
    this.isBoxSelecting = false
    this.boxSelectStart = null
    this.boxSelectEnd = null
    this.selectionBox = null
    this.frustum = new THREE.Frustum()
    this.projScreenMatrix = new THREE.Matrix4()
    this.occlusionCuller = null
    this.lastFrameTime = 0
    this.fps = 0
    this.frameCount = 0
    this.fpsUpdateTime = 0

    this._init()
  }

  _init() {
    const width = this.container.clientWidth
    const height = this.container.clientHeight

    this.camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 10000)
    this.camera.position.set(30, 30, 30)
    this.camera.lookAt(0, 0, 0)

    this.renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: true,
      logarithmicDepthBuffer: true
    })
    this.renderer.setSize(width, height)
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
    this.renderer.shadowMap.enabled = false
    this.renderer.localClippingEnabled = true
    this.renderer.outputColorSpace = THREE.SRGBColorSpace
    this.renderer.toneMapping = THREE.ACESFilmicToneMapping
    this.renderer.toneMappingExposure = 1.0
    this.container.appendChild(this.renderer.domElement)

    this.controls = new OrbitControls(this.camera, this.renderer.domElement)
    this.controls.enableDamping = true
    this.controls.dampingFactor = 0.1
    this.controls.minDistance = 1
    this.controls.maxDistance = 2000

    const ambientLight = new THREE.AmbientLight(0xffffff, 0.6)
    this.scene.add(ambientLight)

    const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8)
    directionalLight.position.set(50, 100, 50)
    this.scene.add(directionalLight)

    const directionalLight2 = new THREE.DirectionalLight(0xffffff, 0.3)
    directionalLight2.position.set(-50, 50, -50)
    this.scene.add(directionalLight2)

    const hemiLight = new THREE.HemisphereLight(0xddeeff, 0x0f0e0d, 0.4)
    this.scene.add(hemiLight)

    this.scene.background = new THREE.Color(0x1a1a2e)

    const grid = new THREE.GridHelper(200, 40, 0x444466, 0x333355)
    grid.material.opacity = 0.3
    grid.material.transparent = true
    this.scene.add(grid)

    this._setupEvents()
    this._animate()
  }

  _setupEvents() {
    const canvas = this.renderer.domElement

    canvas.addEventListener('click', (e) => this._onClick(e))
    canvas.addEventListener('mousemove', (e) => this._onMouseMove(e))
    canvas.addEventListener('contextmenu', (e) => this._onContextMenu(e))
    canvas.addEventListener('mousedown', (e) => this._onMouseDown(e))
    canvas.addEventListener('mouseup', (e) => this._onMouseUp(e))

    window.addEventListener('resize', () => this._onResize())
    window.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        this.clearHighlight()
      }
    })
  }

  _onClick(event) {
    if (this.isBoxSelecting) return

    const rect = this.renderer.domElement.getBoundingClientRect()
    this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1
    this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1

    this.raycaster.setFromCamera(this.mouse, this.camera)

    const meshes = Array.from(this.elementMeshes.values())
    const flatMeshes = []
    for (const mesh of meshes) {
      if (mesh.isInstancedMesh) {
        flatMeshes.push(mesh)
      } else {
        flatMeshes.push(mesh)
      }
    }

    const intersects = this.raycaster.intersectObjects(flatMeshes, false)

    if (intersects.length > 0) {
      const hit = intersects[0]
      const elementId = hit.object.userData.elementId

      if (hit.object.isInstancedMesh && hit.instanceId !== undefined) {
        const instancedData = hit.object.userData.instances
        if (instancedData && instancedData[hit.instanceId]) {
          const clickedId = instancedData[hit.instanceId].elementId
          if (this.onElementClick) {
            this.onElementClick(clickedId, event.shiftKey)
          }
          return
        }
      }

      if (elementId && this.onElementClick) {
        this.onElementClick(elementId, event.shiftKey)
      }
    }
  }

  _onMouseMove(event) {
    const rect = this.renderer.domElement.getBoundingClientRect()
    this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1
    this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1
  }

  _onContextMenu(event) {
    event.preventDefault()
    const rect = this.renderer.domElement.getBoundingClientRect()
    this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1
    this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1

    this.raycaster.setFromCamera(this.mouse, this.camera)
    const meshes = Array.from(this.elementMeshes.values())
    const intersects = this.raycaster.intersectObjects(meshes, false)

    if (intersects.length > 0) {
      const hit = intersects[0]
      let elementId = hit.object.userData.elementId
      if (hit.object.isInstancedMesh && hit.instanceId !== undefined) {
        const instancedData = hit.object.userData.instances
        if (instancedData && instancedData[hit.instanceId]) {
          elementId = instancedData[hit.instanceId].elementId
        }
      }
      if (elementId && this.onElementClick) {
        this.onElementClick(elementId, false, { x: event.clientX, y: event.clientY })
      }
    }
  }

  _onMouseDown(event) {
    if (event.shiftKey && event.button === 0) {
      this.isBoxSelecting = true
      this.boxSelectStart = { x: event.clientX, y: event.clientY }
    }
  }

  _onMouseUp(event) {
    if (this.isBoxSelecting && this.boxSelectStart) {
      this.boxSelectEnd = { x: event.clientX, y: event.clientY }
      this._performBoxSelect()
      this.isBoxSelecting = false
      this.boxSelectStart = null
      this.boxSelectEnd = null
      if (this.selectionBox) {
        this.scene.remove(this.selectionBox)
        this.selectionBox = null
      }
    }
  }

  _performBoxSelect() {
    if (!this.boxSelectStart || !this.boxSelectEnd) return

    const rect = this.renderer.domElement.getBoundingClientRect()
    const x1 = Math.min(this.boxSelectStart.x, this.boxSelectEnd.x)
    const y1 = Math.min(this.boxSelectStart.y, this.boxSelectEnd.y)
    const x2 = Math.max(this.boxSelectStart.x, this.boxSelectEnd.x)
    const y2 = Math.max(this.boxSelectStart.y, this.boxSelectEnd.y)

    if (Math.abs(x2 - x1) < 5 || Math.abs(y2 - y1) < 5) return

    const ndcX1 = ((x1 - rect.left) / rect.width) * 2 - 1
    const ndcY1 = -((y1 - rect.top) / rect.height) * 2 + 1
    const ndcX2 = ((x2 - rect.left) / rect.width) * 2 - 1
    const ndcY2 = -((y2 - rect.top) / rect.height) * 2 + 1

    const selectedIds = []

    for (const [elementId, mesh] of this.elementMeshes) {
      if (!mesh.visible) continue

      if (mesh.isInstancedMesh) {
        const instancedData = mesh.userData.instances
        if (instancedData) {
          for (let i = 0; i < instancedData.length; i++) {
            const pos = new THREE.Vector3()
            const matrix = new THREE.Matrix4()
            mesh.getMatrixAt(i, matrix)
            pos.setFromMatrixPosition(matrix)
            pos.project(this.camera)

            if (pos.x >= ndcX1 && pos.x <= ndcX2 && pos.y >= ndcY1 && pos.y <= ndcY2 && pos.z <= 1) {
              selectedIds.push(instancedData[i].elementId)
            }
          }
        }
      } else {
        const pos = new THREE.Vector3()
        pos.setFromMatrixPosition(mesh.matrixWorld)
        pos.project(this.camera)

        if (pos.x >= ndcX1 && pos.x <= ndcX2 && pos.y >= ndcY1 && pos.y <= ndcY2 && pos.z <= 1) {
          selectedIds.push(elementId)
        }
      }
    }

    if (this.onBoxSelect && selectedIds.length > 0) {
      this.onBoxSelect(selectedIds)
    }
  }

  _onResize() {
    const width = this.container.clientWidth
    const height = this.container.clientHeight
    this.camera.aspect = width / height
    this.camera.updateProjectionMatrix()
    this.renderer.setSize(width, height)
  }

  _animate() {
    this.animationId = requestAnimationFrame(() => this._animate())

    const now = performance.now()
    this.frameCount++
    if (now - this.fpsUpdateTime >= 1000) {
      this.fps = this.frameCount
      this.frameCount = 0
      this.fpsUpdateTime = now
    }

    this.controls.update()

    this._performFrustumCulling()
    this._updateLOD()

    this.renderer.render(this.scene, this.camera)
  }

  _performFrustumCulling() {
    this.projScreenMatrix.multiplyMatrices(
      this.camera.projectionMatrix,
      this.camera.matrixWorldInverse
    )
    this.frustum.setFromProjectionMatrix(this.projScreenMatrix)

    for (const [id, mesh] of this.elementMeshes) {
      if (!mesh.userData.forceVisible) {
        mesh.visible = this.frustum.intersectsObject(mesh)
      }
    }
  }

  _updateLOD() {
    if (!this.lodManager) return
    this.lodManager.update(this.camera)
  }

  addElementMesh(elementId, geometry, material, position) {
    const mesh = new THREE.Mesh(geometry, material)
    mesh.userData.elementId = elementId
    if (position) {
      mesh.position.set(position.x, position.y, position.z)
    }
    mesh.castShadow = false
    mesh.receiveShadow = false
    this.elementMeshes.set(elementId, mesh)
    this.scene.add(mesh)
    return mesh
  }

  addInstancedMesh(geometryHash, geometry, material, instances) {
    const instancedMesh = new THREE.InstancedMesh(geometry, material, instances.length)
    instancedMesh.userData.geometryHash = geometryHash
    instancedMesh.userData.instances = instances

    const dummy = new THREE.Object3D()
    for (let i = 0; i < instances.length; i++) {
      const inst = instances[i]
      dummy.position.set(inst.position.x, inst.position.y, inst.position.z)
      if (inst.rotation) {
        dummy.rotation.set(inst.rotation.x, inst.rotation.y, inst.rotation.z)
      }
      if (inst.scale) {
        dummy.scale.set(inst.scale.x, inst.scale.y, inst.scale.z)
      }
      dummy.updateMatrix()
      instancedMesh.setMatrixAt(i, dummy.matrix)
    }

    instancedMesh.instanceMatrix.needsUpdate = true
    this.instanceGroups.set(geometryHash, instancedMesh)
    this.scene.add(instancedMesh)

    for (const inst of instances) {
      this.elementMeshes.set(inst.elementId, instancedMesh)
    }

    return instancedMesh
  }

  highlightElement(elementId, color = 0x00aaff) {
    const mesh = this.elementMeshes.get(elementId)
    if (!mesh) return

    if (this.highlightMesh) {
      this.scene.remove(this.highlightMesh)
      this.highlightMesh = null
    }

    if (this.boxHelper) {
      this.scene.remove(this.boxHelper)
      this.boxHelper = null
    }

    if (mesh.isInstancedMesh) {
      const instances = mesh.userData.instances
      if (!instances) return
      const idx = instances.findIndex(inst => inst.elementId === elementId)
      if (idx < 0) return

      const matrix = new THREE.Matrix4()
      mesh.getMatrixAt(idx, matrix)
      const pos = new THREE.Vector3()
      const quat = new THREE.Quaternion()
      const scale = new THREE.Vector3()
      matrix.decompose(pos, quat, scale)

      const boxGeo = new THREE.BoxGeometry(
        mesh.geometry.parameters?.width || 1,
        mesh.geometry.parameters?.height || 1,
        mesh.geometry.parameters?.depth || 1
      )
      const edgesGeo = new THREE.EdgesGeometry(boxGeo)
      const edgesMat = new THREE.LineBasicMaterial({ color })
      const edgeMesh = new THREE.LineSegments(edgesGeo, edgesMat)
      edgeMesh.position.copy(pos)
      edgeMesh.quaternion.copy(quat)
      edgeMesh.scale.copy(scale).multiplyScalar(1.05)
      this.highlightMesh = edgeMesh
      this.scene.add(this.highlightMesh)
    } else {
      const edges = new THREE.EdgesGeometry(mesh.geometry)
      const lineMat = new THREE.LineBasicMaterial({ color })
      const wireframe = new THREE.LineSegments(edges, lineMat)
      wireframe.position.copy(mesh.position)
      wireframe.rotation.copy(mesh.rotation)
      wireframe.scale.copy(mesh.scale).multiplyScalar(1.02)
      this.highlightMesh = wireframe
      this.scene.add(this.highlightMesh)
    }

    this.boxHelper = new THREE.BoxHelper(
      mesh.isInstancedMesh ? this._getInstanceAsMesh(mesh, elementId) : mesh,
      color
    )
    if (this.boxHelper) {
      this.scene.add(this.boxHelper)
    }
  }

  _getInstanceAsMesh(instancedMesh, elementId) {
    const instances = instancedMesh.userData.instances
    const idx = instances?.findIndex(inst => inst.elementId === elementId)
    if (idx === undefined || idx < 0) return instancedMesh

    const matrix = new THREE.Matrix4()
    instancedMesh.getMatrixAt(idx, matrix)
    const tempMesh = new THREE.Mesh(instancedMesh.geometry, instancedMesh.material)
    tempMesh.applyMatrix4(matrix)
    return tempMesh
  }

  clearHighlight() {
    if (this.highlightMesh) {
      this.scene.remove(this.highlightMesh)
      this.highlightMesh = null
    }
    if (this.boxHelper) {
      this.scene.remove(this.boxHelper)
      this.boxHelper = null
    }
  }

  setElementVisibility(elementId, visible) {
    const mesh = this.elementMeshes.get(elementId)
    if (mesh) {
      mesh.userData.forceVisible = visible
      mesh.visible = visible
    }
  }

  setElementOpacity(elementId, opacity) {
    const mesh = this.elementMeshes.get(elementId)
    if (mesh && !mesh.isInstancedMesh) {
      mesh.material.transparent = true
      mesh.material.opacity = opacity
      mesh.material.needsUpdate = true
    }
  }

  isolateElements(elementIds) {
    for (const [id, mesh] of this.elementMeshes) {
      mesh.userData.forceVisible = elementIds.includes(id) || elementIds.has(id)
      mesh.visible = mesh.userData.forceVisible
    }
  }

  showAllElements() {
    for (const [id, mesh] of this.elementMeshes) {
      mesh.userData.forceVisible = false
      mesh.visible = true
    }
  }

  setCategoryOpacity(category, opacity, elementStore) {
    const elements = elementStore.elements.filter(e => e.category !== category)
    for (const e of elements) {
      this.setElementOpacity(e.id, opacity)
    }
  }

  addClippingPlane(normal, constant) {
    const plane = new THREE.Plane(
      new THREE.Vector3(normal.x, normal.y, normal.z),
      constant
    )
    this.clippingPlanes.push(plane)
    this._updateClippingPlanes()
    return plane
  }

  removeClippingPlane(plane) {
    const idx = this.clippingPlanes.indexOf(plane)
    if (idx >= 0) {
      this.clippingPlanes.splice(idx, 1)
      this._updateClippingPlanes()
    }
  }

  updateClippingPlane(plane, constant) {
    plane.constant = constant
  }

  clearClippingPlanes() {
    this.clippingPlanes = []
    this._updateClippingPlanes()
  }

  _updateClippingPlanes() {
    for (const [id, mesh] of this.elementMeshes) {
      if (mesh.material) {
        if (Array.isArray(mesh.material)) {
          mesh.material.forEach(m => {
            m.clippingPlanes = this.clippingPlanes
            m.needsUpdate = true
          })
        } else {
          mesh.material.clippingPlanes = this.clippingPlanes
          mesh.material.needsUpdate = true
        }
      }
    }
  }

  flyTo(position, target, duration = 1000) {
    const startPos = this.camera.position.clone()
    const startTarget = this.controls.target.clone()

    const endPos = new THREE.Vector3(position.x, position.y, position.z)
    const endTarget = new THREE.Vector3(target.x, target.y, target.z)

    const startTime = performance.now()

    const animateFly = () => {
      const elapsed = performance.now() - startTime
      const t = Math.min(elapsed / duration, 1)
      const easedT = t < 0.5 ? 2 * t * t : -1 + (4 - 2 * t) * t

      this.camera.position.lerpVectors(startPos, endPos, easedT)
      this.controls.target.lerpVectors(startTarget, endTarget, easedT)
      this.controls.update()

      if (t < 1) {
        requestAnimationFrame(animateFly)
      }
    }

    animateFly()
  }

  fitToView() {
    const box = new THREE.Box3()
    for (const [id, mesh] of this.elementMeshes) {
      if (mesh.visible) {
        if (mesh.isInstancedMesh) {
          mesh.computeBoundingBox()
          if (mesh.boundingBox) box.union(mesh.boundingBox)
        } else {
          box.expandByObject(mesh)
        }
      }
    }

    if (box.isEmpty()) return

    const center = box.getCenter(new THREE.Vector3())
    const size = box.getSize(new THREE.Vector3())
    const maxDim = Math.max(size.x, size.y, size.z)
    const distance = maxDim * 1.5

    const direction = new THREE.Vector3(1, 0.8, 1).normalize()
    const newPos = center.clone().add(direction.multiplyScalar(distance))

    this.flyTo(
      { x: newPos.x, y: newPos.y, z: newPos.z },
      { x: center.x, y: center.y, z: center.z }
    )
  }

  addMeasurementLine(points, color = 0xffff00) {
    const geometry = new THREE.BufferGeometry().setFromPoints(
      points.map(p => new THREE.Vector3(p.x, p.y, p.z))
    )
    const material = new THREE.LineBasicMaterial({ color, linewidth: 2 })
    const line = new THREE.Line(geometry, material)
    this.scene.add(line)
    return line
  }

  addMeasurementLabel(text, position, color = '#ffffff') {
    const canvas = document.createElement('canvas')
    const ctx = canvas.getContext('2d')
    canvas.width = 256
    canvas.height = 64
    ctx.fillStyle = 'rgba(0,0,0,0.7)'
    ctx.fillRect(0, 0, 256, 64)
    ctx.fillStyle = color
    ctx.font = '24px Arial'
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(text, 128, 32)

    const texture = new THREE.CanvasTexture(canvas)
    const spriteMat = new THREE.SpriteMaterial({ map: texture })
    const sprite = new THREE.Sprite(spriteMat)
    sprite.position.set(position.x, position.y + 1, position.z)
    sprite.scale.set(4, 1, 1)
    this.scene.add(sprite)
    return sprite
  }

  addCollisionMarker(position, color = 0xff0000) {
    const geometry = new THREE.SphereGeometry(0.3, 16, 16)
    const material = new THREE.MeshBasicMaterial({ color, transparent: true, opacity: 0.8 })
    const sphere = new THREE.Mesh(geometry, material)
    sphere.position.set(position.x, position.y, position.z)
    this.scene.add(sphere)
    return sphere
  }

  takeScreenshot() {
    this.renderer.render(this.scene, this.camera)
    return this.renderer.domElement.toDataURL('image/png')
  }

  getFPS() {
    return this.fps
  }

  dispose() {
    if (this.animationId) {
      cancelAnimationFrame(this.animationId)
    }

    for (const [id, mesh] of this.elementMeshes) {
      if (mesh.geometry) mesh.geometry.dispose()
      if (mesh.material) {
        if (Array.isArray(mesh.material)) {
          mesh.material.forEach(m => m.dispose())
        } else {
          mesh.material.dispose()
        }
      }
    }

    this.renderer.dispose()
    this.controls.dispose()
    this.elementMeshes.clear()
    this.instanceGroups.clear()
  }
}
