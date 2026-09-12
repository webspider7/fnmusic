<template>
  <div class="h-full flex flex-col overflow-hidden bg-gray-50 select-none">
    <!-- Header -->
    <div class="px-4 md:px-6 py-4 bg-white border-b border-gray-100 flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
      <div>
        <h2 class="text-base font-bold text-gray-800 flex items-center gap-2">
          飞牛 NAS 本地曲库
          <span class="text-[10px] px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-600 border border-emerald-200 font-mono">
            原生存储卷支持
          </span>
        </h2>
        <p class="text-xs text-gray-400 mt-0.5">
          深度适配 fnOS 存储卷目录，自动提取嵌入式高清 ID3 专辑封面，支持 FLAC / APE / WAV / DSD / MP3 原盘极速串流。
        </p>
      </div>

      <div class="flex items-center gap-2 self-end sm:self-auto">
        <button
          @click="playAllNasSongs"
          :disabled="!songs.length"
          class="px-3.5 py-1.5 rounded-lg bg-emerald-50 hover:bg-emerald-100 text-emerald-700 font-semibold text-xs border border-emerald-200/80 shadow-xs flex items-center gap-1.5 transition-all disabled:opacity-50"
        >
          <Play class="w-3.5 h-3.5 fill-current text-emerald-600" />
          <span>播放全部</span>
        </button>

        <button
          @click="openFolderModal"
          class="px-3.5 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white font-semibold text-xs shadow-sm flex items-center gap-1.5 transition-all"
        >
          <FolderOpen class="w-3.5 h-3.5" />
          <span>选择曲库目录</span>
        </button>

        <button 
          @click="scanFolder(currentDir, true)"
          :disabled="isScanning"
          class="px-3.5 py-1.5 rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600 text-xs font-medium transition-all flex items-center gap-1.5"
        >
          <RotateCw :class="['w-3.5 h-3.5 text-emerald-500', isScanning ? 'animate-spin' : '']" />
          <span>{{ isScanning ? '扫描中...' : '重新扫描' }}</span>
        </button>
      </div>
    </div>

    <!-- Current Directory & Quick Switch Bar -->
    <div class="px-4 md:px-6 py-3 bg-white border-b border-gray-100 flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 text-xs">
      <!-- Left: Current Directory display & Breadcrumb -->
      <div class="flex items-center gap-2 overflow-hidden">
        <div class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-emerald-50 text-emerald-700 font-mono border border-emerald-200/60 shrink-0">
          <Folder class="w-3.5 h-3.5 text-emerald-600" />
          <span class="font-bold">当前目录:</span>
        </div>

        <div class="flex items-center gap-1 overflow-x-auto text-gray-600 font-mono py-0.5">
          <button
            v-for="(crumb, idx) in breadcrumbs"
            :key="idx"
            @click="switchDir(crumb.path)"
            class="px-1.5 py-0.5 rounded hover:bg-gray-100 hover:text-emerald-600 transition-colors flex items-center gap-1"
          >
            <span>{{ crumb.name }}</span>
            <span v-if="idx < breadcrumbs.length - 1" class="text-gray-300">/</span>
          </button>
        </div>

        <span class="text-[11px] text-gray-400 font-normal shrink-0">
          (已加载 {{ songs.length }} 首歌曲)
        </span>
      </div>

      <!-- Right: Quick Volumes / Preset Buttons -->
      <div class="flex items-center gap-1.5 overflow-x-auto">
        <span class="text-gray-400 text-xs shrink-0">快速切换:</span>
        <button
          v-for="d in quickDirs"
          :key="d"
          @click="switchDir(d)"
          :class="[
            'px-2.5 py-1 rounded-lg font-mono text-[11px] transition-all whitespace-nowrap',
            currentDir === d
              ? 'bg-emerald-500 text-white font-semibold shadow-xs'
              : 'bg-gray-100 hover:bg-gray-200 text-gray-600'
          ]"
        >
          {{ d }}
        </button>
        <button
          @click="openFolderModal"
          class="px-2 py-1 rounded-lg bg-emerald-50 hover:bg-emerald-100 text-emerald-600 text-[11px] font-medium transition-colors whitespace-nowrap"
        >
          + 更多目录
        </button>
      </div>
    </div>

    <!-- Song Table List -->
    <div class="flex-1 overflow-y-auto p-3 sm:p-5">
      <!-- Loading State -->
      <div v-if="isScanning" class="h-72 flex flex-col items-center justify-center text-gray-400">
        <RotateCw class="w-8 h-8 text-emerald-500 animate-spin mb-3" />
        <p class="text-sm font-medium text-gray-600">正在深度遍历 NAS 音乐文件与提取 ID3 封面...</p>
        <p class="text-xs text-gray-400 mt-1">目录：{{ currentDir }}</p>
      </div>

      <!-- Empty State -->
      <div v-else-if="!songs.length" class="h-72 flex flex-col items-center justify-center text-gray-400 bg-white rounded-2xl border border-gray-100 shadow-xs p-8">
        <div class="w-14 h-14 rounded-2xl bg-emerald-50 flex items-center justify-center mb-3">
          <FolderOpen class="w-7 h-7 text-emerald-500" />
        </div>
        <p class="text-base font-semibold text-gray-700">当前目录未找到音频文件</p>
        <p class="text-xs text-gray-400 mt-1 mb-4">当前路径：{{ currentDir }}</p>
        <button
          @click="openFolderModal"
          class="px-4 py-2 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-semibold shadow-sm transition-all flex items-center gap-1.5"
        >
          <FolderOpen class="w-3.5 h-3.5" />
          <span>点击选择包含音乐的文件夹</span>
        </button>
      </div>

      <!-- Table -->
      <div v-else class="bg-white rounded-2xl border border-gray-100 shadow-xs overflow-hidden">
        <table class="w-full text-left text-xs text-gray-600 border-collapse">
          <thead class="bg-gray-50/80 border-b border-gray-100 text-gray-400 font-medium">
            <tr>
              <th class="py-3 px-2 sm:px-3 w-10 sm:w-12 text-center">#</th>
              <th class="py-3 px-3 sm:px-4">歌曲标题 / 封面</th>
              <th class="py-3 px-3 sm:px-4">艺术家</th>
              <th class="hidden md:table-cell py-3 px-4">专辑</th>
              <th class="hidden sm:table-cell py-3 px-3 text-center">格式</th>
              <th class="hidden lg:table-cell py-3 px-3 text-right">文件大小</th>
              <th class="py-3 px-3 sm:px-4 text-center w-16 sm:w-20">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100">
            <tr 
              v-for="(song, idx) in songs" 
              :key="song.path"
              @dblclick="playSong(song, idx)"
              class="hover:bg-emerald-50/40 transition-colors group cursor-pointer"
            >
              <td class="py-3 px-2 sm:px-3 text-center text-gray-400 font-mono">{{ idx + 1 }}</td>
              <td class="py-3 px-3 sm:px-4 flex items-center gap-2.5 sm:gap-3 min-w-0">
                <div class="w-8 h-8 sm:w-9 sm:h-9 rounded-lg overflow-hidden bg-gray-100 border border-gray-200/60 shrink-0 flex items-center justify-center">
                  <img 
                    v-if="song.cover_url" 
                    :src="song.cover_url" 
                    alt="cover" 
                    referrerpolicy="no-referrer"
                    class="w-full h-full object-cover"
                    @error="song.cover_url = ''"
                  />
                  <span v-else class="text-sm sm:text-base">💿</span>
                </div>
                <div class="overflow-hidden min-w-0">
                  <div class="flex items-center gap-1.5">
                    <p class="font-semibold text-gray-800 truncate max-w-[130px] sm:max-w-xs group-hover:text-emerald-600 transition-colors">
                      {{ song.title || song.filename }}
                    </p>
                    <span v-if="song.has_lyric" class="text-[9px] px-1 py-0.2 rounded bg-emerald-100 text-emerald-700 font-bold shrink-0" title="含本地歌词">词</span>
                  </div>
                  <p class="text-[11px] text-gray-400 truncate max-w-[130px] sm:max-w-xs">{{ song.filename }}</p>
                </div>
              </td>
              <td class="py-3 px-3 sm:px-4 text-gray-600 font-medium truncate max-w-[90px] sm:max-w-[150px]">{{ song.artist || '本地歌手' }}</td>
              <td class="hidden md:table-cell py-3 px-4 text-gray-400 truncate max-w-[150px]">{{ song.album || 'NAS 音乐' }}</td>
              <td class="hidden sm:table-cell py-3 px-3 text-center">
                <span class="text-[10px] uppercase px-1.5 py-0.5 rounded font-mono font-bold bg-emerald-50 text-emerald-700 border border-emerald-200">
                  {{ song.format }}
                </span>
              </td>
              <td class="hidden lg:table-cell py-3 px-3 text-right text-gray-400 font-mono">{{ formatSize(song.size) }}</td>
              <td class="py-3 px-3 sm:px-4 text-center">
                <button 
                  @click.stop="playSong(song, idx)"
                  class="p-1.5 sm:p-2 rounded-lg bg-emerald-50 hover:bg-emerald-500 hover:text-white text-emerald-600 transition-all shadow-xs"
                  title="立即播放"
                >
                  <Play class="w-3.5 h-3.5 fill-current" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- ─── Interactive Folder Selector Modal ─── -->
    <div
      v-if="showFolderModal"
      class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/40 backdrop-blur-xs"
    >
      <div class="bg-white rounded-2xl max-w-xl w-full p-6 shadow-2xl border border-gray-100 flex flex-col max-h-[82vh]">
        <!-- Modal Top Bar -->
        <div class="flex items-center justify-between pb-4 border-b border-gray-100 shrink-0">
          <div>
            <h3 class="text-base font-bold text-gray-800 flex items-center gap-2">
              <FolderOpen class="w-4 h-4 text-emerald-500" />
              <span>选择 NAS 曲库目录</span>
            </h3>
            <p class="text-xs text-gray-400 mt-0.5">点击进入子文件夹，或点击右侧按钮直接选用</p>
          </div>
          <button @click="showFolderModal = false" class="p-1 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600">
            <X class="w-5 h-5" />
          </button>
        </div>

        <!-- Volume Switcher Tabs -->
        <div class="py-3 border-b border-gray-100 flex items-center gap-2 overflow-x-auto shrink-0 text-xs">
          <span class="text-gray-400 shrink-0 font-medium">存储卷:</span>
          <button
            v-for="vol in modalVolumes"
            :key="vol"
            @click="modalDrillDown(vol)"
            :class="[
              'px-3 py-1 rounded-lg font-mono transition-all flex items-center gap-1',
              modalCurrentPath.startsWith(vol)
                ? 'bg-emerald-500 text-white font-semibold shadow-xs'
                : 'bg-gray-100 hover:bg-gray-200 text-gray-600'
            ]"
          >
            <HardDrive class="w-3 h-3" />
            <span>{{ vol }}</span>
          </button>
        </div>

        <!-- Path Breadcrumb Bar & Back to Parent Button -->
        <div class="py-2.5 px-3 bg-gray-50 rounded-xl my-3 flex items-center justify-between gap-2 shrink-0 text-xs">
          <div class="flex items-center gap-1 overflow-x-auto font-mono text-gray-600">
            <span class="text-gray-400">路径:</span>
            <button
              v-for="(crumb, idx) in modalBreadcrumbs"
              :key="idx"
              @click="modalDrillDown(crumb.path)"
              class="px-1 py-0.5 rounded hover:bg-white hover:text-emerald-600 transition-colors"
            >
              {{ crumb.name }}
              <span v-if="idx < modalBreadcrumbs.length - 1" class="text-gray-300 ml-1">/</span>
            </button>
          </div>

          <button
            v-if="modalParentPath"
            @click="modalDrillDown(modalParentPath)"
            class="px-2.5 py-1 rounded-lg bg-white border border-gray-200 hover:bg-gray-100 text-gray-600 text-[11px] font-medium flex items-center gap-1 shrink-0 transition-colors"
            title="返回上一级"
          >
            <CornerLeftUp class="w-3 h-3 text-gray-500" />
            <span>上一级</span>
          </button>
        </div>

        <!-- Folder List Area -->
        <div class="flex-1 overflow-y-auto divide-y divide-gray-50 pr-1 min-h-[220px]">
          <!-- Loading -->
          <div v-if="modalLoading" class="h-44 flex flex-col items-center justify-center text-gray-400">
            <RotateCw class="w-6 h-6 text-emerald-500 animate-spin mb-2" />
            <span class="text-xs">正在读取目录内容...</span>
          </div>

          <!-- Empty subfolders -->
          <div v-else-if="!modalFolders.length" class="h-44 flex flex-col items-center justify-center text-gray-400">
            <Folder class="w-8 h-8 text-gray-300 mb-2" />
            <p class="text-xs text-gray-500">当前目录下无子文件夹</p>
            <p v-if="modalAudioCount > 0" class="text-xs text-emerald-600 mt-1 font-medium">
              ★ 发现当前目录包含 {{ modalAudioCount }} 首音乐文件！
            </p>
          </div>

          <!-- Folder rows -->
          <div
            v-for="f in modalFolders"
            :key="f.path"
            @click="modalDrillDown(f.path)"
            class="py-2.5 px-3 flex items-center justify-between hover:bg-emerald-50/50 rounded-xl cursor-pointer transition-colors group"
          >
            <div class="flex items-center gap-2.5 overflow-hidden">
              <Folder class="w-4 h-4 text-emerald-500 shrink-0 group-hover:scale-110 transition-transform" />
              <span class="text-xs font-semibold text-gray-700 truncate group-hover:text-emerald-700 transition-colors">
                {{ f.name }}
              </span>
              <span
                v-if="f.audio_count > 0"
                class="text-[10px] px-1.5 py-0.5 rounded-full bg-emerald-50 text-emerald-600 border border-emerald-200 font-mono shrink-0"
              >
                🎵 {{ f.audio_count }} 首歌曲
              </span>
            </div>

            <div class="flex items-center gap-1 shrink-0" @click.stop>
              <button
                @click="confirmSelection(f.path)"
                class="px-2.5 py-1 rounded-lg bg-emerald-50 hover:bg-emerald-500 hover:text-white text-emerald-600 text-[11px] font-medium transition-all"
              >
                选择此目录
              </button>
              <button
                @click="modalDrillDown(f.path)"
                class="p-1 rounded-lg text-gray-300 hover:text-gray-600 transition-colors"
                title="进入子目录"
              >
                <ChevronRight class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        <!-- Modal Footer -->
        <div class="pt-4 mt-2 border-t border-gray-100 flex items-center justify-between shrink-0">
          <div class="text-xs text-gray-500 truncate max-w-xs">
            当前选中：<span class="font-mono font-semibold text-emerald-700">{{ modalCurrentPath }}</span>
          </div>
          <div class="flex items-center gap-2">
            <button
              @click="showFolderModal = false"
              class="px-4 py-2 rounded-xl text-xs text-gray-500 hover:bg-gray-100 transition-colors"
            >
              取消
            </button>
            <button
              @click="confirmSelection(modalCurrentPath)"
              class="px-5 py-2 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white font-semibold text-xs shadow-sm flex items-center gap-1.5 transition-all"
            >
              <Check class="w-3.5 h-3.5" />
              <span>确认选择此目录并扫描</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { NasAPI } from '../api/client'
import {
  RotateCw, Folder, FolderOpen, Play, X, Check,
  HardDrive, CornerLeftUp, ChevronRight
} from 'lucide-vue-next'

const emit = defineEmits(['play'])

const quickDirs = ref(['/vol1/Music', '/vol1/music', '/vol2/Music', '/vol1'])
const currentDir = ref(localStorage.getItem('fn_nas_dir') || '/vol1/Music')
const songs = ref([])
const isScanning = ref(false)

// Folder Selector Modal state
const showFolderModal = ref(false)
const modalCurrentPath = ref('/vol1')
const modalParentPath = ref('')
const modalVolumes = ref(['/vol1', '/vol2'])
const modalFolders = ref([])
const modalLoading = ref(false)
const modalAudioCount = ref(0)

const breadcrumbs = computed(() => {
  const p = currentDir.value || '/vol1/Music'
  const parts = p.split('/').filter(Boolean)
  const list = []
  let acc = ''
  for (const part of parts) {
    acc += '/' + part
    list.push({ name: part, path: acc })
  }
  return list.length ? list : [{ name: p, path: p }]
})

const modalBreadcrumbs = computed(() => {
  const p = modalCurrentPath.value || '/vol1'
  const parts = p.split('/').filter(Boolean)
  const list = []
  let acc = ''
  for (const part of parts) {
    acc += '/' + part
    list.push({ name: part, path: acc })
  }
  return list.length ? list : [{ name: p, path: p }]
})

function formatSize(bytes) {
  if (!bytes) return '0 B'
  const mb = bytes / (1024 * 1024)
  if (mb > 1024) return (mb / 1024).toFixed(2) + ' GB'
  return mb.toFixed(1) + ' MB'
}

async function loadDirectories() {
  try {
    const res = await NasAPI.directories()
    if (res.code === 200 && res.data?.length) {
      quickDirs.value = res.data
      if (!localStorage.getItem('fn_nas_dir')) {
        currentDir.value = res.data[0]
      }
    }
  } catch (err) {
    console.warn("Load dirs failed:", err)
  }
}

async function scanFolder(dir, refresh = false) {
  if (!dir) return
  isScanning.value = true
  try {
    const res = await NasAPI.scan(dir, refresh)
    if (res.code === 200) {
      songs.value = res.data.songs || []
      localStorage.setItem('fn_nas_dir', dir)
    }
  } catch (err) {
    console.error("Scan error:", err)
  } finally {
    isScanning.value = false
  }
}

function switchDir(dir) {
  currentDir.value = dir
  scanFolder(dir, false)
}

function openFolderModal() {
  showFolderModal.value = true
  modalCurrentPath.value = currentDir.value || '/vol1'
  loadBrowse(modalCurrentPath.value)
}

async function loadBrowse(targetPath) {
  modalLoading.value = true
  try {
    const res = await NasAPI.browse(targetPath)
    if (res.code === 200 && res.data) {
      modalCurrentPath.value = res.data.current
      modalParentPath.value = res.data.parent || ''
      if (res.data.volumes?.length) {
        modalVolumes.value = res.data.volumes
      }
      modalFolders.value = res.data.folders || []
      modalAudioCount.value = res.data.audio_count || 0
    }
  } catch (err) {
    console.warn("Browse error:", err)
  } finally {
    modalLoading.value = false
  }
}

function modalDrillDown(path) {
  loadBrowse(path)
}

function confirmSelection(selectedPath) {
  showFolderModal.value = false
  switchDir(selectedPath)
}

function mapNasSong(song) {
  return {
    id: song.id,
    path: song.path,
    name: song.title || song.filename,
    title: song.title || song.filename,
    singer: song.artist || '本地音乐',
    artist: song.artist || '本地音乐',
    album: song.album || '飞牛 NAS 本地音乐',
    cover: song.cover_url || '',
    streamUrl: NasAPI.streamUrl(song.path),
    source: 'nas',
    format: song.format,
  }
}

function playSong(song, idx) {
  const allFormatted = songs.value.map(mapNasSong)
  const currentFormatted = mapNasSong(song)
  emit('play', {
    song: currentFormatted,
    playlist: allFormatted,
    index: idx !== undefined ? idx : allFormatted.findIndex(s => s.id === song.id)
  })
}

function playAllNasSongs() {
  if (!songs.value.length) return
  const allFormatted = songs.value.map(mapNasSong)
  emit('play', {
    song: allFormatted[0],
    playlist: allFormatted,
    index: 0
  })
}

onMounted(async () => {
  await loadDirectories()
  await scanFolder(currentDir.value, false)
})
</script>
