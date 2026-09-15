import { ref, computed } from 'vue'
import { DownloadAPI, NasAPI, SearchAPI } from '../api/client'
import { lxRuntime } from '../engine/lx-runtime'

// 任务数据结构：
// {
//   id: string,
//   songKey: string,
//   song: Object,
//   status: 'pending' | 'resolving' | 'downloading' | 'success' | 'failed',
//   error: string,
//   activeSourceId: string,
//   platform: string,
//   addedAt: number,
//   completedAt: number | null
// }

const tasks = ref([])
const downloadedMap = ref({})
const isOpen = ref(false)
const currentDownloadDir = ref('/vol1/Music')

// 目录选择状态
const showDirModal = ref(false)
const editDownloadDir = ref('/vol1/Music')
const isSavingDir = ref(false)
const browseFolders = ref([])
const browseParentPath = ref('')

const MAX_CONCURRENCY = 2
let runningCount = 0
const isPaused = ref(false)

// 计算属性
const activeCount = computed(() => 
  tasks.value.filter(t => t.status === 'pending' || t.status === 'resolving' || t.status === 'downloading').length
)

const pausedCount = computed(() => 
  tasks.value.filter(t => t.status === 'paused').length
)

const completedCount = computed(() => 
  tasks.value.filter(t => t.status === 'success').length
)

const failedCount = computed(() => 
  tasks.value.filter(t => t.status === 'failed').length
)

const totalCount = computed(() => tasks.value.length)

// 获取唯一标识
function getSongKey(song) {
  return song.id || `${song.singer || '未知'} - ${song.name || '未知'}`
}

// 检查是否在下载中
function isSongDownloading(song) {
  const key = getSongKey(song)
  return tasks.value.some(t => 
    t.songKey === key && (t.status === 'pending' || t.status === 'resolving' || t.status === 'downloading')
  )
}

// 检查是否已下载
function isSongDownloaded(song) {
  const key1 = `${song.singer} - ${song.name}`
  const key2 = song.id
  return !!(downloadedMap.value[key1] || (key2 && downloadedMap.value[key2]))
}

// 初始化加载配置
async function loadConfig() {
  try {
    const res = await DownloadAPI.getConfig()
    if (res.code === 200) {
      currentDownloadDir.value = res.download_dir || res.default_nas_dir || '/vol1/Music'
      editDownloadDir.value = currentDownloadDir.value
    }
  } catch (e) {
    console.warn('Failed to load download config:', e)
  }
}

// 批量查询已下载状态
async function checkSongsDownloaded(songs) {
  if (!songs || !songs.length) return
  try {
    const payload = songs.map(s => ({
      id: s.id || '',
      name: s.name || '',
      singer: s.singer || '',
    }))
    const res = await DownloadAPI.checkDownloaded(payload)
    if (res.code === 200 && res.exists) {
      downloadedMap.value = { ...downloadedMap.value, ...res.exists }
    }
  } catch (e) {
    console.warn('Failed to check download status:', e)
  }
}

// 添加单曲下载任务
function addSong(song, activeSourceId, autoOpenOrOptions = true, qualityParam = null) {
  let autoOpen = true
  let quality = qualityParam || song.downloadQuality || song.quality || '320k'

  if (typeof autoOpenOrOptions === 'object' && autoOpenOrOptions !== null) {
    autoOpen = autoOpenOrOptions.autoOpen ?? true
    quality = autoOpenOrOptions.quality || quality
  } else if (typeof autoOpenOrOptions === 'boolean') {
    autoOpen = autoOpenOrOptions
  }

  const key = getSongKey(song)
  // 如果当前已在下载队列中且未结束，则直接打开面板
  const existing = tasks.value.find(t => 
    t.songKey === key && (t.status === 'pending' || t.status === 'resolving' || t.status === 'downloading')
  )
  if (existing) {
    if (autoOpen) isOpen.value = true
    return existing
  }

  const task = {
    id: `task_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
    songKey: key,
    song: { ...song },
    quality: quality,
    status: 'pending',
    error: '',
    activeSourceId: activeSourceId || '',
    platform: song.source || 'kw',
    addedAt: Date.now(),
    completedAt: null
  }

  tasks.value.unshift(task)
  if (autoOpen) {
    isOpen.value = true
  }

  processQueue()
  return task
}

// 批量添加下载任务 (默认 autoOpen 为 false，静默后台加入队列)
function addBatch(songs, activeSourceId, autoOpen = false, targetQuality = null) {
  if (!songs || !songs.length) return

  for (const song of songs) {
    const key = getSongKey(song)
    // 已经下载过或已经在队列中的跳过
    if (isSongDownloaded(song)) continue
    if (isSongDownloading(song)) continue

    const q = targetQuality || song.downloadQuality || song.quality || '320k'

    const task = {
      id: `task_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
      songKey: key,
      song: { ...song },
      quality: q,
      status: isPaused.value ? 'paused' : 'pending',
      error: '',
      activeSourceId: activeSourceId || '',
      platform: song.source || 'kw',
      addedAt: Date.now(),
      completedAt: null,
      _aborted: false
    }
    tasks.value.push(task)
  }

  if (autoOpen) {
    isOpen.value = true
  }

  if (!isPaused.value) {
    processQueue()
  }
}

// 单个任务暂停
function pauseTask(taskId) {
  const task = tasks.value.find(t => t.id === taskId)
  if (task) {
    task.status = 'paused'
    task._aborted = true
  }
}

// 单个任务继续
function resumeTask(taskId, fallbackSourceId) {
  const task = tasks.value.find(t => t.id === taskId)
  if (task) {
    task.status = 'pending'
    task.error = ''
    task._aborted = false
    if (fallbackSourceId) {
      task.activeSourceId = fallbackSourceId
    }
    isPaused.value = false
    processQueue()
  }
}

// 单个任务取消/停止
function cancelTask(taskId) {
  const task = tasks.value.find(t => t.id === taskId)
  if (task) {
    task._aborted = true
  }
  tasks.value = tasks.value.filter(t => t.id !== taskId)
}

// 全局暂停所有下载
function pauseAll() {
  isPaused.value = true
  tasks.value.forEach(t => {
    if (t.status === 'pending' || t.status === 'resolving' || t.status === 'downloading') {
      t.status = 'paused'
      t._aborted = true
    }
  })
}

// 全局继续所有下载
function resumeAll(fallbackSourceId) {
  isPaused.value = false
  tasks.value.forEach(t => {
    if (t.status === 'paused') {
      t.status = 'pending'
      t.error = ''
      t._aborted = false
      if (fallbackSourceId) {
        t.activeSourceId = fallbackSourceId
      }
    }
  })
  processQueue()
}

// 停止并清空所有进行中与等待中任务
function stopAll() {
  tasks.value.forEach(t => {
    if (t.status !== 'success') {
      t._aborted = true
    }
  })
  tasks.value = tasks.value.filter(t => t.status === 'success')
}

// 重试失败任务
function retryTask(taskId, fallbackSourceId) {
  const task = tasks.value.find(t => t.id === taskId)
  if (task) {
    task.status = 'pending'
    task.error = ''
    task._aborted = false
    if (fallbackSourceId) {
      task.activeSourceId = fallbackSourceId
    }
    isPaused.value = false
    processQueue()
  }
}

// 全部重试
function retryAllFailed(fallbackSourceId) {
  isPaused.value = false
  tasks.value.forEach(t => {
    if (t.status === 'failed') {
      t.status = 'pending'
      t.error = ''
      t._aborted = false
      if (fallbackSourceId) {
        t.activeSourceId = fallbackSourceId
      }
    }
  })
  processQueue()
}

// 移除任务
function removeTask(taskId) {
  cancelTask(taskId)
}

// 清空已完成任务
function clearCompleted() {
  tasks.value = tasks.value.filter(t => t.status !== 'success')
}

// 清空全部非进行中任务
function clearAll() {
  tasks.value = tasks.value.filter(t => t.status === 'resolving' || t.status === 'downloading')
}

// 核心队列调度器（页面切换不会打断）
async function processQueue() {
  if (isPaused.value) return
  if (runningCount >= MAX_CONCURRENCY) return

  // 查找下一个等待中的任务
  const nextTask = tasks.value.find(t => t.status === 'pending')
  if (!nextTask) return

  runningCount++
  executeTask(nextTask).finally(() => {
    runningCount--
    // 继续消费后续队列
    if (!isPaused.value) {
      processQueue()
    }
  })

  // 如果并发还有空余，继续拉起
  if (!isPaused.value && runningCount < MAX_CONCURRENCY) {
    processQueue()
  }
}

// 单个任务执行流程
async function executeTask(task) {
  if (task._aborted || task.status === 'paused') return

  const song = task.song
  const payload = { ...song }
  const targetQuality = task.quality || '320k'
  payload.quality = targetQuality

  // 1. 直链解析阶段
  task.status = 'resolving'
  task.error = ''
  task.qualityNote = ''

  let audioUrl = payload.url || payload.streamUrl || ''

  if (!audioUrl && payload.source !== 'nas') {
    const sourceId = task.activeSourceId || localStorage.getItem('fn_active_source_id') || ''
    if (!sourceId) {
      task.status = 'failed'
      task.error = '未导入第三方音源脚本，无法获取下载直链'
      return
    }

    try {
      const platform = task.platform || song.source || 'kw'
      const urlRes = await lxRuntime.getMusicUrl(sourceId, platform, song, targetQuality)
      if (task._aborted || task.status === 'paused') return

      if (urlRes?.url) {
        audioUrl = urlRes.url
        payload.url = audioUrl
        if (urlRes.headers?.Referer) {
          payload.referer = urlRes.headers.Referer
        }

        // 若用户指定无损(FLAC/Hi-Res)，但原平台音源仅能返回 MP3 流时，尝试跨平台寻找真正的 FLAC 流
        const isLosslessReq = targetQuality === 'flac' || targetQuality === 'flac24bit'
        const isDowngraded = isLosslessReq && (audioUrl.includes('.mp3') || audioUrl.includes('type=mp3'))
        if (isDowngraded) {
          console.log(`[DOWNLOAD] 原平台 [${platform}] 降级为 MP3，尝试跨平台检索真实 FLAC:`, song.name)
          const altPlatforms = ['kw', 'kg', 'tx'].filter(p => p !== platform)
          for (const altP of altPlatforms) {
            try {
              const query = `${song.name} ${song.singer || ''}`.trim()
              const searchRes = await SearchAPI.search(query, altP, 1)
              const candidates = searchRes?.data?.list || searchRes?.data?.songs || []
              const matched = candidates.find(c => 
                (c.name.includes(song.name) || song.name.includes(c.name)) &&
                (!song.singer || c.singer.includes(song.singer) || song.singer.includes(c.singer))
              )
              if (matched) {
                const altRes = await lxRuntime.getMusicUrl(sourceId, altP, matched, targetQuality)
                const altUrl = altRes?.url || ''
                // 严格校验：备选平台返回的必须是真实的无损直链，严禁采用未经验证的脚本模板或MP3流
                const isRealFlac = altUrl && (
                  altUrl.includes('.flac') ||
                  altUrl.includes('format=flac') ||
                  altUrl.includes('rate=flac') ||
                  altUrl.includes('fLaC')
                ) && !altUrl.includes('.php') && !altUrl.includes('.mp3') && !altUrl.includes('type=mp3')

                if (isRealFlac) {
                  audioUrl = altUrl
                  payload.url = audioUrl
                  if (altRes.headers?.Referer) payload.referer = altRes.headers.Referer
                  console.log(`[DOWNLOAD] 跨平台 [${altP.toUpperCase()}] 成功获取真实无损 FLAC 直链:`, audioUrl)
                  break
                }
              }
            } catch (_) {}
          }
        }
      } else {
        throw new Error('音源返回空下载直链')
      }
    } catch (resolveErr) {
      if (task._aborted || task.status === 'paused') return

      // 如果是无损格式请求失败，先同平台降级到 320k
      const isLosslessReq2 = targetQuality === 'flac' || targetQuality === 'flac24bit'
      if (isLosslessReq2) {
        try {
          const platform2 = task.platform || song.source || 'kw'
          const sourceId2 = task.activeSourceId || localStorage.getItem('fn_active_source_id') || ''
          console.log(`[DOWNLOAD] FLAC 解析失败，同平台 [${platform2}] 降级到 320k:`, resolveErr.message)
          const res320 = await lxRuntime.getMusicUrl(sourceId2, platform2, song, '320k')
          if (res320?.url && !res320.url.includes('.php')) {
            audioUrl = res320.url
            payload.url = audioUrl
            if (res320.headers?.Referer) payload.referer = res320.headers.Referer
            task.qualityNote = '音源仅提供最高MP3流 (320k)'
          }
        } catch (_) {}
      }

      // 仍然没有 URL：尝试备选跨平台检索
      if (!audioUrl) {
        try {
          const query = `${song.singer} ${song.name}`.trim()
          const searchRes = await SearchAPI.search(query, 'kw', 1)
          const candidates = searchRes?.data?.list || searchRes?.data?.songs || []
          const matched = candidates.find(c => 
            (c.name.includes(song.name) || song.name.includes(c.name)) &&
            (c.singer.includes(song.singer) || song.singer.includes(c.singer))
          )
          if (matched) {
            const fallbackRes = await lxRuntime.getMusicUrl(sourceId, 'kw', matched, targetQuality)
            if (fallbackRes?.url && !fallbackRes.url.includes('.php')) {
              audioUrl = fallbackRes.url
              payload.url = audioUrl
              if (fallbackRes.headers?.Referer) payload.referer = fallbackRes.headers.Referer
            }
          }
        } catch (_) {}
      }

      if (!audioUrl) {
        if (task._aborted || task.status === 'paused') return
        task.status = 'failed'
        task.error = `音源解析直链失败: ${resolveErr.message || '未能获取到有效流'}`
        return
      }
    }
  }

  if (task._aborted || task.status === 'paused') return

  // 2. 发起 NAS 存储下载阶段
  task.status = 'downloading'

  try {
    const res = await DownloadAPI.downloadSong(payload)
    if (task._aborted || task.status === 'paused') return

    if (res.code === 200) {
      task.status = 'success'
      task.completedAt = Date.now()
      const savedPath = res.data?.path || res.result?.path || ''
      if (savedPath) {
        task.savedPath = savedPath
        const ext = savedPath.split('.').pop()
        if (ext) {
          task.actualFormat = ext.toUpperCase()
          if ((targetQuality === 'flac' || targetQuality === 'flac24bit') && task.actualFormat === 'MP3') {
            task.qualityNote = '音源仅提供最高MP3流'
          }
        }
      }
      // 登记已下载映射
      downloadedMap.value[task.songKey] = true
      downloadedMap.value[`${song.singer} - ${song.name}`] = true
      if (song.id) {
        downloadedMap.value[song.id] = true
      }
    } else {
      task.status = 'failed'
      task.error = res.message || '后端下载失败'
    }
  } catch (dlErr) {
    if (task._aborted || task.status === 'paused') return
    task.status = 'failed'
    task.error = dlErr.message || '网络连接异常'
  }
}

// 目录浏览与保存
async function openDirModal() {
  editDownloadDir.value = currentDownloadDir.value
  showDirModal.value = true
  browseDir(currentDownloadDir.value)
}

async function browseDir(path) {
  try {
    const res = await NasAPI.browse(path)
    if (res.code === 200 && res.data) {
      browseFolders.value = res.data.folders || []
      browseParentPath.value = res.data.parent || ''
      if (res.data.current) editDownloadDir.value = res.data.current
    }
  } catch (e) {
    console.warn('Failed to browse dir:', e)
  }
}

async function saveDownloadDir() {
  const dir = editDownloadDir.value.trim()
  if (!dir) return
  isSavingDir.value = true
  try {
    const res = await DownloadAPI.saveConfig(dir)
    if (res.code === 200) {
      currentDownloadDir.value = dir
      showDirModal.value = false
    } else {
      alert('保存目录失败: ' + res.message)
    }
  } catch (e) {
    alert('保存目录失败: ' + e.message)
  } finally {
    isSavingDir.value = false
  }
}

export const downloadManager = {
  tasks,
  downloadedMap,
  isOpen,
  isPaused,
  currentDownloadDir,
  showDirModal,
  editDownloadDir,
  isSavingDir,
  browseFolders,
  browseParentPath,

  activeCount,
  pausedCount,
  completedCount,
  failedCount,
  totalCount,

  open: () => { isOpen.value = true },
  close: () => { isOpen.value = false },
  toggle: () => { isOpen.value = !isOpen.value },

  addSong,
  addBatch,
  pauseTask,
  resumeTask,
  cancelTask,
  pauseAll,
  resumeAll,
  stopAll,
  retryTask,
  retryAllFailed,
  removeTask,
  clearCompleted,
  clearAll,

  isSongDownloading,
  isSongDownloaded,
  checkSongsDownloaded,
  loadConfig,

  openDirModal,
  browseDir,
  saveDownloadDir,
}
