/**
 * Web Audio Real-Time Spectrum Visualizer Engine
 * Bulletproof implementation:
 * - Singleton AudioContext & SourceNode to prevent browser errors
 * - Lazy initialization on user gesture
 * - Never throws unhandled exceptions that break UI
 */

let sharedAudioCtx = null
let sharedSourceNode = null
let sharedAnalyser = null
let attachedAudioEl = null

export class AudioVisualizerEngine {
  constructor(canvasElement, audioElement) {
    this.canvas = canvasElement
    this.ctx = canvasElement.getContext('2d')
    this.audio = audioElement

    this.dataArray = null
    this.bufferLength = 0
    this.animationId = null

    this.mode = 'bars' // 'bars' | 'wave' | 'reactor'
    this.peaks = []
    this.peakDecay = 1.2

    this.resize()
    window.addEventListener('resize', () => this.resize())
    this.tryInitAudio()
  }

  tryInitAudio() {
    if (!this.audio) return
    try {
      const AudioContextClass = window.AudioContext || window.webkitAudioContext
      if (!AudioContextClass) return

      if (!sharedAudioCtx) {
        sharedAudioCtx = new AudioContextClass()
      }

      if (!sharedAnalyser) {
        sharedAnalyser = sharedAudioCtx.createAnalyser()
        sharedAnalyser.fftSize = 256
        sharedAnalyser.smoothingTimeConstant = 0.82
      }

      if (attachedAudioEl !== this.audio) {
        try {
          sharedSourceNode = sharedAudioCtx.createMediaElementSource(this.audio)
          sharedSourceNode.connect(sharedAnalyser)
          sharedAnalyser.connect(sharedAudioCtx.destination)
          attachedAudioEl = this.audio
        } catch (err) {
          // Already connected or cross-origin constraint
          console.debug("[VISUALIZER] Media element connection notice:", err)
        }
      }

      if (sharedAnalyser) {
        this.bufferLength = sharedAnalyser.frequencyBinCount
        this.dataArray = new Uint8Array(this.bufferLength)
        this.peaks = new Array(this.bufferLength).fill(0)
      }
    } catch (e) {
      console.warn("[VISUALIZER] Safe catch in tryInitAudio:", e)
    }
  }

  resume() {
    this.tryInitAudio()
    if (sharedAudioCtx && sharedAudioCtx.state === 'suspended') {
      sharedAudioCtx.resume().catch(() => {})
    }
  }

  setMode(mode) {
    this.mode = mode
    if (sharedAnalyser) {
      sharedAnalyser.fftSize = (mode === 'wave') ? 1024 : 256
      this.bufferLength = sharedAnalyser.frequencyBinCount
      this.dataArray = new Uint8Array(this.bufferLength)
      this.peaks = new Array(this.bufferLength).fill(0)
    }
  }

  resize() {
    if (!this.canvas) return
    const rect = this.canvas.parentElement?.getBoundingClientRect() || { width: 800, height: 260 }
    const dpr = window.devicePixelRatio || 1
    this.width = rect.width
    this.height = rect.height
    this.canvas.width = this.width * dpr
    this.canvas.height = this.height * dpr
    this.ctx.scale(dpr, dpr)
  }

  start() {
    if (this.animationId) cancelAnimationFrame(this.animationId)
    const render = () => {
      this.animationId = requestAnimationFrame(render)
      this.draw()
    }
    render()
  }

  stop() {
    if (this.animationId) {
      cancelAnimationFrame(this.animationId)
      this.animationId = null
    }
    if (this.ctx && this.width && this.height) {
      this.ctx.clearRect(0, 0, this.width, this.height)
    }
  }

  draw() {
    if (!this.ctx || !this.width || !this.height) return
    const ctx = this.ctx
    const w = this.width
    const h = this.height

    ctx.clearRect(0, 0, w, h)

    if (!sharedAnalyser || !this.dataArray) {
      this.drawIdleEffect(ctx, w, h)
      return
    }

    try {
      if (this.mode === 'wave') {
        this.drawWave(ctx, w, h)
      } else if (this.mode === 'reactor') {
        this.drawReactor(ctx, w, h)
      } else {
        this.drawBars(ctx, w, h)
      }
    } catch {
      this.drawIdleEffect(ctx, w, h)
    }
  }

  drawBars(ctx, w, h) {
    sharedAnalyser.getByteFrequencyData(this.dataArray)

    const barCount = 48
    const step = Math.floor(this.bufferLength / barCount) || 1
    const barWidth = Math.max(3, (w / barCount) - 3)
    const gap = (w - (barWidth * barCount)) / (barCount - 1)

    const grad = ctx.createLinearGradient(0, h, 0, 0)
    grad.addColorStop(0, 'rgba(0, 242, 254, 0.2)')
    grad.addColorStop(0.5, 'rgba(114, 9, 183, 0.7)')
    grad.addColorStop(1, 'rgba(247, 37, 133, 0.95)')

    for (let i = 0; i < barCount; i++) {
      const val = this.dataArray[i * step] || 0
      const barHeight = Math.max(4, (val / 255) * (h * 0.85))
      const x = i * (barWidth + gap)
      const y = h - barHeight

      ctx.fillStyle = grad
      ctx.beginPath()
      ctx.roundRect(x, y, barWidth, barHeight, [2, 2, 0, 0])
      ctx.fill()

      if (val > (this.peaks[i] || 0)) {
        this.peaks[i] = val
      } else {
        this.peaks[i] = Math.max(0, (this.peaks[i] || 0) - this.peakDecay)
      }
      const peakY = h - Math.max(6, ((this.peaks[i] || 0) / 255) * (h * 0.85)) - 4

      ctx.fillStyle = '#00f2fe'
      ctx.shadowColor = '#00f2fe'
      ctx.shadowBlur = 6
      ctx.fillRect(x, peakY, barWidth, 2)
      ctx.shadowBlur = 0
    }
  }

  drawWave(ctx, w, h) {
    sharedAnalyser.getByteTimeDomainData(this.dataArray)

    ctx.lineWidth = 3
    ctx.strokeStyle = '#00f2fe'
    ctx.shadowColor = '#00f2fe'
    ctx.shadowBlur = 12

    ctx.beginPath()
    const sliceWidth = w / this.bufferLength
    let x = 0

    for (let i = 0; i < this.bufferLength; i++) {
      const v = this.dataArray[i] / 128.0
      const y = (v * h) / 2

      if (i === 0) {
        ctx.moveTo(x, y)
      } else {
        ctx.lineTo(x, y)
      }
      x += sliceWidth
    }

    ctx.lineTo(w, h / 2)
    ctx.stroke()
    ctx.shadowBlur = 0

    ctx.lineWidth = 1.5
    ctx.strokeStyle = 'rgba(247, 37, 133, 0.6)'
    ctx.beginPath()
    x = 0
    for (let i = 0; i < this.bufferLength; i++) {
      const v = this.dataArray[i] / 128.0
      const y = (v * h) / 2 + Math.sin(i * 0.05) * 6
      if (i === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
      x += sliceWidth
    }
    ctx.stroke()
  }

  drawReactor(ctx, w, h) {
    sharedAnalyser.getByteFrequencyData(this.dataArray)

    const centerX = w / 2
    const centerY = h / 2
    const baseRadius = Math.min(w, h) * 0.22
    const barCount = 64
    const angleStep = (Math.PI * 2) / barCount

    let avg = 0
    for (let i = 0; i < 30; i++) avg += this.dataArray[i]
    avg = avg / 30

    ctx.beginPath()
    ctx.arc(centerX, centerY, baseRadius * 0.7 + (avg / 255) * 15, 0, Math.PI * 2)
    ctx.strokeStyle = 'rgba(0, 242, 254, 0.4)'
    ctx.lineWidth = 2
    ctx.shadowColor = '#00f2fe'
    ctx.shadowBlur = 15
    ctx.stroke()
    ctx.shadowBlur = 0

    for (let i = 0; i < barCount; i++) {
      const val = this.dataArray[i * 2] || 0
      const barLen = (val / 255) * (Math.min(w, h) * 0.28)
      const angle = i * angleStep

      const x1 = centerX + Math.cos(angle) * baseRadius
      const y1 = centerY + Math.sin(angle) * baseRadius
      const x2 = centerX + Math.cos(angle) * (baseRadius + barLen + 2)
      const y2 = centerY + Math.sin(angle) * (baseRadius + barLen + 2)

      ctx.strokeStyle = `hsla(${180 + (val / 255) * 140}, 100%, 65%, 0.85)`
      ctx.lineWidth = 3
      ctx.lineCap = 'round'
      ctx.beginPath()
      ctx.moveTo(x1, y1)
      ctx.lineTo(x2, y2)
      ctx.stroke()
    }
  }

  drawIdleEffect(ctx, w, h) {
    ctx.strokeStyle = 'rgba(0, 242, 254, 0.15)'
    ctx.lineWidth = 1
    ctx.beginPath()
    ctx.moveTo(0, h - 2)
    ctx.lineTo(w, h - 2)
    ctx.stroke()
  }
}
