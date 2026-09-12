<template>
  <div class="relative flex flex-col items-center justify-center select-none py-6">
    <!-- Outer Glow Ambient -->
    <div 
      :class="[
        'absolute w-72 h-72 rounded-full filter blur-3xl transition-opacity duration-1000 -z-10',
        isPlaying ? 'opacity-40 bg-gradient-to-tr from-aurora-cyan via-aurora-violet to-aurora-magenta' : 'opacity-10 bg-slate-700'
      ]"
    ></div>

    <!-- Vinyl Disc & Cover Container -->
    <div class="relative w-64 h-64 sm:w-80 sm:h-80 flex items-center justify-center">
      <!-- Vinyl Base Disc with Grooves -->
      <div 
        :class="[
          'relative w-full h-full rounded-full shadow-2xl transition-transform duration-1000',
          isPlaying ? 'animate-spin-slow' : ''
        ]"
        style="background: radial-gradient(circle, #0e111a 0%, #151928 20%, #0a0c14 45%, #181d2e 65%, #08090f 85%, #05060a 100%); box-shadow: 0 10px 40px rgba(0,0,0,0.8), inset 0 0 15px rgba(255,255,255,0.06);"
      >
        <!-- Concentric Vinyl Texture Grooves -->
        <div class="absolute inset-4 rounded-full border border-white/5 pointer-events-none"></div>
        <div class="absolute inset-8 rounded-full border border-white/5 pointer-events-none"></div>
        <div class="absolute inset-12 rounded-full border border-white/5 pointer-events-none"></div>
        <div class="absolute inset-16 rounded-full border border-white/5 pointer-events-none"></div>
        <div class="absolute inset-20 rounded-full border border-white/5 pointer-events-none"></div>

        <!-- Center Label & Album Artwork -->
        <div class="absolute inset-0 m-auto w-28 h-28 sm:w-36 sm:h-36 rounded-full overflow-hidden border-4 border-dark-900 shadow-inner flex items-center justify-center bg-dark-800">
          <img 
            v-if="coverUrl" 
            :src="coverUrl" 
            alt="Song Cover" 
            class="w-full h-full object-cover"
            @error="onCoverError"
          />
          <div v-else class="w-full h-full flex items-center justify-center bg-gradient-to-tr from-aurora-cyan/30 to-aurora-magenta/30">
            <span class="text-3xl">🎵</span>
          </div>

          <!-- Center Spindle Hole -->
          <div class="absolute w-6 h-6 rounded-full bg-dark-950 border-2 border-white/20 shadow-md"></div>
        </div>
      </div>

      <!-- Realistic Tonearm (唱针) -->
      <div 
        :class="[
          'absolute -top-4 right-2 sm:right-6 w-20 h-36 pointer-events-none tonearm transition-transform duration-700 z-20',
          isPlaying ? 'tonearm-playing' : 'tonearm-paused'
        ]"
      >
        <!-- Tonearm Pivot Base -->
        <div class="absolute top-2 left-3 w-8 h-8 rounded-full bg-gradient-to-br from-slate-400 to-slate-700 border border-white/20 shadow-lg flex items-center justify-center">
          <div class="w-3 h-3 rounded-full bg-dark-950"></div>
        </div>
        <!-- Metallic Arm Pole -->
        <div class="absolute top-6 left-6 w-1.5 h-24 bg-gradient-to-r from-slate-300 via-slate-400 to-slate-500 rounded-full shadow-md origin-top transform rotate-12"></div>
        <!-- Cartridge & Stylus Head -->
        <div class="absolute bottom-5 left-10 w-4 h-7 bg-gradient-to-br from-aurora-cyan to-aurora-violet rounded shadow-md transform rotate-20 border border-white/20">
          <div class="absolute bottom-0 left-1.5 w-1 h-2 bg-slate-200"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  isPlaying: Boolean,
  cover: String,
})

const coverUrl = ref(props.cover || '')

watch(() => props.cover, (val) => {
  coverUrl.value = val || ''
})

function onCoverError() {
  coverUrl.value = ''
}
</script>
