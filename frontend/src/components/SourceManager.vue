<template>
  <div class="h-full flex flex-col overflow-hidden select-none">
    <div class="px-4 md:px-6 py-4 bg-white border-b border-gray-100 flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
      <div>
        <h2 class="text-sm font-bold text-gray-800 flex items-center gap-2">
          音源引擎管理
          <span class="text-[10px] px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-600 border border-emerald-200 font-mono">已导入 {{ sources.length }} 个</span>
        </h2>
        <p class="text-xs text-gray-400 mt-0.5">支持导入自定义第三方音源脚本规则，轻松解锁全网高品质音频</p>
      </div>
      <div class="flex gap-2 self-end sm:self-auto">
        <button @click="testActiveResolution" :disabled="isTesting || !activeSourceId"
          class="px-3 py-1.5 rounded-lg bg-gray-100 hover:bg-gray-200 text-xs text-gray-600 flex items-center gap-1.5 transition-all disabled:opacity-50"
          title="测试当前激活音源是否能成功解析歌曲">
          <PlayCircle :class="['w-3.5 h-3.5 text-blue-500', isTesting ? 'animate-spin' : '']" />
          {{ isTesting ? '测试中...' : '测试解析' }}
        </button>
        <button @click="pingAllSources" :disabled="isPinging || sources.length === 0"
          class="px-3 py-1.5 rounded-lg bg-gray-100 hover:bg-gray-200 text-xs text-gray-600 flex items-center gap-1.5 transition-all disabled:opacity-50">
          <Activity :class="['w-3.5 h-3.5 text-emerald-500', isPinging ? 'animate-spin' : '']" />
          {{ isPinging ? '测速中...' : '全部测速' }}
        </button>
        <button @click="showImportModal = true"
          class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-semibold flex items-center gap-1.5 shadow-sm">
          <Plus class="w-3.5 h-3.5" />导入音源
        </button>
      </div>
    </div>

    <div class="flex-1 overflow-y-auto bg-gray-50 p-4 md:p-5">
      <!-- 空状态提示 (纯播放器，0 内置音源) -->
      <div v-if="sources.length === 0" class="flex flex-col items-center justify-center py-16 text-center bg-white rounded-2xl border border-gray-100 p-6 shadow-2xs">
        <div class="w-14 h-14 rounded-2xl bg-gray-100 flex items-center justify-center text-2xl mb-3 text-gray-400">
          📦
        </div>
        <h3 class="text-base font-semibold text-gray-700 mb-1">暂无已导入音源</h3>
        <p class="text-xs text-gray-400 max-w-sm mb-5 leading-relaxed px-4">
          本软件为本地与流媒体播放器，不内置任何音源。请点击下方按钮手动导入您自己的第三方音源脚本规则（支持网络 URL 或本地 .js 文件）。
        </p>
        <button @click="showImportModal = true"
          class="px-4 py-2 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-semibold flex items-center gap-1.5 shadow-sm transition-all">
          <Plus class="w-4 h-4" />导入第三方音源
        </button>
      </div>

      <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3.5">
        <div v-for="s in sources" :key="s.id"
          :class="[
            'bg-white rounded-xl border p-4 flex items-center gap-3 transition-all cursor-pointer hover:border-emerald-300 hover:shadow-sm',
            activeSourceId === s.id ? 'border-emerald-400 shadow-sm ring-1 ring-emerald-400/20' : 'border-gray-200'
          ]"
          @click="activateSource(s.id)"
        >
          <div class="w-10 h-10 rounded-xl bg-gray-50 border border-gray-100 flex items-center justify-center text-xl shrink-0">{{ getSourceIcon(s.id) }}</div>
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-1.5 flex-wrap">
              <span class="font-semibold text-sm text-gray-800">{{ s.name }}</span>
              <span v-if="s.version" class="text-[10px] px-1.5 py-0.5 rounded bg-gray-100 text-gray-400 font-mono">v{{ s.version }}</span>
              <span class="text-[10px] px-1.5 py-0.5 rounded bg-amber-50 text-amber-700 border border-amber-200/80 font-medium">第三方</span>
              <span v-if="activeSourceId === s.id" class="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500 text-white font-bold flex items-center gap-0.5">
                <Check class="w-2.5 h-2.5" />生效中
              </span>
            </div>
            <div class="flex items-center gap-2 mt-1 text-xs text-gray-400">
              <span class="truncate max-w-[120px]">{{ s.author || '社区开发者' }}</span>
              <span class="text-gray-200">|</span>
              <span v-for="p in ['wy','tx','kg','kw']" :key="p" class="px-1.5 py-0.5 rounded bg-gray-100 text-gray-500 font-mono text-[10px]">{{ p }}</span>
            </div>
          </div>
          <div class="flex flex-col items-end gap-1.5 shrink-0" @click.stop>
            <div class="flex items-center gap-1">
              <span :class="['w-2 h-2 rounded-full', getLatencyDot(s.id)]"></span>
              <span class="text-xs text-gray-400 font-mono">{{ latencies[s.id] || '待测速' }}</span>
            </div>
            <div class="flex items-center gap-1">
              <button @click="deleteSource(s.id)" class="p-1 rounded-lg hover:bg-red-50 text-gray-300 hover:text-red-400 transition-colors" title="删除">
                <Trash2 class="w-3.5 h-3.5" />
              </button>
              <button @click="activateSource(s.id)" :disabled="activeSourceId === s.id"
                :class="['px-2.5 py-1 rounded-lg text-xs font-semibold transition-all', activeSourceId === s.id ? 'bg-emerald-50 text-emerald-600 cursor-default' : 'bg-gray-100 hover:bg-emerald-500 hover:text-white text-gray-600']"
              >{{ activeSourceId === s.id ? '当前激活' : '启用' }}</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showImportModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/30 backdrop-blur-sm">
      <div class="bg-white rounded-2xl max-w-lg w-full p-6 shadow-xl border border-gray-100">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-base font-bold text-gray-800">导入第三方自定义音源</h3>
          <button @click="showImportModal = false" class="text-gray-400 hover:text-gray-600"><X class="w-5 h-5" /></button>
        </div>
        <p class="text-xs text-gray-500 mb-3">输入在线音源 raw 链接（GitHub/Gitee/私服），或上传本地 .js 文件</p>
        <div class="p-3 bg-amber-50 rounded-xl border border-amber-200/80 mb-3 text-[11px] text-amber-800 leading-relaxed">
          💡 <strong>说明：</strong> 本播放器为本地容器，不内置任何网络音源。您可导入第三方开发者维护的 LX Music 音源脚本直链（支持 kw、tx、kg、wy 等平台音频解析），或直接上传本地 .js 脚本文件。
        </div>
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">脚本网络 URL</label>
            <input v-model="importUrl" type="text" placeholder="https://raw.githubusercontent.com/.../latest.js"
              class="w-full px-3 py-2 rounded-lg bg-gray-50 border border-gray-200 text-gray-800 text-xs placeholder:text-gray-400 focus:outline-none focus:border-emerald-400" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">音源别名（可选）</label>
            <input v-model="importName" type="text" placeholder="例如：私人无损高品质音源"
              class="w-full px-3 py-2 rounded-lg bg-gray-50 border border-gray-200 text-gray-800 text-xs placeholder:text-gray-400 focus:outline-none focus:border-emerald-400" />
          </div>
          <div>
            <p class="text-xs text-gray-400 mb-1">或上传本地文件：</p>
            <input type="file" accept=".js" @change="handleFileUpload"
              class="block w-full text-xs text-gray-500 file:mr-3 file:py-1.5 file:px-3 file:rounded-lg file:border-0 file:text-xs file:font-semibold file:bg-emerald-50 file:text-emerald-700 hover:file:bg-emerald-100 cursor-pointer" />
          </div>
        </div>
        <div class="mt-5 flex justify-end gap-2">
          <button @click="showImportModal = false" class="px-4 py-2 rounded-lg text-xs text-gray-500 hover:bg-gray-50">取消</button>
          <button @click="submitUrlImport" :disabled="isImporting"
            class="px-5 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white font-semibold text-xs shadow-sm disabled:opacity-50">
            {{ isImporting ? '导入中...' : '立即导入' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { SourcesAPI } from '../api/client'
import { lxRuntime } from '../engine/lx-runtime'
import { Activity, Plus, Check, Trash2, X, PlayCircle } from 'lucide-vue-next'

const sources = ref([])
const activeSourceId = ref('')
const latencies = ref({})
const isPinging = ref(false)
const isTesting = ref(false)
const showImportModal = ref(false)
const importUrl = ref('')
const importName = ref('')
const isImporting = ref(false)
const emit = defineEmits(['source-changed'])

function platformLabel(p) { return p }
function getSourceIcon(id) { return { sixyin: '⚡', flower: '🌷', huibq: '💎', lx: '❄️', ikun: '🏀', grass: '🌿', juhe: '🔥', svip: '👑' }[id] || '🎧' }
function getLatencyDot(id) {
  const lat = latencies.value[id]
  if (!lat || lat === '待测速') return 'bg-gray-300'
  if (lat.includes('超时') || lat.includes('异常')) return 'bg-red-400'
  const ms = parseInt(lat)
  if (ms < 300) return 'bg-emerald-400'
  if (ms < 800) return 'bg-amber-400'
  return 'bg-orange-400'
}

async function loadSources() {
  try {
    const res = await SourcesAPI.list()
    if (res.code === 200) {
      sources.value = res.data.sources || []
      activeSourceId.value = res.data.active_id || (sources.value.length > 0 ? sources.value[0].id : '')
      if (activeSourceId.value) await loadScriptIntoRuntime(activeSourceId.value)
    }
  } catch (err) { console.error('Failed to load sources:', err) }
}

async function loadScriptIntoRuntime(sourceId) {
  try {
    const meta = sources.value.find(s => s.id === sourceId)
    if (!meta) return
    const res = await SourcesAPI.getScript(sourceId)
    if (res.code === 200) { await lxRuntime.loadScript(meta, res.data.script); emit('source-changed', meta) }
  } catch (e) { console.warn('Failed to load script:', e) }
}

async function activateSource(id) {
  if (activeSourceId.value === id) return
  try {
    const res = await SourcesAPI.setActive(id)
    if (res.code === 200) { activeSourceId.value = id; await loadScriptIntoRuntime(id) }
  } catch (err) { alert('切换音源失败: ' + err.message) }
}

async function pingAllSources() {
  isPinging.value = true
  for (const s of sources.value) {
    const start = performance.now()
    try {
      const res = await SourcesAPI.getScript(s.id)
      latencies.value[s.id] = res.code === 200 ? Math.round(performance.now() - start) + 'ms' : '异常'
    } catch { latencies.value[s.id] = '超时' }
  }
  isPinging.value = false
}

async function testActiveResolution() {
  if (!activeSourceId.value) {
    alert('请先选择并启用一个音源')
    return
  }
  isTesting.value = true
  const meta = sources.value.find(s => s.id === activeSourceId.value)
  const name = meta?.name || '当前音源'
  try {
    const res = await lxRuntime.getMusicUrl(
      activeSourceId.value,
      'kw',
      { id: '624683929', songmid: '624683929', name: '山风山风等等我', singer: '万海东' },
      '128k'
    )
    if (res?.url && /^https?:/.test(res.url)) {
      alert(`【音源解析测试成功 ✅】\n音源：${name}\n平台：KW\n状态：音频直链解析正常，可以正常播放！`)
    } else {
      alert(`【音源解析未返回有效直链 ⚠️】\n音源：${name}\n可能该音源上游接口暂时不可用或规则已失效，建议前往导入最新有效音源。`)
    }
  } catch (e) {
    alert(`【音源解析测试失败 ❌】\n音源：${name}\n错误：${e.message}\n建议前往导入或切换其他有效音源规则。`)
  } finally {
    isTesting.value = false
  }
}

async function submitUrlImport() {
  if (!importUrl.value) return
  isImporting.value = true
  try {
    const res = await SourcesAPI.importUrl(importUrl.value, importName.value)
    if (res.code === 200) { showImportModal.value = false; importUrl.value = ''; importName.value = ''; await loadSources(); alert('音源导入成功！') }
    else alert('导入失败: ' + res.message)
  } catch (err) { alert('导入异常: ' + err.message) }
  finally { isImporting.value = false }
}

async function handleFileUpload(e) {
  const file = e.target.files?.[0]; if (!file) return
  const reader = new FileReader()
  reader.onload = async (ev) => {
    try {
      const res = await SourcesAPI.upload({ filename: file.name, script: ev.target.result })
      if (res.code === 200) { showImportModal.value = false; await loadSources(); alert('音源上传成功！') }
      else alert('上传失败: ' + res.message)
    } catch (err) { alert('上传出错: ' + err.message) }
  }
  reader.readAsText(file)
}

async function deleteSource(id) {
  if (!confirm('确定要删除此音源吗？')) return
  try {
    const res = await SourcesAPI.delete(id)
    if (res.code === 200) await loadSources()
  } catch (err) { alert('删除失败: ' + err.message) }
}

onMounted(() => { loadSources() })
</script>
