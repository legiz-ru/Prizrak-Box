<script setup lang="ts">
import {useI18n} from "vue-i18n";
import createApi from "@/api";
import IconKey from "~icons/tabler/key";
import {toast} from "@/components/ui";

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
  if (!visible) {
    onClosed();
  }
});

const algoOptions = [
  {value: 'mlkem768-x25519' as const, label: 'MLKEM768-X25519'},
  {value: 'x25519' as const, label: 'X25519'},
];

watch(selectedType, () => regenerate());

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
  toast('success', t('copy.success'));
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

<template>
  <UiModal v-model="dialogVisible" :title="t('age.keypair.title')" :icon="IconKey" :width="480">
    <div class="algo">
      <span class="algo__label">{{ t('age.keypair.algorithm') }}</span>
      <UiPillTabs v-model="selectedType" :options="algoOptions" :aria-label="t('age.keypair.algorithm')"/>
    </div>

    <div v-if="loading" class="age-loading" role="status">
      <span class="dots" aria-hidden="true"><span></span><span></span><span></span></span>
      {{ t('age.keypair.generating') }}
    </div>
    <div v-else-if="error" class="px-alert px-alert--error">
      <icon-tabler-alert-circle width="15" height="15"/>
      <span>{{ error }}</span>
    </div>
    <template v-else-if="keypair">
      <div class="key">
        <span class="key__label">{{ t('age.keypair.publicKey') }}</span>
        <div class="key__area">
          <span class="key__value mono ellipsis" v-tip="keypair.publicKey">{{ keypair.publicKey }}</span>
          <button type="button" class="key__copy" :class="{ 'is-done': copiedPub }" @click="copyKey(keypair.publicKey, 'pub')">
            {{ copiedPub ? t('age.keypair.copied') : t('age.keypair.copy') }}
          </button>
        </div>
      </div>
      <div class="key">
        <span class="key__label">{{ t('age.keypair.secretKey') }}</span>
        <div class="key__area">
          <span class="key__value mono ellipsis">{{ keypair.secretKey }}</span>
          <button type="button" class="key__copy" :class="{ 'is-done': copiedSec }" @click="copyKey(keypair.secretKey, 'sec')">
            {{ copiedSec ? t('age.keypair.copied') : t('age.keypair.copy') }}
          </button>
        </div>
      </div>
    </template>

    <template #footer>
      <button type="button" class="px-btn px-btn--pill regen" :disabled="loading" @click="regenerate">
        <UiSpinner v-if="loading" :size="12"/>
        <icon-tabler-refresh v-else width="14" height="14"/>
        {{ t('age.keypair.regenerate') }}
      </button>
    </template>
  </UiModal>
</template>

<style scoped>
.algo {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.algo__label {
  font-size: 13px;
  font-weight: 600;
}

.age-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 28px 0;
  font-size: 13px;
  color: var(--text-2);
}

.dots {
  display: flex;
  gap: 3px;
}

.dots span {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--accent);
  animation: px-dot-pulse 1s ease-in-out infinite;
}

.dots span:nth-child(2) { animation-delay: .15s; }
.dots span:nth-child(3) { animation-delay: .3s; }

.key {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
}

.key__label {
  font-size: 11px;
  font-weight: 700;
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: .05em;
}

.key__area {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--input-bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 8px 8px 8px 12px;
  min-width: 0;
}

.key__value {
  flex: 1;
  font-size: 12px;
  user-select: all;
}

.key__copy {
  flex-shrink: 0;
  padding: 4px 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  border: 1.5px solid var(--border);
  background: transparent;
  color: var(--text);
}

.key__copy.is-done {
  border-color: var(--success);
  color: var(--success);
}

.regen {
  margin: 0 auto;
}
</style>
