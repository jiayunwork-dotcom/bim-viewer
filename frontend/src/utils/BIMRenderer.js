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
    this.elementInstanceInfo = new Map()
    this.instanceGroups = new Map()
    this.clippingPlanes = []
    this.highlightObjects = []
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
    this.annotationPins = new Map()
    this.onAnnotationClick = null
    this.onAnnotationDblClick = null

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
    canvas.addEventListener('dblclick', (e) => this._onDblClick(e))
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

    const pinId = this.getAnnotationPinAtPosition(this.mouse)
    if (pinId && this.onAnnotationClick) {
      this.onAnnotationClick(pinId)
      return
    }

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

  _onDblClick(event) {
    const rect = this.renderer.domElement.getBoundingClientRect()
    this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1
    this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1

    this.raycaster.setFromCamera(this.mouse, this.camera)

    const pinId = this.getAnnotationPinAtPosition(this.mouse)
    if (pinId && this.onAnnotationDblClick) {
      this.onAnnotationDblClick(pinId)
      return
    }

    const meshes = Array.from(this.elementMeshes.values()).filter(m => m.visible)
    const intersects = this.raycaster.intersectObjects(meshes, false)

    if (intersects.length > 0) {
      const hit = intersects[0]
      const point = hit.point
      let elementId = hit.object.userData.elementId

      if (hit.object.isInstancedMesh && hit.instanceId !== undefined) {
        const instancedData = hit.object.userData.instances
        if (instancedData && instancedData[hit.instanceId]) {
          elementId = instancedData[hit.instanceId].elementId
        }
      }

      if (this.onAnnotationDblClick) {
        this.onAnnotationDblClick(null, elementId, { x: point.x, y: point.y, z: point.z })
      }
    } else {
      const groundPoint = this.getGroundPointAtPosition(this.mouse)
      if (groundPoint && this.onAnnotationDblClick) {
        this.onAnnotationDblClick(null, null, groundPoint)
      }
    }
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
          if (m.userData.forceHidden) {
            m.visible = false
          } else {
            m.visible = (lod === targetLOD) && !m.userData.occluded
          }
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
    if (!this.elementMeshesByLOD.has(elementId)) {
      this.elementMeshesByLOD.set(elementId, [])
    }

    for (let lod = 0; lod < 3; lod++) {
      const lodGeo = this._createLODGeometry(geometry, lod)
      const mesh = new THREE.Mesh(lodGeo, material)
      mesh.userData.elementId = elementId
      mesh.userData.lod = lod
      if (position) {
        mesh.position.set(position.x, position.y, position.z)
      }
      mesh.castShadow = false
      mesh.receiveShadow = false
      mesh.frustumCulled = false
      mesh.visible = (lod === 0)
      this.elementMeshesByLOD.get(elementId)[lod] = mesh
      if (lod === 0) {
        this.elementMeshes.set(elementId, mesh)
      }
      this.scene.add(mesh)
    }

    return this.elementMeshesByLOD.get(elementId)[0]
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

        if (lod === 0) {
          this.elementInstanceInfo.set(inst.elementId, {
            geometryHash,
            instanceIndex: i,
            baseScale: { ...dummy.scale },
            basePosition: { ...dummy.position },
            baseRotation: { ...dummy.rotation }
          })
        }
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
    const geo = baseGeometry
    if (geo instanceof THREE.CylinderGeometry) {
      const params = geo.parameters
      let segments = 16
      if (lod === 1) segments = 8
      else if (lod === 2) segments = 4
      return new THREE.CylinderGeometry(
        params.radiusTop, params.radiusBottom, params.height,
        segments, 1, params.openEnded
      )
    }

    if (geo instanceof THREE.BoxGeometry) {
      const params = geo.parameters
      const width = params.width
      const height = params.height
      const depth = params.depth

      if (lod === 0) {
        const maxDim = Math.max(width, height, depth)
        let sx = 1, sy = 1, sz = 1
        if (maxDim > 0) {
          sx = Math.max(1, Math.round(width / 1000))
          sy = Math.max(1, Math.round(height / 1000))
          sz = Math.max(1, Math.round(depth / 1000))
        }
        sx = Math.min(sx, 8)
        sy = Math.min(sy, 8)
        sz = Math.min(sz, 8)
        return new THREE.BoxGeometry(width, height, depth, sx, sy, sz)
      } else if (lod === 1) {
        return new THREE.BoxGeometry(width, height, depth, 1, 1, 1)
      } else {
        const w = width * 0.98
        const h = height * 0.98
        const d = depth * 0.98
        const simple = new THREE.BufferGeometry()
        const hw = w / 2, hh = h / 2, hd = d / 2
        const positions = new Float32Array([
          -hw, -hh, -hd,  hw, -hh, -hd,  hw, hh, -hd,  -hw, hh, -hd,
          -hw, -hh, hd,   hw, -hh, hd,   hw, hh, hd,   -hw, hh, hd
        ])
        const indices = [
          0,1,2, 0,2,3,
          4,6,5, 4,7,6,
          0,4,5, 0,5,1,
          2,6,7, 2,7,3,
          0,3,7, 0,7,4,
          1,5,6, 1,6,2
        ]
        simple.setAttribute('position', new THREE.BufferAttribute(positions, 3))
        simple.setIndex(indices)
        simple.computeVertexNormals()
        return simple
      }
    }

    return lod === 0 ? baseGeometry : baseGeometry.clone()
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

      const geo = new THREE.EdgesGeometry(sourceMesh.geometry)
      const edgesMat = new THREE.LineBasicMaterial({ color, linewidth: 2 })
      const wireframe = new THREE.LineSegments(geo, edgesMat)
      wireframe.position.copy(pos)
      wireframe.quaternion.copy(quat)
      wireframe.scale.copy(scale).multiplyScalar(1.03)
      wireframe.userData.isHighlight = true
      this.scene.add(wireframe)
      this.highlightObjects.push(wireframe)

      const bbox = new THREE.Box3().setFromObject(wireframe)
      const boxHelper = new THREE.Box3Helper(bbox, color)
      boxHelper.userData.isHighlight = true
      this.scene.add(boxHelper)
      this.highlightObjects.push(boxHelper)
    } else {
      const edges = new THREE.EdgesGeometry(sourceMesh.geometry)
      const lineMat = new THREE.LineBasicMaterial({ color, linewidth: 2 })
      const wireframe = new THREE.LineSegments(edges, lineMat)
      wireframe.position.copy(sourceMesh.position)
      wireframe.rotation.copy(sourceMesh.rotation)
      wireframe.scale.copy(sourceMesh.scale).multiplyScalar(1.03)
      wireframe.userData.isHighlight = true
      this.scene.add(wireframe)
      this.highlightObjects.push(wireframe)

      const bbox = new THREE.Box3().setFromObject(sourceMesh)
      const boxHelper = new THREE.Box3Helper(bbox, color)
      boxHelper.userData.isHighlight = true
      this.scene.add(boxHelper)
      this.highlightObjects.push(boxHelper)
    }
  }

  clearHighlight() {
    for (const obj of this.highlightObjects) {
      this.scene.remove(obj)
      if (obj.geometry) obj.geometry.dispose()
      if (obj.material) {
        if (Array.isArray(obj.material)) {
          for (const m of obj.material) m.dispose()
        } else {
          obj.material.dispose()
        }
      }
    }
    this.highlightObjects = []
  }

  setElementVisibility(elementId, visible) {
    const instInfo = this.elementInstanceInfo.get(elementId)
    if (instInfo) {
      const { geometryHash, instanceIndex, baseScale, basePosition, baseRotation } = instInfo
      const dummy = new THREE.Object3D()
      dummy.position.set(basePosition.x, basePosition.y, basePosition.z)
      dummy.rotation.set(baseRotation.x, baseRotation.y, baseRotation.z)
      if (visible) {
        dummy.scale.set(baseScale.x, baseScale.y, baseScale.z)
      } else {
        dummy.scale.set(0, 0, 0)
      }
      dummy.updateMatrix()

      for (let lod = 0; lod < 3; lod++) {
        const mesh = this.instanceGroups.get(`${geometryHash}_lod${lod}`)
        if (mesh) {
          mesh.setMatrixAt(instanceIndex, dummy.matrix)
          mesh.instanceMatrix.needsUpdate = true
        }
      }
      return
    }

    const lodMeshes = this.elementMeshesByLOD.get(elementId)
    if (lodMeshes) {
      for (const m of lodMeshes) {
        if (m) {
          m.userData.forceHidden = !visible
          if (!visible) {
            m.visible = false
          }
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

    for (const [elementId, instInfo] of this.elementInstanceInfo) {
      const { geometryHash, instanceIndex, baseScale, basePosition, baseRotation } = instInfo
      const shouldShow = idSet.has(elementId)
      const dummy = new THREE.Object3D()
      dummy.position.set(basePosition.x, basePosition.y, basePosition.z)
      dummy.rotation.set(baseRotation.x, baseRotation.y, baseRotation.z)
      if (shouldShow) {
        dummy.scale.set(baseScale.x, baseScale.y, baseScale.z)
      } else {
        dummy.scale.set(0, 0, 0)
      }
      dummy.updateMatrix()
      for (let lod = 0; lod < 3; lod++) {
        const mesh = this.instanceGroups.get(`${geometryHash}_lod${lod}`)
        if (mesh) {
          mesh.setMatrixAt(instanceIndex, dummy.matrix)
          mesh.instanceMatrix.needsUpdate = true
        }
      }
    }

    for (const [id, lodMeshes] of this.elementMeshesByLOD) {
      const shouldShow = idSet.has(id)
      const hasInst = this.elementInstanceInfo.has(id)
      if (hasInst) continue
      for (const m of lodMeshes) {
        if (m) {
          m.userData.forceHidden = !shouldShow
          if (!shouldShow) m.visible = false
        }
      }
    }
  }

  showAllElements() {
    for (const [elementId, instInfo] of this.elementInstanceInfo) {
      const { geometryHash, instanceIndex, baseScale, basePosition, baseRotation } = instInfo
      const dummy = new THREE.Object3D()
      dummy.position.set(basePosition.x, basePosition.y, basePosition.z)
      dummy.rotation.set(baseRotation.x, baseRotation.y, baseRotation.z)
      dummy.scale.set(baseScale.x, baseScale.y, baseScale.z)
      dummy.updateMatrix()
      for (let lod = 0; lod < 3; lod++) {
        const mesh = this.instanceGroups.get(`${geometryHash}_lod${lod}`)
        if (mesh) {
          mesh.setMatrixAt(instanceIndex, dummy.matrix)
          mesh.instanceMatrix.needsUpdate = true
        }
      }
    }

    for (const [id, lodMeshes] of this.elementMeshesByLOD) {
      for (let lod = 0; lod < lodMeshes.length; lod++) {
        const m = lodMeshes[lod]
        if (m) {
          m.userData.forceHidden = false
          m.userData.occluded = false
          m.visible = (lod === 0)
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

  addAnnotationPin(annotation, onClick) {
    const existing = this.annotationPins.get(annotation.id)
    if (existing) {
      this.removeAnnotationPin(annotation.id)
    }

    const color = this._getAnnotationColor(annotation.priority, annotation.status)
    const opacity = annotation.status === 'closed' ? 0.4 : 1.0

    const group = new THREE.Group()
    group.userData = { annotationId: annotation.id, isAnnotationPin: true }

    const pinHeight = 600
    const headRadius = 200
    const stickRadius = 40

    const stickGeo = new THREE.CylinderGeometry(stickRadius, stickRadius, pinHeight, 8)
    const stickMat = new THREE.MeshBasicMaterial({
      color,
      transparent: true,
      opacity: opacity * 0.8,
      depthTest: false
    })
    const stick = new THREE.Mesh(stickGeo, stickMat)
    stick.position.y = pinHeight / 2
    stick.renderOrder = 998
    group.add(stick)

    const headGeo = new THREE.SphereGeometry(headRadius, 16, 16)
    const headMat = new THREE.MeshBasicMaterial({
      color,
      transparent: true,
      opacity,
      depthTest: false
    })
    const head = new THREE.Mesh(headGeo, headMat)
    head.position.y = pinHeight + headRadius
    head.renderOrder = 999
    head.userData = { annotationId: annotation.id, isAnnotationPinHead: true }
    group.add(head)

    const ringGeo = new THREE.RingGeometry(headRadius + 30, headRadius + 60, 32)
    const ringMat = new THREE.MeshBasicMaterial({
      color,
      transparent: true,
      opacity: opacity * 0.5,
      depthTest: false,
      side: THREE.DoubleSide
    })
    const ring = new THREE.Mesh(ringGeo, ringMat)
    ring.position.y = pinHeight + headRadius
    ring.rotation.x = -Math.PI / 2
    ring.renderOrder = 999
    group.add(ring)

    group.position.set(annotation.position[0], annotation.position[1], annotation.position[2])

    this.scene.add(group)
    this.annotationPins.set(annotation.id, group)

    return group
  }

  removeAnnotationPin(annotationId) {
    const group = this.annotationPins.get(annotationId)
    if (group) {
      this.scene.remove(group)
      group.traverse(child => {
        if (child.geometry) child.geometry.dispose()
        if (child.material) child.material.dispose()
      })
      this.annotationPins.delete(annotationId)
    }
  }

  updateAnnotationPin(annotation) {
    const group = this.annotationPins.get(annotation.id)
    if (!group) return

    const color = this._getAnnotationColor(annotation.priority, annotation.status)
    const opacity = annotation.status === 'closed' ? 0.4 : 1.0

    group.traverse(child => {
      if (child.isMesh && child.material) {
        child.material.color.setHex(color)
        child.material.opacity = opacity * (child.userData.isAnnotationPinHead ? 1.0 : 0.8)
        child.material.needsUpdate = true
      }
    })
  }

  clearAnnotationPins() {
    for (const [id] of this.annotationPins) {
      this.removeAnnotationPin(id)
    }
  }

  _getAnnotationColor(priority, status) {
    const colors = {
      urgent: 0xff4444,
      normal: 0x409eff,
      low: 0x909399
    }
    return colors[priority] || 0x409eff
  }

  getAnnotationPinAtPosition(mouse) {
    this.raycaster.setFromCamera(mouse, this.camera)

    const pinMeshes = []
    for (const [, group] of this.annotationPins) {
      group.traverse(child => {
        if (child.isMesh) {
          pinMeshes.push(child)
        }
      })
    }

    if (pinMeshes.length === 0) return null

    const intersects = this.raycaster.intersectObjects(pinMeshes, false)
    if (intersects.length > 0) {
      let obj = intersects[0].object
      while (obj && !obj.userData.annotationId) {
        obj = obj.parent
      }
      if (obj && obj.userData.annotationId) {
        return obj.userData.annotationId
      }
    }
    return null
  }

  getGroundPointAtPosition(mouse) {
    this.raycaster.setFromCamera(mouse, this.camera)
    const plane = new THREE.Plane(new THREE.Vector3(0, 1, 0), 0)
    const point = new THREE.Vector3()
    this.raycaster.ray.intersectPlane(plane, point)
    return point ? { x: point.x, y: point.y, z: point.z } : null
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
    this.clearHighlight()
    this.clearAnnotationPins()
    this.renderer.dispose()
    this.controls.dispose()
    this.elementMeshes.clear()
    this.elementMeshesByLOD.clear()
    this.elementInstanceInfo.clear()
    this.instanceGroups.clear()
  }
}
