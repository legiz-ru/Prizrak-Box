<template>
  <el-dialog
      v-model="dialogVisible"
      :title="$t('age.keypair.title')"
      width="480"
      draggable
      append-to-body
      @closed="onClosed"
  >
    <div class="age-keypair-body">
      <div class="algo-section">
        <span class="algo-label">{{ $t('age.keypair.algorithm') }}</span>
        <el-radio-group v-model="selectedType" @change="regenerate">
          <el-radio value="mlkem768-x25519">MLKEM768-X25519</el-radio>
          <el-radio value="x25519">X25519</el-radio>
        </el-radio-group>
      </div>

      <div v-if="loading" class="loading-hint">{{ $t('age.keypair.generating') }}</div>
      <div v-else-if="error" class="error-hint">{{ error }}</div>
      <template v-else-if="keypair">
        <div class="key-row">
          <div class="key-label">{{ $t('age.keypair.publicKey') }}</div>
          <div class="key-area">
            <span class="key-value">{{ keypair.publicKey }}</span>
            <button class="pill-btn" @click="copyKey(keypair.publicKey, 'pub')">
              {{ copiedPub ? $t('age.keypair.copied') : $t('age.keypair.copy') }}
            </button>
          </div>
        </div>
        <div class="key-row">
          <div class="key-label">{{ $t('age.keypair.secretKey') }}</div>
          <div class="key-area">
            <span class="key-value">{{ keypair.secretKey }}</span>
            <button class="pill-btn" @click="copyKey(keypair.secretKey, 'sec')">
              {{ copiedSec ? $t('age.keypair.copied') : $t('age.keypair.copy') }}
            </button>
          </div>
        </div>
      </template>
    </div>

    <template #footer>
      <div class="age-keypair-footer">
        <button class="pill-btn" :disabled="loading" @click="regenerate">
          {{ keypair ? $t('age.keypair.regenerate') : $t('age.keypair.generate') }}
        </button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import {useI18n} from "vue-i18n";
import createApi from "@/api";

const props = defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>();

const {t} = useI18n();
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
});

const selectedType = ref<'mlkem768-x25519' | 'x25519'>('mlkem768-x25519');
const keypair = ref<{ secretKey: string; publicKey: string } | null>(null);
const loading = ref(false);
const error = ref('');
const copiedPub = ref(false);
const copiedSec = ref(false);

watch(() => props.modelValue, (visible) => {
  if (visible && !keypair.value) {
    regenerate();
  }
});

async function regenerate() {
  loading.value = true;
  error.value = '';
  copiedPub.value = false;
  copiedSec.value = false;
  try {
    const res = await (proxy as any).$http.get(`/age/keypair?type=${selectedType.value}`);
    keypair.value = res;
  } catch (e: any) {
    error.value = e?.message || String(e);
  } finally {
    loading.value = false;
  }
}

function copyKey(text: string, which: 'pub' | 'sec') {
  navigator.clipboard.writeText(text).catch(() => {
    const el = document.createElement('textarea');
    el.value = text;
    document.body.appendChild(el);
    el.select();
    document.execCommand('copy');
    document.body.removeChild(el);
  });
  if (which === 'pub') {
    copiedPub.value = true;
    setTimeout(() => { copiedPub.value = false; }, 2000);
  } else {
    copiedSec.value = true;
    setTimeout(() => { copiedSec.value = false; }, 2000);
  }
}

function onClosed() {
  keypair.value = null;
  error.value = '';
}
</script>

<style scoped>
.age-keypair-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.algo-section {
  display: flex;
  align-items: center;
  gap: 12px;
}

.algo-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.key-row {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.key-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.key-area {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  padding: 8px 12px;
  min-width: 0;
}

.key-value {
  flex: 1;
  font-family: monospace;
  font-size: 12px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.loading-hint,
.error-hint {
  font-size: 13px;
  text-align: center;
  padding: 12px 0;
}

.error-hint {
  color: var(--el-color-danger);
}

.age-keypair-footer {
  display: flex;
  justify-content: center;
}

.pill-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 28px;
  padding: 0 14px;
  border: 1.5px solid var(--el-border-color);
  border-radius: 999px;
  cursor: pointer;
  font-size: 12px;
  background: transparent;
  color: var(--el-text-color-primary);
  user-select: none;
  transition: border-color 0.2s, color 0.2s;
  white-space: nowrap;
  flex-shrink: 0;
}

.pill-btn:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

.pill-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
