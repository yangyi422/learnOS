import type {
  ContextState,
  FrameContext,
  QualityTier,
  RenderMetrics,
  Renderer,
  RendererOptions,
  SceneId,
  ShaderScene,
  SurfaceLight,
  UniformMap,
} from './types'
import { FrameSampler, targetPixelRatio } from './quality'
import { glassFragmentShader, glassVertexShader, scenes } from './shaders'

const CONTEXT_ATTRIBUTES: WebGLContextAttributes = {
  alpha: false,
  antialias: false,
  depth: false,
  stencil: false,
  premultipliedAlpha: false,
  // A frozen frame has to survive a pause, so the drawing buffer is preserved.
  preserveDrawingBuffer: true,
  powerPreference: 'low-power',
}

const RESTORE_TIMEOUT_MS = 8000
const MAX_PENDING_GPU_QUERIES = 4
const FULL_SCREEN_TRIANGLE = new Float32Array([-1, -1, 3, -1, -1, 3])

/** Total frames drawn by every surface; a dev-only diagnostic for lifecycle tests. */
let renderedFrames = 0

export function renderedFrameCount(): number {
  return renderedFrames
}

/** Live renderer instances; dev diagnostics for lifecycle tests. */
let liveInstances = 0
/** Live GL programs across instances; proves scene switches free the old one. */
let livePrograms = 0

export function rendererInstanceCount(): number {
  return liveInstances
}

export function liveProgramCount(): number {
  return livePrograms
}

interface BuiltScene {
  program: WebGLProgram
  uniforms: UniformMap
  vertexArray: WebGLVertexArrayObject
  buffer: WebGLBuffer
}

interface BuiltGlass {
  program: WebGLProgram
  uniforms: UniformMap
  vertexArray: WebGLVertexArrayObject
  buffer: WebGLBuffer
  framebuffer: WebGLFramebuffer | null
  texture: WebGLTexture | null
  texWidth: number
  texHeight: number
}

function compile(gl: WebGL2RenderingContext, type: number, source: string): WebGLShader {
  const shader = gl.createShader(type)
  if (!shader) throw new Error('shader creation failed')
  gl.shaderSource(shader, source)
  gl.compileShader(shader)
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    const log = gl.getShaderInfoLog(shader) ?? 'shader compile failed'
    gl.deleteShader(shader)
    throw new Error(log)
  }
  return shader
}

function buildScene(gl: WebGL2RenderingContext, scene: ShaderScene): BuiltScene {
  const vertexShader = compile(gl, gl.VERTEX_SHADER, scene.vertexShader)
  const fragmentShader = compile(gl, gl.FRAGMENT_SHADER, scene.fragmentShader)
  const program = gl.createProgram()
  if (!program) throw new Error('program creation failed')
  gl.attachShader(program, vertexShader)
  gl.attachShader(program, fragmentShader)
  gl.linkProgram(program)
  gl.deleteShader(vertexShader)
  gl.deleteShader(fragmentShader)
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    const log = gl.getProgramInfoLog(program) ?? 'program link failed'
    gl.deleteProgram(program)
    throw new Error(log)
  }

  const uniforms: UniformMap = {}
  for (const name of scene.uniformNames) uniforms[name] = gl.getUniformLocation(program, name)

  const vertexArray = gl.createVertexArray()
  const buffer = gl.createBuffer()
  if (!vertexArray || !buffer) throw new Error('buffer creation failed')
  gl.bindVertexArray(vertexArray)
  gl.bindBuffer(gl.ARRAY_BUFFER, buffer)
  gl.bufferData(gl.ARRAY_BUFFER, FULL_SCREEN_TRIANGLE, gl.STATIC_DRAW)
  const position = gl.getAttribLocation(program, 'aPosition')
  gl.enableVertexAttribArray(position)
  gl.vertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0)
  gl.bindVertexArray(null)

  livePrograms += 1
  return { program, uniforms, vertexArray, buffer }
}

export function createRenderer(canvas: HTMLCanvasElement, options: RendererOptions): Renderer {
  const context = canvas.getContext('webgl2', CONTEXT_ATTRIBUTES)
  if (!context) throw new Error('WebGL2 context unavailable')
  const gl: WebGL2RenderingContext = context
  liveInstances += 1

  let built: Partial<Record<SceneId, BuiltScene>> = {}
  let glass: BuiltGlass | null = null
  let lastDrawCalls = 1
  let activeScene: SceneId = options.scene
  let light: SurfaceLight = options.light
  let strength = options.strength ?? 1
  let quality: QualityTier = options.quality ?? 'high'
  let running = false
  let frameHandle = 0
  let restoreTimer = 0
  let lastFrameAt = 0
  let elapsed = 0
  let cssWidth = 0
  let cssHeight = 0
  let pixelWidth = 0
  let pixelHeight = 0
  let pixelRatio = 1
  let contextState: ContextState = 'ok'
  let disposed = false
  const restoreTimeoutMs = options.restoreTimeoutMs ?? RESTORE_TIMEOUT_MS

  const sampler = new FrameSampler()

  /* ---------------- GPU timing (optional) ---------------- */
  const timerExtension = gl.getExtension('EXT_disjoint_timer_query_webgl2')
  const gpuAvailable = Boolean(timerExtension)
  let activeQuery: WebGLQuery | null = null
  const pendingQueries: WebGLQuery[] = []
  let gpuMs: number | null = null

  function beginGpuQuery(): void {
    if (!timerExtension || activeQuery || pendingQueries.length >= MAX_PENDING_GPU_QUERIES) return
    const query = gl.createQuery()
    if (!query) return
    gl.beginQuery(timerExtension.TIME_ELAPSED_EXT, query)
    activeQuery = query
  }

  function endGpuQuery(): void {
    if (!timerExtension || !activeQuery) return
    gl.endQuery(timerExtension.TIME_ELAPSED_EXT)
    pendingQueries.push(activeQuery)
    activeQuery = null
  }

  function pollGpuQueries(): void {
    if (!timerExtension) return
    for (let index = pendingQueries.length - 1; index >= 0; index--) {
      const query = pendingQueries[index]!
      if (!gl.getQueryParameter(query, gl.QUERY_RESULT_AVAILABLE)) continue
      const disjoint = Boolean(gl.getParameter(timerExtension.GPU_DISJOINT_EXT))
      if (!disjoint) {
        const nanoseconds = Number(gl.getQueryParameter(query, gl.QUERY_RESULT))
        if (Number.isFinite(nanoseconds) && nanoseconds > 0) {
          gpuMs = Math.round((nanoseconds / 1e6) * 1000) / 1000
        }
      }
      gl.deleteQuery(query)
      pendingQueries.splice(index, 1)
    }
  }

  /* ---------------- metrics ---------------- */
  function reportMetrics(): void {
    if (!options.onMetrics) return
    const metrics: RenderMetrics = {
      fps: sampler.value,
      drawCalls: lastDrawCalls,
      // Backing-store pixels, not CSS pixels: DPR is visible in this number.
      width: pixelWidth,
      height: pixelHeight,
      aspect: cssHeight > 0 ? Math.round((cssWidth / cssHeight) * 100) / 100 : 0,
      pixelRatio,
      gpuAvailable,
      gpuMs,
      context: contextState,
    }
    options.onMetrics(metrics)
  }

  /* ---------------- context lifecycle ---------------- */
  function releaseScenes(): void {
    for (const scene of Object.values(built)) {
      if (!scene) continue
      gl.deleteBuffer(scene.buffer)
      gl.deleteVertexArray(scene.vertexArray)
      gl.deleteProgram(scene.program)
      livePrograms -= 1
    }
    built = {}
  }

  function buildActive(): void {
    releaseScenes()
    built[activeScene] = buildScene(gl, scenes[activeScene])
  }

  function buildGlass(): void {
    glass = null
    if (!options.glass) return
    try {
      const vertexShader = compile(gl, gl.VERTEX_SHADER, glassVertexShader)
      const fragmentShader = compile(gl, gl.FRAGMENT_SHADER, glassFragmentShader)
      const program = gl.createProgram()
      if (!program) throw new Error('glass program creation failed')
      gl.attachShader(program, vertexShader)
      gl.attachShader(program, fragmentShader)
      gl.linkProgram(program)
      gl.deleteShader(vertexShader)
      gl.deleteShader(fragmentShader)
      if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
        const log = gl.getProgramInfoLog(program) ?? 'glass program link failed'
        gl.deleteProgram(program)
        throw new Error(log)
      }
      const uniforms: UniformMap = {}
      for (const name of ['uBg', 'uResolution', 'uRect', 'uRadius', 'uStrength', 'uTheme']) {
        uniforms[name] = gl.getUniformLocation(program, name)
      }
      const vertexArray = gl.createVertexArray()
      const buffer = gl.createBuffer()
      if (!vertexArray || !buffer) throw new Error('glass buffer creation failed')
      gl.bindVertexArray(vertexArray)
      gl.bindBuffer(gl.ARRAY_BUFFER, buffer)
      gl.bufferData(gl.ARRAY_BUFFER, FULL_SCREEN_TRIANGLE, gl.STATIC_DRAW)
      const position = gl.getAttribLocation(program, 'aPosition')
      gl.enableVertexAttribArray(position)
      gl.vertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0)
      gl.bindVertexArray(null)
      glass = { program, uniforms, vertexArray, buffer, framebuffer: null, texture: null, texWidth: 0, texHeight: 0 }
      livePrograms += 1
    } catch {
      // Refraction is optional: a failed glass program must not break the ambient.
      glass = null
    }
  }

  function ensureGlassTarget(): void {
    if (!glass) return
    if (!glass.texture) {
      const texture = gl.createTexture()
      if (!texture) return
      glass.texture = texture
      gl.bindTexture(gl.TEXTURE_2D, texture)
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
      glass.texWidth = 0
      glass.texHeight = 0
    }
    if (glass.texWidth !== pixelWidth || glass.texHeight !== pixelHeight) {
      gl.bindTexture(gl.TEXTURE_2D, glass.texture)
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, pixelWidth, pixelHeight, 0, gl.RGBA, gl.UNSIGNED_BYTE, null)
      glass.texWidth = pixelWidth
      glass.texHeight = pixelHeight
    }
    if (!glass.framebuffer) glass.framebuffer = gl.createFramebuffer()
    gl.bindFramebuffer(gl.FRAMEBUFFER, glass.framebuffer)
    gl.framebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, glass.texture, 0)
    gl.bindFramebuffer(gl.FRAMEBUFFER, null)
  }

  function disposeGlass(): void {
    if (!glass) return
    if (glass.framebuffer) gl.deleteFramebuffer(glass.framebuffer)
    if (glass.texture) gl.deleteTexture(glass.texture)
    gl.deleteBuffer(glass.buffer)
    gl.deleteVertexArray(glass.vertexArray)
    gl.deleteProgram(glass.program)
    livePrograms -= 1
    glass = null
  }

  function resize(): void {
    if (disposed) return
    cssWidth = canvas.clientWidth
    cssHeight = canvas.clientHeight
    if (!cssWidth || !cssHeight) return
    pixelRatio = targetPixelRatio(window.devicePixelRatio || 1, quality)
    const nextWidth = Math.max(1, Math.round(cssWidth * pixelRatio))
    const nextHeight = Math.max(1, Math.round(cssHeight * pixelRatio))
    if (nextWidth !== pixelWidth || nextHeight !== pixelHeight) {
      pixelWidth = nextWidth
      pixelHeight = nextHeight
      canvas.width = pixelWidth
      canvas.height = pixelHeight
    }
    gl.viewport(0, 0, pixelWidth, pixelHeight)
    reportMetrics()
    // While paused or in single-frame mode the frozen frame still has to match
    // the new size, so redraw exactly once.
    if (!frameHandle) draw(performance.now(), false)
  }

  /* ---------------- drawing ---------------- */
  function draw(now: number, advance: boolean): void {
    // Drawing on a lost context produces GL errors; the frozen frame stays visible.
    if (disposed || contextState !== 'ok') return
    const builtScene = built[activeScene]
    const shader = scenes[activeScene]
    if (!builtScene || !shader || !pixelWidth || !pixelHeight) return
    if (advance) {
      elapsed += Math.max(0, now - lastFrameAt) / 1000
      lastFrameAt = now
    }
    const frame: FrameContext = { width: pixelWidth, height: pixelHeight, time: elapsed, strength, light }
    const glassOptions = options.glass
    const rect = glassOptions?.enabled() ? glassOptions.getRect() : null
    const useGlass = Boolean(rect && glass && glassOptions && pixelWidth && pixelHeight)
    beginGpuQuery()
    if (useGlass && glass && glassOptions && rect) {
      ensureGlassTarget()
      gl.bindFramebuffer(gl.FRAMEBUFFER, glass.framebuffer)
      gl.viewport(0, 0, pixelWidth, pixelHeight)
      gl.useProgram(builtScene.program)
      gl.bindVertexArray(builtScene.vertexArray)
      shader.write(gl, builtScene.uniforms, frame)
      gl.drawArrays(gl.TRIANGLES, 0, 3)

      gl.bindFramebuffer(gl.FRAMEBUFFER, null)
      gl.viewport(0, 0, pixelWidth, pixelHeight)
      const canvasRect = canvas.getBoundingClientRect()
      const scaleX = pixelWidth / Math.max(1, canvasRect.width)
      const scaleY = pixelHeight / Math.max(1, canvasRect.height)
      const width = rect.width * scaleX
      const height = rect.height * scaleY
      const x = (rect.left - canvasRect.left) * scaleX
      const yTop = (rect.top - canvasRect.top) * scaleY
      gl.useProgram(glass.program)
      gl.bindVertexArray(glass.vertexArray)
      gl.activeTexture(gl.TEXTURE0)
      gl.bindTexture(gl.TEXTURE_2D, glass.texture)
      gl.uniform1i(glass.uniforms.uBg ?? null, 0)
      gl.uniform2f(glass.uniforms.uResolution ?? null, pixelWidth, pixelHeight)
      gl.uniform4f(glass.uniforms.uRect ?? null, x, pixelHeight - yTop - height, width, height)
      gl.uniform1f(glass.uniforms.uRadius ?? null, (glassOptions.radius ?? 16) * scaleX)
      gl.uniform1f(glass.uniforms.uStrength ?? null, (glassOptions.strength ?? 22) * scaleX)
      gl.uniform1f(glass.uniforms.uTheme ?? null, glassOptions.getTheme?.() === 'dark' ? 1 : 0)
      gl.drawArrays(gl.TRIANGLES, 0, 3)
      lastDrawCalls = 2
    } else {
      gl.useProgram(builtScene.program)
      gl.bindVertexArray(builtScene.vertexArray)
      shader.write(gl, builtScene.uniforms, frame)
      gl.drawArrays(gl.TRIANGLES, 0, 3)
      lastDrawCalls = 1
    }
    endGpuQuery()
    pollGpuQueries()
    renderedFrames += 1
    if (advance) {
      const fps = sampler.sample(now)
      if (fps !== null) reportMetrics()
    }
  }

  function loop(now: number): void {
    if (!running) {
      frameHandle = 0
      return
    }
    if (document.hidden) {
      frameHandle = 0
      return
    }
    draw(now, true)
    frameHandle = requestAnimationFrame(loop)
  }

  function startLoop(): void {
    if (!running || frameHandle) return
    lastFrameAt = performance.now()
    sampler.reset()
    frameHandle = requestAnimationFrame(loop)
  }

  function stopLoop(): void {
    if (frameHandle) cancelAnimationFrame(frameHandle)
    frameHandle = 0
  }

  function handleVisibility(): void {
    if (document.hidden) {
      stopLoop()
      // A hidden tab must not keep reporting a stale live frame rate.
      sampler.reset()
      reportMetrics()
    } else if (running) {
      startLoop()
    }
  }

  function handleContextLost(event: Event): void {
    event.preventDefault()
    stopLoop()
    sampler.reset()
    // Timer queries are invalid on a lost context; drop them without reading.
    for (const query of pendingQueries) gl.deleteQuery(query)
    pendingQueries.length = 0
    if (activeQuery) gl.deleteQuery(activeQuery)
    activeQuery = null
    gpuMs = null
    contextState = 'lost'
    options.onContext?.('lost')
    window.clearTimeout(restoreTimer)
    restoreTimer = window.setTimeout(() => {
      // A dispose during the wait must not resurrect anything.
      if (disposed) return
      contextState = 'unavailable'
      options.onContext?.('unavailable')
    }, restoreTimeoutMs)
  }

  function handleContextRestored(): void {
    // After the deadline the renderer stays degraded until the host recreates it.
    if (contextState === 'unavailable') return
    window.clearTimeout(restoreTimer)
    try {
      buildActive()
      buildGlass()
      resize()
      contextState = 'ok'
      options.onContext?.('ok')
      if (running) startLoop()
    } catch {
      contextState = 'unavailable'
      options.onContext?.('unavailable')
    }
  }

  /* ---------------- resize scheduling ---------------- */
  let resizeQueued = false
  let ratioMedia: MediaQueryList | null = null

  /** Coalesces bursts of size changes into one buffer reallocation per frame. */
  function queueResize(): void {
    if (resizeQueued || disposed) return
    resizeQueued = true
    requestAnimationFrame(() => {
      resizeQueued = false
      resize()
    })
  }

  function handlePixelRatioChange(): void {
    queueResize()
    watchPixelRatio()
  }

  function stopPixelRatioWatch(): void {
    const media = ratioMedia
    if (!media) return
    media.removeEventListener('change', handlePixelRatioChange)
    ratioMedia = null
  }

  /** Moving the window to another monitor changes DPR without changing CSS size. */
  function watchPixelRatio(): void {
    if (typeof window.matchMedia !== 'function') return
    ratioMedia?.removeEventListener('change', handlePixelRatioChange)
    ratioMedia = window.matchMedia(`(resolution: ${window.devicePixelRatio}dppx)`)
    ratioMedia.addEventListener('change', handlePixelRatioChange)
  }

  canvas.addEventListener('webglcontextlost', handleContextLost, false)
  canvas.addEventListener('webglcontextrestored', handleContextRestored, false)
  document.addEventListener('visibilitychange', handleVisibility)
  const observer = new ResizeObserver(() => queueResize())
  observer.observe(canvas)
  watchPixelRatio()

  try {
    buildActive()
    buildGlass()
    resize()
  } catch (error) {
    // Construction failed after the context existed: undo every resource so a
    // failed attempt leaves no instance, program, listener or context behind.
    observer.disconnect()
    stopPixelRatioWatch()
    document.removeEventListener('visibilitychange', handleVisibility)
    canvas.removeEventListener('webglcontextlost', handleContextLost)
    canvas.removeEventListener('webglcontextrestored', handleContextRestored)
    for (const query of pendingQueries) gl.deleteQuery(query)
    pendingQueries.length = 0
    if (activeQuery) gl.deleteQuery(activeQuery)
    activeQuery = null
    releaseScenes()
    gl.getExtension('WEBGL_lose_context')?.loseContext()
    liveInstances = Math.max(0, liveInstances - 1)
    throw error
  }

  return {
    setScene(next: SceneId) {
      if (disposed || next === activeScene) return
      activeScene = next
      buildActive()
      if (!frameHandle) draw(performance.now(), false)
    },
    setLight(next: SurfaceLight) {
      if (disposed) return
      light = next
      if (!frameHandle) draw(performance.now(), false)
    },
    setStrength(next: number) {
      if (disposed) return
      strength = Math.min(1, Math.max(0, next))
      if (!frameHandle) draw(performance.now(), false)
    },
    setQuality(next: QualityTier) {
      if (disposed || quality === next) return
      quality = next
      resize()
      if (next === 'static') {
        stopLoop()
        draw(performance.now(), false)
      }
    },
    start() {
      if (disposed) return
      running = true
      if (quality === 'static') {
        stopLoop()
        draw(performance.now(), false)
        return
      }
      startLoop()
    },
    stop() {
      running = false
      stopLoop()
      // Paused means no live frame rate; report the truth instead of the last value.
      sampler.reset()
      reportMetrics()
    },
    renderOnce() {
      if (disposed) return
      draw(performance.now(), false)
    },
    dispose() {
      if (disposed) return
      disposed = true
      running = false
      stopLoop()
      window.clearTimeout(restoreTimer)
      observer.disconnect()
      stopPixelRatioWatch()
      document.removeEventListener('visibilitychange', handleVisibility)
      canvas.removeEventListener('webglcontextlost', handleContextLost)
      canvas.removeEventListener('webglcontextrestored', handleContextRestored)
      for (const query of pendingQueries) gl.deleteQuery(query)
      pendingQueries.length = 0
      if (activeQuery) gl.deleteQuery(activeQuery)
      activeQuery = null
      disposeGlass()
      releaseScenes()
      gl.getExtension('WEBGL_lose_context')?.loseContext()
      liveInstances = Math.max(0, liveInstances - 1)
    },
  }
}
