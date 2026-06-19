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
    this.elementMeshesByLOD = new Map()
    this.instanceGroups = new Map()
    this.clippingPlanes = []
    this.highlightMesh = null
    this.boxHelper = null
    this.animationId = null
    this.onElementClick = null
    this.onElementHover = null
    this.onBoxSelect = null
    this.onMeasureClick = null
    this.isBoxSelecting = false
    this.boxSelectStart = null
    this.boxSelectEnd = null
    this.selectionBox = null
    this.frustum = new THREE.Frustum()
    this.projScreenMatrix = new THREE.Matrix4()
    this.fps = 0
    this.frameCount = 0
    this.fpsUpdateTime = 0
    this.measureMode = false
    this.currentLOD = 0
    this.lodDistanceThresholds = [5000, 15000, 50000]
    this.occlusionMap = new Map()
    this.occlusionUpdateInterval = 10
    this.frameCounter = 0
    this.measurementObjects = []

    this._init()
  }

  _init() {
    const width = this.container.clientWidth
    const height = this.container.clientHeight

    this.camera = new THREE.PerspectiveCamera(60, width / height, 1, 200000)
    this.camera.position.set(50000, 50000, 50000)
    this.camera.lookAt(15000, 3500, 10000)

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
    this.controls.minDistance = 500
    this.controls.maxDistance = 200000

    const ambientLight = new THREE.AmbientLight(0xffffff, 0.7)
    this.scene.add(ambientLight)

    const directionalLight = new THREE.DirectionalLight(0xffffff, 0.9)
    directionalLight.position.set(50000, 100000, 50000)
    this.scene.add(directionalLight)

    const directionalLight2 = new THREE.DirectionalLight(0xffffff, 0.3)
    directionalLight2.position.set(-50000, 50000, -50000)
    this.scene.add(directionalLight2)

    const hemiLight = new THREE.HemisphereLight(0xddeeff, 0x0f0e0d, 0.4)
    this.scene.add(hemiLight)

    this.scene.background = new THREE.Color(0x1a1a2e)

    const gridSize = 60000
    const gridDivisions = 60
    const grid = new THREE.GridHelper(gridSize, gridDivisions, 0x444466, 0x333355)
    grid.material.opacity = 0.3
    grid.material.transparent = true
    grid.position.y = 0
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

    if (this.measureMode) {
      const intersects = this.raycaster.intersectObjects(
        Array.from(this.elementMeshes.values()).filter(m => m.visible),
        true
      )
      if (intersects.length > 0) {
        const point = intersects[0].point
        if (this.onMeasureClick) {
          this.onMeasureClick(point)
        }
      } else {
        const plane = new THREE.Plane(new THREE.Vector3(0, 1, 0), 0)
        const point = new THREE.Vector3()
        this.raycaster.ray.intersectPlane(plane, point)
        if (point && this.onMeasureClick) {
          this.onMeasureClick(point)
        }
      }
      return
    }

    const meshes = Array.from(this.elementMeshes.values()).filter(m => m.visible)
    const intersects = this.raycaster.intersectObjects(meshes, false)

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
    const meshes = Array.from(this.elementMeshes.values()).filter(m => m.visible)
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

    this.frameCounter++
    if (this.frameCounter % this.occlusionUpdateInterval === 0) {
      this._performOcclusionCulling()
    }

    this.renderer.render(this.scene, this.camera)
  }

  _performFrustumCulling() {
    this.projScreenMatrix.multiplyMatrices(
      this.camera.projectionMatrix,
      this.camera.matrixWorldInverse
    )
    this.frustum.setFromProjectionMatrix(this.projScreenMatrix)

    for (const [id, mesh] of this.elementMeshes) {
      if (mesh.userData.forceVisible) continue
      if (mesh.userData.occluded) {
        mesh.visible = false
        continue
      }
      if (mesh.isInstancedMesh) {
        mesh.visible = true
        continue
      }
      mesh.visible = this.frustum.intersectsObject(mesh)
    }
  }

  _performOcclusionCulling() {
    if (this.elementMeshes.size === 0) return

    const cameraPos = this.camera.position.clone()
    const elementsWithDistance = []

    for (const [id, mesh] of this.elementMeshes) {
      if (mesh.userData.forceVisible) continue
      const pos = new THREE.Vector3()
      if (mesh.isInstancedMesh) continue
      pos.setFromMatrixPosition(mesh.matrixWorld)
      const dist = pos.distanceTo(cameraPos)
      elementsWithDistance.push({ id, mesh, pos, dist })
    }

    elementsWithDistance.sort((a, b) => a.dist - b.dist)

    const occluders = []
    const maxOccluders = 30

    for (let i = 0; i < Math.min(maxOccluders, elementsWithDistance.length); i++) {
      const e = elementsWithDistance[i]
      if (!e.mesh.geometry.boundingBox) {
        e.mesh.geometry.computeBoundingBox()
      }
      occluders.push({
        id: e.id,
        pos: e.pos,
        aabb: e.mesh.geometry.boundingBox.clone().applyMatrix4(e.mesh.matrixWorld),
        dist: e.dist
      })
    }

    for (let i = maxOccluders; i < elementsWithDistance.length; i++) {
      const target = elementsWithDistance[i]
      let occluded = false

      const toTarget = new THREE.Vector3().subVectors(target.pos, cameraPos).normalize()
      const targetDist = target.dist

      for (const occluder of occluders) {
        const toOccluder = new THREE.Vector3().subVectors(occluder.pos, cameraPos).normalize()
        const dot = toTarget.dot(toOccluder)

        if (dot > 0.98 && occluder.dist < targetDist) {
          const angle = Math.acos(dot)
          const occluderSize = Math.max(
            occluder.aabb.max.x - occluder.aabb.min.x,
            occluder.aabb.max.y - occluder.aabb.min.y,
            occluder.aabb.max.z - occluder.aabb.min.z
          )
          const angularSize = Math.atan(occluderSize / 2 / occluder.dist)
          if (angle < angularSize * 0.6) {
            occluded = true
            break
          }
        }
      }

      target.mesh.userData.occluded = occluded
    }
  }

  _updateLOD() {
    const cameraPos = this.camera.position.clone()

    for (const [elementId, lodMeshes] of this.elementMeshesByLOD) {
      const primaryMesh = lodMeshes[0]
      if (!primaryMesh) continue

      const pos = new THREE.Vector3()
      if (primaryMesh.isInstancedMesh) {
        const m = new THREE.Matrix4()
        primaryMesh.getMatrixAt(0, m)
        pos.setFromMatrixPosition(m)
      } else {
        pos.setFromMatrixPosition(primaryMesh.matrixWorld)
      }

      const dist = cameraPos.distanceTo(pos)

      let targetLOD = 0
      if (dist > this.lodDistanceThresholds[2]) targetLOD = 2
      else if (dist > this.lodDistanceThresholds[1]) targetLOD = 1

      for (let lod = 0; lod < lodMeshes.length; lod++) {
        const m = lodMeshes[lod]
        if (m) {
          m.visible = (lod === targetLOD) && !m.userData.occluded
        }
      }
    }
  }

  setMeasureMode(enabled) {
    this.measureMode = enabled
    this.renderer.domElement.style.cursor = enabled ? 'crosshair' : 'default'
  }

  addElementMeshLOD(elementId, lod, mesh) {
    if (!this.elementMeshesByLOD.has(elementId)) {
      this.elementMeshesByLOD.set(elementId, [])
    }
    const lodArray = this.elementMeshesByLOD.get(elementId)
    lodArray[lod] = mesh

    if (lod === 0) {
      this.elementMeshes.set(elementId, mesh)
    }

    mesh.userData.elementId = elementId
    mesh.userData.lod = lod
    mesh.castShadow = false
    mesh.receiveShadow = false
    this.scene.add(mesh)
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

    if (!this.elementMeshesByLOD.has(elementId)) {
      this.elementMeshesByLOD.set(elementId, [])
    }
    this.elementMeshesByLOD.get(elementId)[0] = mesh
    mesh.userData.lod = 0

    this.scene.add(mesh)
    return mesh
  }

  addInstancedMesh(geometryHash, geometry, material, instances) {
    const lodMeshes = []

    for (let lod = 0; lod < 3; lod++) {
      const reducedGeo = this._createLODGeometry(geometry, lod)
      const instancedMesh = new THREE.InstancedMesh(reducedGeo, material, instances.length)
      instancedMesh.userData.geometryHash = geometryHash
      instancedMesh.userData.instances = instances
      instancedMesh.userData.lod = lod

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
      instancedMesh.frustumCulled = false
      instancedMesh.visible = (lod === 0)
      this.instanceGroups.set(`${geometryHash}_lod${lod}`, instancedMesh)
      this.scene.add(instancedMesh)
      lodMeshes.push(instancedMesh)

      for (const inst of instances) {
        if (lod === 0) {
          this.elementMeshes.set(inst.elementId, instancedMesh)
        }
        if (!this.elementMeshesByLOD.has(inst.elementId)) {
          this.elementMeshesByLOD.set(inst.elementId, [])
        }
        this.elementMeshesByLOD.get(inst.elementId)[lod] = instancedMesh
      }
    }

    return lodMeshes[0]
  }

  _createLODGeometry(baseGeometry, lod) {
    if (lod === 0) return baseGeometry

    const simplified = baseGeometry.clone()

    if (simplified instanceof THREE.BufferGeometry) {
      const pos = simplified.attributes.position
      if (pos && pos.count > 0) {
        const reduceFactor = lod === 1 ? 0.5 : 0.2
        const targetCount = Math.max(4, Math.floor(pos.count * reduceFactor))

        const newPositions = new Float32Array(targetCount * 3)
        const step = Math.max(1, Math.floor(pos.count / targetCount))

        let idx = 0
        for (let i = 0; i < pos.count && idx < targetCount; i += step) {
          newPositions[idx * 3] = pos.getX(i)
          newPositions[idx * 3 + 1] = pos.getY(i)
          newPositions[idx * 3 + 2] = pos.getZ(i)
          idx++
        }

        simplified.setAttribute('position', new THREE.BufferAttribute(newPositions, 3))
        simplified.computeVertexNormals()
      }
    }

    return simplified
  }

  highlightElement(elementId, color = 0x00aaff) {
    this.clearHighlight()

    const lodMeshes = this.elementMeshesByLOD.get(elementId)
    if (!lodMeshes || lodMeshes.length === 0) return

    const sourceMesh = lodMeshes.find(m => m && m.visible) || lodMeshes[0]
    if (!sourceMesh) return

    if (sourceMesh.isInstancedMesh) {
      const instances = sourceMesh.userData.instances
      if (!instances) return
      const idx = instances.findIndex(inst => inst.elementId === elementId)
      if (idx < 0) return

      const matrix = new THREE.Matrix4()
      sourceMesh.getMatrixAt(idx, matrix)
      const pos = new THREE.Vector3()
      const quat = new THREE.Quaternion()
      const scale = new THREE.Vector3()
      matrix.decompose(pos, quat, scale)

      let geo
      if (sourceMesh.geometry instanceof THREE.CylinderGeometry) {
        geo = new THREE.EdgesGeometry(sourceMesh.geometry)
      } else {
        geo = new THREE.EdgesGeometry(sourceMesh.geometry)
      }
      const edgesMat = new THREE.LineBasicMaterial({ color, linewidth: 2 })
      const wireframe = new THREE.LineSegments(geo, edgesMat)
      wireframe.position.copy(pos)
      wireframe.quaternion.copy(quat)
      wireframe.scale.copy(scale).multiplyScalar(1.03)
      this.highlightMesh = wireframe
      this.scene.add(this.highlightMesh)

      const bbox = new THREE.Box3().setFromObject(wireframe)
      this.boxHelper = new THREE.Box3Helper(bbox, color)
      this.scene.add(this.boxHelper)
    } else {
      const edges = new THREE.EdgesGeometry(sourceMesh.geometry)
      const lineMat = new THREE.LineBasicMaterial({ color, linewidth: 2 })
      const wireframe = new THREE.LineSegments(edges, lineMat)
      wireframe.position.copy(sourceMesh.position)
      wireframe.rotation.copy(sourceMesh.rotation)
      wireframe.scale.copy(sourceMesh.scale).multiplyScalar(1.03)
      this.highlightMesh = wireframe
      this.scene.add(this.highlightMesh)

      const bbox = new THREE.Box3().setFromObject(sourceMesh)
      this.boxHelper = new THREE.Box3Helper(bbox, color)
      this.scene.add(this.boxHelper)
    }
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
    const lodMeshes = this.elementMeshesByLOD.get(elementId)
    if (lodMeshes) {
      for (const m of lodMeshes) {
        if (m) {
          m.userData.forceVisible = visible
          m.visible = visible
        }
      }
    }
  }

  setElementOpacity(elementId, opacity) {
    const lodMeshes = this.elementMeshesByLOD.get(elementId)
    if (!lodMeshes) return
    for (const m of lodMeshes) {
      if (m && !m.isInstancedMesh) {
        m.material.transparent = true
        m.material.opacity = opacity
        m.material.needsUpdate = true
      }
    }

    const mesh = this.elementMeshes.get(elementId)
    if (mesh && mesh.isInstancedMesh) {
      mesh.material.transparent = true
      mesh.material.opacity = opacity
      mesh.material.needsUpdate = true
    }
  }

  isolateElements(elementIds) {
    const idSet = new Set(elementIds)
    for (const [id, lodMeshes] of this.elementMeshesByLOD) {
      const shouldShow = idSet.has(id)
      for (const m of lodMeshes) {
        if (m) {
          m.userData.forceVisible = shouldShow
          m.visible = shouldShow
        }
      }
    }
  }

  showAllElements() {
    for (const [id, lodMeshes] of this.elementMeshesByLOD) {
      for (const m of lodMeshes) {
        if (m) {
          m.userData.forceVisible = false
          m.userData.occluded = false
        }
      }
    }
  }

  setCategoryOpacity(category, opacity, elementsMap) {
    for (const e of elementsMap.elements) {
      if (e.category !== category) {
        this.setElementOpacity(e.id, opacity)
      }
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
    const allMeshes = new Set()
    for (const m of this.elementMeshes.values()) allMeshes.add(m)
    for (const m of this.instanceGroups.values()) allMeshes.add(m)

    for (const mesh of allMeshes) {
      if (!mesh.material) continue
      if (Array.isArray(mesh.material)) {
        mesh.material.forEach(m => {
          m.clippingPlanes = this.clippingPlanes
          m.clipShadows = true
          m.needsUpdate = true
        })
      } else {
        mesh.material.clippingPlanes = this.clippingPlanes
        mesh.material.clipShadows = true
        mesh.material.needsUpdate = true
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
    let hasVisible = false

    for (const [id, mesh] of this.elementMeshes) {
      if (!mesh.visible && !mesh.userData.forceVisible) continue
      hasVisible = true
      mesh.updateMatrixWorld(true)

      if (mesh.isInstancedMesh) {
        const tempBox = new THREE.Box3()
        const instances = mesh.userData.instances
        if (instances) {
          for (let i = 0; i < instances.length; i++) {
            const matrix = new THREE.Matrix4()
            mesh.getMatrixAt(i, matrix)
            tempBox.makeEmpty()
            tempBox.setFromObject({ geometry: mesh.geometry, matrixWorld: matrix })
            box.union(tempBox)
          }
        }
      } else {
        box.expandByObject(mesh)
      }
    }

    if (!hasVisible || box.isEmpty()) return

    const center = box.getCenter(new THREE.Vector3())
    const size = box.getSize(new THREE.Vector3())
    const maxDim = Math.max(size.x, size.y, size.z)
    const fov = this.camera.fov * (Math.PI / 180)
    const distance = Math.abs(maxDim / 2 / Math.tan(fov / 2)) * 1.5

    const direction = new THREE.Vector3(0.5, 0.8, 0.7).normalize()
    const newPos = center.clone().add(direction.multiplyScalar(distance))

    this.flyTo(
      { x: newPos.x, y: newPos.y, z: newPos.z },
      { x: center.x, y: center.y, z: center.z }
    )
  }

  addMeasurementLine(points, color = 0xffff00) {
    const vertices = points.map(p => new THREE.Vector3(p.x, p.y, p.z))
    const geometry = new THREE.BufferGeometry().setFromPoints(vertices)
    const material = new THREE.LineBasicMaterial({ color, linewidth: 3, depthTest: false })
    const line = new THREE.Line(geometry, material)
    line.renderOrder = 999
    this.scene.add(line)
    this.measurementObjects.push(line)

    for (const p of points) {
      const sphereGeo = new THREE.SphereGeometry(80, 16, 16)
      const sphereMat = new THREE.MeshBasicMaterial({ color, depthTest: false })
      const sphere = new THREE.Mesh(sphereGeo, sphereMat)
      sphere.position.set(p.x, p.y, p.z)
      sphere.renderOrder = 999
      this.scene.add(sphere)
      this.measurementObjects.push(sphere)
    }
    return line
  }

  addMeasurementLabel(text, position, color = '#ffffff') {
    const canvas = document.createElement('canvas')
    canvas.width = 512
    canvas.height = 128
    const ctx = canvas.getContext('2d')

    ctx.fillStyle = 'rgba(0,0,0,0.8)'
    ctx.fillRect(0, 0, 512, 128)
    ctx.strokeStyle = color
    ctx.lineWidth = 4
    ctx.strokeRect(4, 4, 504, 120)
    ctx.fillStyle = color
    ctx.font = 'bold 48px Arial'
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(text, 256, 64)

    const texture = new THREE.CanvasTexture(canvas)
    texture.needsUpdate = true
    const spriteMat = new THREE.SpriteMaterial({
      map: texture,
      depthTest: false,
      transparent: true
    })
    const sprite = new THREE.Sprite(spriteMat)
    sprite.position.set(position.x, position.y + 600, position.z)
    sprite.scale.set(2500, 625, 1)
    sprite.renderOrder = 1000
    this.scene.add(sprite)
    this.measurementObjects.push(sprite)
    return sprite
  }

  removeMeasurement(index) {
    const obj = this.measurementObjects[index]
    if (obj) {
      this.scene.remove(obj)
    }
  }

  clearMeasurements() {
    for (const obj of this.measurementObjects) {
      this.scene.remove(obj)
    }
    this.measurementObjects = []
  }

  addCollisionMarker(position, color = 0xff0000) {
    const geometry = new THREE.SphereGeometry(300, 16, 16)
    const material = new THREE.MeshBasicMaterial({
      color,
      transparent: true,
      opacity: 0.8,
      depthTest: false
    })
    const sphere = new THREE.Mesh(geometry, material)
    sphere.position.set(position.x, position.y, position.z)
    sphere.renderOrder = 998
    this.scene.add(sphere)
    this.measurementObjects.push(sphere)
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

    const allMeshes = new Set()
    for (const m of this.elementMeshes.values()) allMeshes.add(m)
    for (const m of this.instanceGroups.values()) allMeshes.add(m)
    for (const arr of this.elementMeshesByLOD.values()) {
      for (const m of arr) if (m) allMeshes.add(m)
    }

    for (const mesh of allMeshes) {
      if (mesh.geometry) mesh.geometry.dispose()
      if (mesh.material) {
        if (Array.isArray(mesh.material)) {
          mesh.material.forEach(m => m.dispose())
        } else {
          mesh.material.dispose()
        }
      }
    }

    this.clearMeasurements()
    this.renderer.dispose()
    this.controls.dispose()
    this.elementMeshes.clear()
    this.elementMeshesByLOD.clear()
    this.instanceGroups.clear()
  }
}
