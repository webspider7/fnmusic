import { ref } from 'vue'
import { Buffer } from 'buffer'
import CryptoJS from 'crypto-js'
import { ProxyAPI } from '../api/client'

// Reactive version token so Vue computed properties re-evaluate when sources init
export const runtimeVersion = ref(0)

// Expose Buffer globally so scripts can use Buffer.from directly
if (typeof globalThis.Buffer === 'undefined') {
  globalThis.Buffer = Buffer
}

class LxRuntime {
  constructor() {
    this.requestHandlers = new Map()   // sourceId -> handler function
    this.sourceStatus = new Map()      // sourceId -> { inited, sources }
    this.loadedSources = new Map()     // sourceId -> sourceMeta
  }

  async loadScript(sourceMeta, scriptText) {
    const sourceId = sourceMeta.id
    console.log('[LX-RUNTIME] Loading script:', sourceId)

    const EVENT_NAMES = {
      request: 'request',
      inited: 'inited',
      updateAlert: 'updateAlert'
    }

    const self = this

    const lxEnv = {
      EVENT_NAMES,
      env: 'desktop',
      version: '2.0.0',
      currentScriptInfo: {
        name: sourceMeta.name,
        description: sourceMeta.description || '',
        version: sourceMeta.version || '1.0.0',
        author: sourceMeta.author || '',
        homepage: sourceMeta.homepage || '',
        rawScript: scriptText,
      },

      request(url, options, callback) {
        let opts = {}, cb = callback
        if (typeof options === 'function') {
          cb = options
          opts = {}
        } else if (options) {
          opts = options
        }

        const method = (opts.method || 'GET').toUpperCase()
        const headers = { ...(opts.headers || {}) }
        let bodyData = opts.body || opts.form || opts.formData || opts.json || null
        if (opts.json) {
          headers['Content-Type'] = headers['Content-Type'] || 'application/json; charset=utf-8'
          if (typeof opts.json === 'object' && opts.json !== null) {
            bodyData = JSON.stringify(opts.json)
          }
        }
        const timeout = opts.timeout || 15000

        let aborted = false

        ProxyAPI.request({
          url,
          method,
          headers,
          body: typeof bodyData === 'object' && bodyData !== null ? JSON.stringify(bodyData) : bodyData,
          timeout,
        }).then(res => {
          if (aborted || !cb) return
          if (res.error && (!res.body || res.statusCode >= 500 || !res.statusCode)) {
            cb(new Error(res.error || `HTTP ${res.statusCode}`), null, null)
            return
          }

          let parsedBody = res.body
          if (typeof parsedBody === 'string') {
            try {
              parsedBody = JSON.parse(parsedBody)
            } catch (_) {}
          }

          const resp = {
            statusCode: res.statusCode || 200,
            statusMessage: 'OK',
            headers: res.headers || {},
            bytes: res.body ? res.body.length : 0,
            raw: res.body,
            body: parsedBody,
          }

          // Official signature: callback(err, resp, body)
          cb(null, resp, parsedBody)
        }).catch(err => {
          if (aborted || !cb) return
          console.warn('[LX-RUNTIME] Proxy request failed:', url, err)
          cb(err, null, null)
        })

        // Official signature returns abort function
        return () => {
          aborted = true
        }
      },

      on(eventName, handler) {
        if (eventName === EVENT_NAMES.request) {
          self.requestHandlers.set(sourceId, handler)
          console.log('[LX-RUNTIME] Registered request handler for:', sourceId)
        }
      },

      send(eventName, data) {
        if (eventName === EVENT_NAMES.inited) {
          self.sourceStatus.set(sourceId, { inited: true, sources: data?.sources || {} })
          runtimeVersion.value++
          console.log('[LX-RUNTIME] Source inited:', sourceId, data?.sources ? Object.keys(data.sources) : data)
        } else if (eventName === EVENT_NAMES.updateAlert) {
          console.log('[LX-RUNTIME] updateAlert from:', sourceId, data)
        }
      },

      utils: {
        buffer: {
          from(...args) {
            return Buffer.from(...args)
          },
          bufToString(buf, format = 'utf8') {
            if (typeof buf === 'string') return buf
            if (Buffer.isBuffer(buf)) return buf.toString(format)
            try {
              return Buffer.from(buf).toString(format)
            } catch (_) {
              return String(buf)
            }
          },
        },
        crypto: {
          md5(str) {
            const hex = CryptoJS.MD5(str).toString(CryptoJS.enc.Hex)
            const buf = Buffer.from(hex, 'hex')
            buf.toString = function(enc) {
              if (!enc || enc === 'hex') return hex
              return Buffer.prototype.toString.call(this, enc)
            }
            return buf
          },
          randomBytes(size) {
            const arr = new Uint8Array(size)
            if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
              crypto.getRandomValues(arr)
            } else {
              for (let i = 0; i < size; i++) arr[i] = Math.floor(Math.random() * 256)
            }
            return Buffer.from(arr)
          },
          aesEncrypt(buffer, mode, key, iv) {
            try {
              const keyHex = CryptoJS.enc.Hex.parse(Buffer.from(key).toString('hex'))
              const ivHex = iv ? CryptoJS.enc.Hex.parse(Buffer.from(iv).toString('hex')) : undefined
              const wordArray = CryptoJS.lib.WordArray.create(buffer)
              let encrypted
              if (mode && mode.toLowerCase().includes('ecb')) {
                encrypted = CryptoJS.AES.encrypt(wordArray, keyHex, {
                  mode: CryptoJS.mode.ECB,
                  padding: CryptoJS.pad.Pkcs7
                })
              } else {
                encrypted = CryptoJS.AES.encrypt(wordArray, keyHex, {
                  iv: ivHex,
                  mode: CryptoJS.mode.CBC,
                  padding: CryptoJS.pad.Pkcs7
                })
              }
              const hex = encrypted.ciphertext.toString(CryptoJS.enc.Hex)
              const resBuf = Buffer.from(hex, 'hex')
              resBuf.toString = function(enc) {
                if (!enc || enc === 'hex') return hex
                return Buffer.prototype.toString.call(this, enc)
              }
              return resBuf
            } catch (e) {
              console.warn('[LX-RUNTIME] aesEncrypt fallback:', e)
              return Buffer.from(buffer)
            }
          },
          rsaEncrypt(buffer, key) {
            return Buffer.from(buffer)
          }
        },
        bufToString(buf, format = 'utf8') {
          if (typeof buf === 'string') return buf
          if (Buffer.isBuffer(buf)) return buf.toString(format)
          try {
            return Buffer.from(buf).toString(format)
          } catch (_) {
            return String(buf)
          }
        }
      }
    }

    // Set globally
    globalThis.lx = lxEnv
    this.loadedSources.set(sourceId, sourceMeta)

    try {
      const fn = new Function('lx', scriptText)
      fn(lxEnv)
      console.log('[LX-RUNTIME] Loaded OK:', sourceId)
      return true
    } catch (e) {
      console.error('[LX-RUNTIME] Load error for:', sourceId, e)
      return false
    }
  }

  async getMusicUrl(sourceId, platform, musicInfo, quality = '128k') {
    // 1. Try currently active source
    if (sourceId && this.requestHandlers.has(sourceId)) {
      try {
        const res = await this._callHandler(sourceId, platform, musicInfo, quality)
        if (res?.url) return res
      } catch (e) {
        console.warn(`[LX-RUNTIME] Active source (${quality}) failed:`, e.message)
        if (quality !== '128k') {
          try {
            console.log('[LX-RUNTIME] Active source fallback to 128k...')
            const res128 = await this._callHandler(sourceId, platform, musicInfo, '128k')
            if (res128?.url) return res128
          } catch (e2) {
            console.warn('[LX-RUNTIME] Active source (128k) failed:', e2.message)
          }
        }
      }
    }

    // 2. Auto-fallback to other registered sources
    for (const [otherId] of this.requestHandlers.entries()) {
      if (otherId === sourceId) continue
      try {
        console.log('[LX-RUNTIME] Fallback trying other source:', otherId, musicInfo.name)
        const res = await this._callHandler(otherId, platform, musicInfo, quality)
        if (res?.url) {
          console.log('[LX-RUNTIME] Fallback succeeded with:', otherId, res.url)
          return res
        }
      } catch (e) {
        if (quality !== '128k') {
          try {
            const res128 = await this._callHandler(otherId, platform, musicInfo, '128k')
            if (res128?.url) return res128
          } catch (_) {}
        }
      }
    }

    throw new Error('当前音源及备用音源均未能解析该歌曲，请尝试在「音源管理」切换其他音源')
  }

  async _callHandler(sourceId, platform, musicInfo, quality) {
    const handler = this.requestHandlers.get(sourceId)
    if (!handler) {
      throw new Error('音源尚未就绪')
    }

    // 彻底清洗平台前缀，恢复纯净 ID / songmid / hash
    let cleanId = String(musicInfo.id || musicInfo.songmid || '').trim()
    cleanId = cleanId.replace(/^(wy|tx|kw|kg|mg)_/i, '')

    let cleanSongmid = String(musicInfo.songmid || musicInfo.id || '').trim()
    cleanSongmid = cleanSongmid.replace(/^(wy|tx|kw|kg|mg)_/i, '')

    let cleanHash = ''
    if (platform === 'kg') {
      cleanHash = String(musicInfo.hash || cleanId).trim().replace(/^(wy|tx|kw|kg|mg)_/i, '')
    }

    const cleanMusicInfo = {
      ...musicInfo,
      source: platform,
      id: cleanId,
      songmid: cleanSongmid,
      name: musicInfo.name || musicInfo.title || '',
      singer: musicInfo.singer || musicInfo.artist || '',
      album: musicInfo.album || '',
    }

    if (platform === 'kg') {
      cleanMusicInfo.hash = cleanHash
    } else {
      // 关键：非酷狗平台彻底删除 hash 字段，杜绝第三方脚本误判为酷狗或引发非法 hash 校验失败！
      delete cleanMusicInfo.hash
    }

    const payload = {
      source: platform,
      action: 'musicUrl',
      info: {
        type: quality || '128k',
        musicInfo: cleanMusicInfo
      }
    }

    // Call handler with 15s timeout for reliable third-party resolution
    const result = await Promise.race([
      Promise.resolve().then(() => handler(payload)),
      new Promise((_, reject) => setTimeout(() => reject(new Error('音源响应超时(15s)')), 15000))
    ])

    let finalUrl = ''
    let finalHeaders = {}

    if (typeof result === 'string') {
      finalUrl = result.trim()
    } else if (result && typeof result === 'object') {
      finalUrl = result.url || result.data?.url || ''
      finalHeaders = result.headers || {}
    }

    // Critical Shield: reject panspace fake / notice audio / error mp3
    if (finalUrl && /^https?:/.test(finalUrl)) {
      const lower = finalUrl.toLowerCase()
      if (lower.includes('panspace') || lower.includes('notice') || lower.includes('audio_forbidden') || lower.includes('error.mp3')) {
        throw new Error('第三方音源返回了渠道限制提示录音(panspace)，已自动屏蔽')
      }
      return { url: finalUrl, headers: finalHeaders }
    }

    throw new Error('返回的播放地址无效')
  }

  async getLyric(sourceId, platform, musicInfo) {
    const handler = this.requestHandlers.get(sourceId)
    if (!handler) return null
    try {
      let cleanId = String(musicInfo.id || musicInfo.songmid || '').trim().replace(/^(wy|tx|kw|kg|mg)_/i, '')
      let cleanSongmid = String(musicInfo.songmid || musicInfo.id || '').trim().replace(/^(wy|tx|kw|kg|mg)_/i, '')
      const payload = {
        source: platform,
        action: 'lyric',
        info: {
          musicInfo: {
            source: platform,
            id: cleanId,
            songmid: cleanSongmid,
            name: musicInfo.name || musicInfo.title || '',
            singer: musicInfo.singer || musicInfo.artist || '',
            album: musicInfo.album || '',
          }
        }
      }
      const res = await Promise.race([
        Promise.resolve().then(() => handler(payload)),
        new Promise((_, reject) => setTimeout(() => reject(new Error('timeout')), 5000))
      ])
      if (res && (res.lyric || res.lrc)) {
        return { lyric: res.lyric || res.lrc, tlyric: res.tlyric || '' }
      }
    } catch (_) {}
    return null
  }

  getSupportedSources(sourceId) {
    // Touch reactive ref so Vue computed properties re-evaluate when runtime changes!
    const _ = runtimeVersion.value
    if (sourceId && this.sourceStatus.has(sourceId)) {
      const status = this.sourceStatus.get(sourceId)
      if (status && status.sources && Object.keys(status.sources).length > 0) {
        return Object.keys(status.sources).filter(s => s !== 'mg' && s !== 'local')
      }
    }
    for (const s of this.sourceStatus.values()) {
      if (s.sources && Object.keys(s.sources).length > 0) {
        return Object.keys(s.sources).filter(k => k !== 'mg' && k !== 'local')
      }
    }
    return []
  }
}

export const lxRuntime = new LxRuntime()
