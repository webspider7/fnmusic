<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="modelValue"
        class="fixed inset-0 z-60 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4"
        @click="onBackdropClick"
      >
        <div
          @click.stop
          class="w-full max-w-lg bg-white rounded-2xl shadow-2xl border border-gray-100 overflow-hidden flex flex-col max-h-[90vh] animate-in fade-in zoom-in-95 duration-200"
        >
          <!-- 标题栏 -->
          <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between bg-gradient-to-r from-amber-50/60 to-orange-50/60">
            <div class="flex items-center gap-2.5">
              <div class="w-8 h-8 rounded-xl bg-amber-500 text-white flex items-center justify-center shadow-xs">
                <ShieldAlert class="w-4 h-4" />
              </div>
              <div>
                <h3 class="text-sm font-bold text-gray-800">
                  免责声明与服务条款
                </h3>
                <p class="text-[11px] text-gray-500">
                  极光音乐 · 纯净开源容器合规声明
                </p>
              </div>
            </div>

            <!-- 如果非强制首次确认，允许直接右上角关闭 -->
            <button
              v-if="!isForce"
              @click="close"
              class="p-1.5 rounded-xl text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
            >
              <X class="w-4 h-4" />
            </button>
          </div>

          <!-- 正文条款区域 -->
          <div class="p-5 overflow-y-auto space-y-3.5 text-xs leading-relaxed text-gray-600">
            <div class="p-3 rounded-xl bg-amber-50/80 border border-amber-200/80 text-amber-900 text-[11px] flex items-start gap-2">
              <AlertTriangle class="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
              <div>
                <strong>重要提醒：</strong>本播放器为纯本地播放器技术容器，不内置、不提供、不分发任何在线音频及侵权内容。使用本软件前，请务必仔细阅读并理解以下声明条款。
              </div>
            </div>

            <div class="space-y-3">
              <div>
                <h4 class="font-bold text-gray-800 text-xs flex items-center gap-1.5 mb-1">
                  <span>1. 技术中立与容器定位</span>
                </h4>
                <p class="text-[11px] text-gray-500 text-justify">
                  本软件为运行于私有私有云（如飞牛 NAS）环境下的个人本地音频播放与文件管理容器。软件仅具备音频解码串流播放、本地存储卷文件浏览、元数据展示等基础播放器功能，源码与安装包中绝无任何版权音频资源的存储、分发或聚合盗链。
                </p>
              </div>

              <div>
                <h4 class="font-bold text-gray-800 text-xs flex items-center gap-1.5 mb-1">
                  <span>2. 第三方自定义音源脚本责任</span>
                </h4>
                <p class="text-[11px] text-gray-500 text-justify">
                  软件内置的音源引擎仅为通用的客户端 JavaScript 脚本执行沙箱环境。软件未内置、未预装、亦不推荐任何特定第三方音源规则。所有网络音源规则均由使用者自愿自行寻找、导入并启用。因用户个人导入的第三方规则所产生的任何权利争议，均由导入者和脚本提供方自行承担，与本软件开发者无关。
                </p>
              </div>

              <div>
                <h4 class="font-bold text-gray-800 text-xs flex items-center gap-1.5 mb-1">
                  <span>3. 严格限于个人学习与家庭研究</span>
                </h4>
                <p class="text-[11px] text-gray-500 text-justify">
                  本开源项目完全基于技术交流、NAS 场景容器化适配研究及个人家庭无损音乐管理目的开发，仅供非商业用途学习体验。严禁将本软件或衍生版本用于任何形式的商业营利行为或违法活动。
                </p>
              </div>

              <div>
                <h4 class="font-bold text-gray-800 text-xs flex items-center gap-1.5 mb-1">
                  <span>4. 知识产权保护与下架机制</span>
                </h4>
                <p class="text-[11px] text-gray-500 text-justify">
                  本软件高度尊重各大内容平台及版权方的知识产权。若任何权利方认为本开源技术项目的代码实现涉及侵权争议，请通过 GitHub 仓库提交合法有效的权利通知，开发者核实后将在第一时间积极响应并配合处理。
                </p>
              </div>

              <div>
                <h4 class="font-bold text-gray-800 text-xs flex items-center gap-1.5 mb-1">
                  <span>5. 免责范围</span>
                </h4>
                <p class="text-[11px] text-gray-500 text-justify">
                  在法律允许的最大范围内，因使用或无法使用本软件（包括但不限于第三方脚本失效、网络不稳定、数据异常等）所造成的任何直接或间接后果，作者及开源贡献者均不承担任何连带法律责任。
                </p>
              </div>
            </div>
          </div>

          <!-- 底部确认区域 -->
          <div class="px-5 py-3.5 bg-gray-50 border-t border-gray-100 flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
            <label v-if="isForce" class="flex items-center gap-2 cursor-pointer select-none text-[11px] text-gray-700">
              <input
                type="checkbox"
                v-model="hasAgreed"
                class="w-4 h-4 text-emerald-600 rounded border-gray-300 focus:ring-emerald-500"
              />
              <span>我已充分阅读并同意以上免责声明</span>
            </label>
            <span v-else class="text-[11px] text-gray-400 font-mono">
              协议已生效
            </span>

            <div class="flex items-center gap-2 justify-end">
              <button
                v-if="isForce"
                @click="onDecline"
                class="px-3 py-1.5 rounded-xl border border-gray-200 hover:bg-white text-gray-500 text-xs transition-all"
              >
                暂不同意
              </button>
              <button
                @click="onAccept"
                :disabled="isForce && !hasAgreed"
                class="px-5 py-1.5 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-semibold shadow-sm transition-all disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {{ isForce ? '同意并继续使用' : '我知道了' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref } from 'vue'
import { ShieldAlert, AlertTriangle, X } from 'lucide-vue-next'

const props = defineProps({
  modelValue: Boolean,
  isForce: {
    type: Boolean,
    default: false,
  }
})

const emit = defineEmits(['update:modelValue', 'accepted'])

const hasAgreed = ref(false)

function close() {
  emit('update:modelValue', false)
}

function onBackdropClick() {
  if (!props.isForce) {
    close()
  }
}

function onAccept() {
  localStorage.setItem('fn_disclaimer_accepted', 'true')
  emit('accepted')
  close()
}

function onDecline() {
  alert('您需要同意本免责声明与服务条款后方可正常使用极光音乐各项功能。')
}
</script>
